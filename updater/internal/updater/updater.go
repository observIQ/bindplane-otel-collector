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
	"context"
	"errors"
	"fmt"
	"os"
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

// Constants for service file templates
// Removed redeclared constants that are now in templates.go

// Updater is a struct that can be used to perform a collector update
type Updater struct {
	installDir string
	installer  install.Installer
	svc        service.Service
	rollbacker rollback.Rollbacker
	monitor    state.Monitor
	logger     *zap.Logger
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
		installDir: installDir,
		installer:  install.NewInstaller(logger, installDir, svc),
		svc:        svc,
		rollbacker: rollback.NewRollbacker(logger, installDir),
		monitor:    monitor,
		logger:     logger,
	}, nil
}

// generateServiceFiles writes necessary service files to the install directory.
func (u *Updater) generateServiceFiles() error {
	systemdServiceFilePath := filepath.Join(u.installDir, "install", "observiq-otel-collector.service")
	initServiceFilePath := filepath.Join(u.installDir, "install", "observiq-otel-collector")

	// Install dir needs to be pre-created if it does not exist already.
	if err := os.MkdirAll(filepath.Dir(systemdServiceFilePath), 0750); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}

	// #nosec G306 - Systemd service file should have 0640 permissions
	if err := os.WriteFile(systemdServiceFilePath, []byte(systemdServiceTemplate), 0640); err != nil {
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
	if err := u.generateServiceFiles(); err != nil {
		u.logger.Error("Failed to generate service files", zap.Error(err))
		return fmt.Errorf("failed to generate service files: %w", err)
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
