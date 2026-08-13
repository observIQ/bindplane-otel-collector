// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows

package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/kballard/go-shellquote"
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
)

const (
	defaultProductName = "observIQ Distro for OpenTelemetry Collector"
	defaultServiceName = "observiq-otel-collector"

	// defaultRecoveryDelay is the duration in which to wait between service restarts due to failures
	defaultRecoveryDelay = 5 * time.Second
	// defaultResetPeriod is the time in which to reset the service failure count to zero
	defaultResetPeriod = 30 * 24 * time.Hour
)

// Option is an extra option for creating a Service
type Option func(winSvc *windowsService)

// WithServiceFile returns an option setting the service file to use when updating using the service
func WithServiceFile(svcFilePath string) Option {
	return func(winSvc *windowsService) {
		winSvc.newServiceFilePath = svcFilePath
	}
}

// NewService returns an instance of the Service interface for managing the observiq-otel-collector service on the current OS.
func NewService(logger *zap.Logger, installDir string, opts ...Option) Service {
	winSvc := &windowsService{
		newServiceFilePath: filepath.Join(path.ServiceFileDir(installDir), "windows_service.json"),
		serviceName:        defaultServiceName,
		productName:        defaultProductName,
		installDir:         installDir,
		logger:             logger.Named("windows-service"),
	}

	for _, opt := range opts {
		opt(winSvc)
	}

	return winSvc
}

type windowsService struct {
	// newServiceFilePath is the file path to the new unit file
	newServiceFilePath string
	// serviceName is the name of the service
	serviceName string
	// productName is the name of the installed product
	productName string
	installDir  string
	logger      *zap.Logger
}

// Start the service
func (w windowsService) Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// Stop the service
func (w windowsService) Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	if _, err := s.Control(svc.Stop); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	if err := waitForServiceStatus(s, svc.Stopped); err != nil {
		return fmt.Errorf("failed waiting for service to stop: %w", err)
	}

	return nil
}

func (w windowsService) Update() error {
	// parse the service definition from disk
	wsc, err := readWindowsServiceConfig(w.newServiceFilePath)
	if err != nil {
		return fmt.Errorf("failed to read service config: %w", err)
	}

	// expand the arguments to be properly formatted (expand [INSTALLDIR], clean '&quot;' to be '"')
	expandArguments(wsc, w.installDir)

	// Get the start type
	startType, delayed, err := winapiStartType(wsc.Service.Start)
	if err != nil {
		return fmt.Errorf("failed to parse start type in service config: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Get the installed service handle
	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			w.logger.Error("failed to close service after update", zap.Error(err))
		}
	}()

	// Get the current config; We will use the current config as the basis
	// for the new config.
	newConf, err := s.Config()
	if err != nil {
		return fmt.Errorf("failed to get current service configuration: %w", err)
	}

	// Get the full path the the collector
	fullCollectorPath := filepath.Join(w.installDir, wsc.Path)
	// binary path is the path to the EXE, then the space separated list of arguments.
	// we quote the collector path, in case it contains spaces.
	binaryPathName := fmt.Sprintf("\"%s\" %s", fullCollectorPath, wsc.Service.Arguments)

	// Fill in the new config values
	newConf.BinaryPathName = binaryPathName
	newConf.Description = wsc.Service.Description
	newConf.DisplayName = wsc.Service.DisplayName
	newConf.StartType = startType
	newConf.DelayedAutoStart = delayed

	// Update the service in-place
	err = s.UpdateConfig(newConf)
	if err != nil {
		return fmt.Errorf("failed to updater service: %w", err)
	}

	// Set the recovery actions
	if err := setRecoveryActions(s); err != nil {
		return fmt.Errorf("failed to set recovery actions for the service: %w", err)
	}

	return nil
}

func (w windowsService) Backup() error {

	wsc, err := w.currentServiceConfig()
	if err != nil {
		return fmt.Errorf("failed to construct service config: %w", err)
	}

	// Marshal config as json
	wscBytes, err := json.Marshal(wsc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Open with O_EXCL to fail if the file already exists
	f, err := os.OpenFile(path.BackupServiceFile(w.installDir), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to create backup service file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			w.logger.Error("Failed to close backup service file", zap.Error(err))
		}
	}()

	// finally, write the config out so we can rollback.
	if _, err := f.Write(wscBytes); err != nil {
		return fmt.Errorf("failed to write backup service config: %w", err)
	}

	return nil
}

// windowsServiceConfig defines how the service should be configured, including the entrypoint for the service.
type windowsServiceConfig struct {
	// Path is the file that will be executed for the service. It is relative to the install directory.
	Path string `json:"path"`
	// Configuration for the service (e.g. start type, display name, desc)
	Service windowsServiceDefinitionConfig `json:"service"`
}

// windowsServiceDefinitionConfig defines how the service should be configured.
// Name is a part of the on disk config, but we keep the service name hardcoded; We do not want to use a different service name.
type windowsServiceDefinitionConfig struct {
	// Start gives the start type of the service.
	// See: https://wixtoolset.org/documentation/manual/v3/xsd/wix/serviceinstall.html
	Start string `json:"start"`
	// DisplayName is the human-readable name of the service.
	DisplayName string `json:"display-name"`
	// Description is a human-readable description of the service.
	Description string `json:"description"`
	// Arguments is a list of space-separated
	Arguments string `json:"arguments"`
}

