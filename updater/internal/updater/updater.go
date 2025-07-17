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

// Package updater handles all aspects of updating the collector from a provided archive
package updater

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"text/template"
	"time"

	"path/filepath"

	"github.com/observiq/bindplane-otel-collector/packagestate"
	"github.com/observiq/bindplane-otel-collector/updater/internal/action"
	"github.com/observiq/bindplane-otel-collector/updater/internal/install"
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/observiq/bindplane-otel-collector/updater/internal/rollback"
	"github.com/observiq/bindplane-otel-collector/updater/internal/service"
	"github.com/observiq/bindplane-otel-collector/updater/internal/state"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

const (
	// DefaultSystemdUnitFilePath is the default path to the systemd unit file
	// for the collector service.
	DefaultSystemdUnitFilePath = "/usr/lib/systemd/system/observiq-otel-collector.service"
)

// Updater is a struct that can be used to perform a collector update
type Updater struct {
	installDir               string
	installer                install.Installer
	svc                      service.Service
	rollbacker               rollback.Rollbacker
	monitor                  state.Monitor
	logger                   *zap.Logger
	installedSystemdUnitPath string
}

// NewUpdater creates a new updater which can be used to update the installation based at
// installDir
func NewUpdater(logger *zap.Logger, installDir string) (*Updater, error) {
	monitor, err := state.NewCollectorMonitor(logger, installDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	svc := service.NewService(logger, installDir)
	return &Updater{
		installDir:               installDir,
		installer:                install.NewInstaller(logger, installDir, svc),
		svc:                      svc,
		rollbacker:               rollback.NewRollbacker(logger, installDir),
		monitor:                  monitor,
		logger:                   logger,
		installedSystemdUnitPath: DefaultSystemdUnitFilePath,
	}, nil
}

// readGroupFromSystemdFile reads the systemd unit file and extracts the Group value.
func (u *Updater) readGroupFromSystemdFile() (string, error) {
	// #nosec G304 - systemdUnitFilePath is not user configurable, and comes
	// from the constant DefaultSystemdUnitFilePath. Unit tests will override
	// this path to a test file.
	fileContent, err := os.ReadFile(u.installedSystemdUnitPath)
	if err != nil {
		return "", fmt.Errorf("failed to read systemd unit file: %w", err)
	}

	lines := bytes.Split(fileContent, []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("Group=")) {
			return string(bytes.TrimSpace(bytes.TrimPrefix(line, []byte("Group=")))), nil
		}
	}

	return "", errors.New("Group not found in systemd unit file")
}

// generateLinuxServiceFiles writes necessary service files to the install directory
// to be copied to their final locations by the updater.
func (u *Updater) generateLinuxServiceFiles() error {
	systemdServiceFilePath := filepath.Join(u.installDir, "install", "observiq-otel-collector.service")
	initServiceFilePath := filepath.Join(u.installDir, "install", "observiq-otel-collector")

	// Install dir needs to be pre-created if it does not exist already.
	if err := os.MkdirAll(filepath.Dir(systemdServiceFilePath), 0750); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}

	// Read the Group value from the systemd unit file
	group, err := u.readGroupFromSystemdFile()
	if err != nil {
		return fmt.Errorf("read group from systemd file %s: %w", u.installedSystemdUnitPath, err)
	}

	// Get the install directory from path package. This will default
	// to /opt/observiq-otel-collector unless BDOT_CONFIG_HOME is set
	// in a package config file such as /etc/default/observiq-otel-collector
	// or /etc/sysconfig/observiq-otel-collector.
	installDir, err := path.InstallDir(u.logger, path.DefaultConfigOverrides)
	if err != nil {
		return fmt.Errorf("read working directory from systemd file %s: %w", u.installedSystemdUnitPath, err)
	}

	params := map[string]string{
		"Group":      group,
		"InstallDir": installDir,
	}

	// Render the systemd service template with the Group value
	systemdTemplate, err := template.New("systemdService").Parse(systemdServiceTemplate)
	if err != nil {
		return fmt.Errorf("parse systemd service template: %w", err)
	}

	var systemdServiceContent bytes.Buffer
	if err := systemdTemplate.Execute(&systemdServiceContent, params); err != nil {
		return fmt.Errorf("execute systemd service template: %w", err)
	}

	// #nosec G306 - Systemd service file should have 0640 permissions
	if err := os.WriteFile(systemdServiceFilePath, systemdServiceContent.Bytes(), 0640); err != nil {
		return fmt.Errorf("write systemd service file: %w", err)
	}

	// #nosec G306 - init.d service file should have 0755 permissions
	if err := os.WriteFile(initServiceFilePath, []byte(initScriptTemplate), 0755); err != nil {
		return fmt.Errorf("write init.d script file: %w", err)
	}

	return nil
}

