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

//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/observiq/bindplane-otel-collector/updater/internal/file"
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"go.uber.org/zap"
)

const (
	darwinServiceFilePath = "/Library/LaunchDaemons/com.observiq.collector.plist"
)

// Option is an extra option for creating a Service
type Option func(darwinSvc *darwinService)

// WithServiceFile returns an option setting the service file to use when updating using the service
func WithServiceFile(svcFilePath string) Option {
	return func(darwinSvc *darwinService) {
		darwinSvc.newServiceFilePath = svcFilePath
	}
}

// NewService returns an instance of the Service interface for managing the observiq-otel-collector service on the current OS.
func NewService(logger *zap.Logger, installDir string, opts ...Option) Service {
	darwinSvc := &darwinService{
		newServiceFilePath:       filepath.Join(path.ServiceFileDir(installDir), "com.observiq.collector.plist"),
		installedServiceFilePath: darwinServiceFilePath,
		installDir:               path.DarwinInstallDir,
		logger:                   logger.Named("darwin-service"),
	}

	for _, opt := range opts {
		opt(darwinSvc)
	}

	return darwinSvc
}

type darwinService struct {
	// newServiceFilePath is the file path to the new plist file
	newServiceFilePath string
	// installedServiceFilePath is the file path to the installed plist file
	installedServiceFilePath string
	// installDir is the root directory of the main installation
	installDir string
	logger     *zap.Logger
}

// Start the service
func (d darwinService) Start() error {
	// Launchctl exits with error code 0 if the file does not exist.
	// We want to ensure that we error in this scenario.
	if _, err := os.Stat(d.installedServiceFilePath); err != nil {
		return fmt.Errorf("failed to stat installed service file: %w", err)
	}

	//#nosec G204 -- installedServiceFilePath is not determined by user input
	cmd := exec.Command("launchctl", "load", d.installedServiceFilePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running launchctl failed: %w", err)
	}
	return nil
}

// Stop the service
func (d darwinService) Stop() error {
	// Launchctl exits with error code 0 if the file does not exist.
	// We want to ensure that we error in this scenario.
	if _, err := os.Stat(d.installedServiceFilePath); err != nil {
		return fmt.Errorf("failed to stat installed service file: %w", err)
	}

	//#nosec G204 -- installedServiceFilePath is not determined by user input
	cmd := exec.Command("launchctl", "unload", d.installedServiceFilePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running launchctl failed: %w", err)
	}
	return nil
}

// Installs the service
func (d darwinService) install() error {
	serviceFileBytes, err := os.ReadFile(d.newServiceFilePath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}

	// Ensure the parent directory exists, but only for standard LaunchAgents path
	parentDir := filepath.Dir(d.installedServiceFilePath)
	if filepath.Base(parentDir) == "LaunchAgents" {
		if err := os.MkdirAll(parentDir, 0750); err != nil {
			return fmt.Errorf("failed to create service directory: %w", err)
		}
	}

	expandedServiceFileBytes := replaceInstallDir(serviceFileBytes, d.installDir)
	if err := os.WriteFile(d.installedServiceFilePath, expandedServiceFileBytes, 0600); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	return d.Start()
}

// Uninstalls the service
func (d darwinService) uninstall() error {
	// Check if service file exists first
	if _, err := os.Stat(d.installedServiceFilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("failed to stat installed service file: service file does not exist")
		}
		return fmt.Errorf("failed to stat installed service file: %w", err)
	}

	if err := d.Stop(); err != nil {
		return err
	}

	if err := os.Remove(d.installedServiceFilePath); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	return nil
}

func (d darwinService) Update() error {
	if err := d.uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall old service: %w", err)
	}

	if err := d.install(); err != nil {
		return fmt.Errorf("failed to install new service: %w", err)
	}

	return nil
}

func (d darwinService) Backup() error {
	if err := file.CopyFileNoOverwrite(d.logger.Named("copy-file"), d.installedServiceFilePath, path.BackupServiceFile(d.installDir)); err != nil {
		return fmt.Errorf("failed to copy service file: %w", err)
	}

	return nil
}