// readWindowsServiceConfig reads the service config from the file at the given path
func readWindowsServiceConfig(path string) (*windowsServiceConfig, error) {
	cleanPath := filepath.Clean(path)
	b, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var wsc windowsServiceConfig
	err = json.Unmarshal(b, &wsc)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return &wsc, nil
}

// expandArguments expands [INSTALLDIR] to the actual install directory and
// expands '&quot;' to the literal '"'
func expandArguments(wsc *windowsServiceConfig, installDir string) {
	wsc.Service.Arguments = string(replaceInstallDir([]byte(wsc.Service.Arguments), installDir))
	wsc.Service.Arguments = strings.ReplaceAll(wsc.Service.Arguments, "&quot;", `"`)
}

func (w windowsService) currentServiceConfig() (*windowsServiceConfig, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Get the current config of the service
	conf, err := s.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get service config: %w", err)
	}

	fullBinaryPath, argString, err := splitServiceBinaryName(conf.BinaryPathName)
	if err != nil {
		return nil, fmt.Errorf("failed to split service BinaryPathName: %w", err)
	}

	// In the original config, the Path is the main binary path, relative to the install directory.
	binaryPath, err := filepath.Rel(w.installDir, fullBinaryPath)
	if err != nil {
		return nil, fmt.Errorf("could not find service exe relative to install dir: %w", err)
	}

	// Convert windows api start type to the config file service type
	confStartType, err := configStartType(conf.StartType, conf.DelayedAutoStart)
	if err != nil {
		return nil, fmt.Errorf("failed to get start type: %w", err)
	}

	// Construct the config
	return &windowsServiceConfig{
		Path: binaryPath,
		Service: windowsServiceDefinitionConfig{
			Start:       confStartType,
			DisplayName: conf.DisplayName,
			Description: conf.Description,
			Arguments:   argString,
		},
	}, nil
}

func splitServiceBinaryName(binaryPathName string) (binaryPath, argString string, err error) {
	// Split the service arguments into an array of arguments
	args, err := shellquote.Split(binaryPathName)
	if err != nil {
		return "", "", fmt.Errorf("failed to split service config args: %w", err)
	}

	// The first argument is always the binary name; If the length of the array is 0, we know this is an invalid argument list.
	if len(args) < 1 {
		return "", "", fmt.Errorf("no binary specified in service config")
	}

	// The absolute path to the binary is the first argument
	binaryPath = args[0]

	// Stored argument string doesn't include the binary path (first arg)
	args = args[1:]

	// Args should end up being a string, where literal quotes are "&quot;"
	argString = shellquote.Join(args...)
	// shellquote uses ' to quote, so we convert those to "&quot;"
	argString = strings.ReplaceAll(argString, "'", "&quot;")

	return binaryPath, argString, nil
}

// winapiStartType converts the start type from the windowsServiceConfig to a start type recognizable by the windows
// service API
func winapiStartType(cfgStartType string) (startType uint32, delayed bool, err error) {
	switch cfgStartType {
	case "auto":
		// Automatically starts on system bootup.
		startType = mgr.StartAutomatic
	case "demand":
		// Must be started manually
		startType = mgr.StartManual
	case "disabled":
		// Does not start, must be enabled to run.
		startType = mgr.StartDisabled
	case "delayed":
		// Boots automatically on start, but AFTER bootup has completed.
		startType = mgr.StartAutomatic
		delayed = true
	default:
		err = fmt.Errorf("invalid start type in service config: %s", cfgStartType)
	}
	return
}

func configStartType(winapiStartType uint32, delayed bool) (string, error) {
	switch winapiStartType {
	case mgr.StartAutomatic:
		if delayed {
			return "delayed", nil
		}
		return "auto", nil
	case mgr.StartDisabled:
		return "disabled", nil
	case mgr.StartManual:
		return "manual", nil
	default:
		return "", fmt.Errorf("invalid winapi start type: %d", winapiStartType)
	}
}

func waitForServiceStatus(s *mgr.Service, status svc.State) error {
	for {
		svc, err := s.Query()
		if err != nil {
			return err
		}
		if svc.State == status {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// setRecoveryActions sets the service recovery actions for the passed in service
func setRecoveryActions(s *mgr.Service) error {
	// Windows requires specifying 3 recoveryActions. We want to ensure the service always restarts after 5 seconds
	recoveryActions := []mgr.RecoveryAction{
		{
			Type:  mgr.ServiceRestart,
			Delay: defaultRecoveryDelay,
		},
		{
			Type:  mgr.ServiceRestart,
			Delay: defaultRecoveryDelay,
		},
		{
			Type:  mgr.ServiceRestart,
			Delay: defaultRecoveryDelay,
		},
	}

	return s.SetRecoveryActions(recoveryActions, uint32(defaultResetPeriod.Seconds()))
}