// Update performs the update of the collector binary
func (u *Updater) Update() error {
	// Generate service files before stopping the service. If
	// this fails, the collector will still be running.
	if runtime.GOOS == "linux" {
		if err := u.generateLinuxServiceFiles(); err != nil {
			return fmt.Errorf("failed to generate service files: %w", err)
		}
	}

	// Stop the service before backing up the install directory;
	// We want to stop as early as possible so that we don't hit the collector's timeout
	// while it waits to be shutdown.
	if err := u.svc.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	// Record that we stopped the service
	u.rollbacker.AppendAction(action.NewServiceStopAction(u.svc))

	// Now that we stopped the service, it will be our responsibility to cleanup the tmp dir.
	// We will do this regardless of whether we succeed or fail after this point.
	defer u.removeTmpDir()

	u.logger.Debug("Stopped the service")

	// Create the backup
	if err := u.rollbacker.Backup(); err != nil {
		u.logger.Error("Failed to backup", zap.Error(err))

		// Set the state to failed before rollback so collector knows it failed
		if setErr := u.monitor.SetState(packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err); setErr != nil {
			u.logger.Error("Failed to set state on backup failure", zap.Error(setErr))
		}

		u.rollbacker.Rollback()

		u.logger.Error("Rollback complete")
		return fmt.Errorf("failed to backup: %w", err)
	}

	// Install artifacts
	if err := u.installer.Install(u.rollbacker); err != nil {
		u.logger.Error("Failed to install", zap.Error(err))

		// Set the state to failed before rollback so collector knows it failed
		if setErr := u.monitor.SetState(packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err); setErr != nil {
			u.logger.Error("Failed to set state on install failure", zap.Error(setErr))
		}

		u.rollbacker.Rollback()

		u.logger.Error("Rollback complete")
		return fmt.Errorf("failed to install: %w", err)
	}

	// Create a context with timeout to wait for a success or failed status
	checkCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	u.logger.Debug("Installation successful, begin monitor for success")

	// Monitor the install state
	if err := u.monitor.MonitorForSuccess(checkCtx, packagestate.CollectorPackageName); err != nil {
		u.logger.Error("Failed to install", zap.Error(err))

		// If this is not an error due to the collector setting a failed status we need to set a failed status
		if !errors.Is(err, state.ErrFailedStatus) {
			// Set the state to failed before rollback so collector knows it failed
			if setErr := u.monitor.SetState(packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err); setErr != nil {
				u.logger.Error("Failed to set state on install failure", zap.Error(setErr))
			}
		}

		u.rollbacker.Rollback()

		u.logger.Error("Rollback complete")
		return fmt.Errorf("failed while monitoring for success: %w", err)
	}

	// Successful update
	u.logger.Info("Update Complete")
	return nil
}

// removeTmpDir removes the temporary directory that holds the update artifacts.
func (u *Updater) removeTmpDir() {
	err := os.RemoveAll(path.TempDir(u.installDir))
	if err != nil {
		u.logger.Error("failed to remove temporary directory", zap.Error(err))
	}
}
