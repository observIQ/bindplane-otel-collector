// Copyright  Bindplane, Inc.
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

// Package state contains structures to monitor and update the state of the collector in the package status
package state

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/observiq/bindplane-otel-collector/packagestate"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

var (
	// ErrFailedStatus is the error when the Package status indicates a failure
	ErrFailedStatus = errors.New("package status indicates failure")
)

// Monitor allows checking and setting state of active install
//
//go:generate mockery --name Monitor --filename mock_monitor.go --structname MockMonitor
type Monitor interface {
	// SetState sets the state for the package.
	// If passed in statusErr is not nil it will record the error as the message
	SetState(packageName string, status protobufs.PackageStatusEnum, statusErr error) error

	// MonitorForSuccess will periodically check the state of the package. It will keep checking until the context is canceled or a failed/success state is detected.
	// It will return an error if status is Failed or if the context times out.
	MonitorForSuccess(ctx context.Context, packageName string) error
}

// CollectorMonitor implements Monitor interface for monitoring the Collector Package Status file
type CollectorMonitor struct {
	stateManager  packagestate.StateManager
	currentStatus *protobufs.PackageStatuses
}

// NewCollectorMonitor create a new Monitor specifically for the collector
func NewCollectorMonitor(logger *zap.Logger, installDir string) (Monitor, error) {
	namedLogger := logger.Named("collector-monitor")

	// Create a collector monitor
	packageStatusPath := filepath.Join(installDir, packagestate.DefaultFileName)
	collectorMonitor := &CollectorMonitor{
		stateManager: packagestate.NewFileStateManager(namedLogger, packageStatusPath),
	}

	// Load the current status to ensure the package status file exists
	var err error
	collectorMonitor.currentStatus, err = collectorMonitor.stateManager.LoadStatuses()
	if err != nil {
		return nil, fmt.Errorf("failed to load package statues: %w", err)
	}

	return collectorMonitor, nil

}

// SetState sets the status on the specified package and saves it to the package status file
func (c *CollectorMonitor) SetState(packageName string, status protobufs.PackageStatusEnum, statusErr error) error {
	// Verify we have package by that name
	targetPackage, ok := c.currentStatus.GetPackages()[packageName]
	if !ok {
		return fmt.Errorf("no package for name %s", packageName)
	}

	// Update the status
	targetPackage.Status = status

	// If that passed in error is not nil set it as the error message
	if statusErr != nil {
		targetPackage.ErrorMessage = statusErr.Error()
	}

	c.currentStatus.GetPackages()[packageName] = targetPackage

	// Save to updated status to disk
	return c.stateManager.SaveStatuses(c.currentStatus)
}

// MonitorForSuccess intermittently checks the package status file for either an install failed or success status.
// If an InstallFailed status is read this returns ErrFailedStatus error.
// If the context is canceled the context error will be returned.
func (c *CollectorMonitor) MonitorForSuccess(ctx context.Context, packageName string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			packageStatus, err := c.stateManager.LoadStatuses()
			switch {
			// If there is any error we'll just continue. Some valid reasons we could error and should retry:
			// - File was deleted by new collector before it's rewritten
			// - File is being written to while we're reading it so we'll get invalid JSON
			case err != nil:
				continue
			default:
				targetPackage, ok := packageStatus.GetPackages()[packageName]
				// Target package might not exist yet so continue
				if !ok {
					continue
				}

				switch targetPackage.GetStatus() {
				case protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed:
					return ErrFailedStatus
				case protobufs.PackageStatusEnum_PackageStatusEnum_Installed:
					// Install successful
					return nil
				default:
					// Collector may still be starting up or we may have read the file while it's being written
					continue
				}
			}
		}
	}
}
