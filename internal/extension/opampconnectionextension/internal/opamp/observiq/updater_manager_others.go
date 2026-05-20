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

//go:build !windows

package observiq

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Ensure interface is satisfied
var _ updaterManager = (*othersUpdaterManager)(nil)

const defaultOthersUpdaterName = "updater"

// othersUpdaterManager handles starting a Updater binary and watching it for failure with a timeout
type othersUpdaterManager struct {
	tmpPath             string
	cwd                 string
	updaterName         string
	logger              *zap.Logger
	shutdownWaitTimeout time.Duration
}

// newUpdaterManager creates a new UpdaterManager
func newUpdaterManager(defaultLogger *zap.Logger, tmpPath string) (updaterManager, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get cwd: %w", err)
	}

	return &othersUpdaterManager{
		tmpPath:             filepath.Clean(tmpPath),
		logger:              defaultLogger.Named("updater manager"),
		updaterName:         defaultOthersUpdaterName,
		cwd:                 cwd,
		shutdownWaitTimeout: defaultShutdownWaitTimeout,
	}, nil
}

// StartAndMonitorUpdater will start the Updater binary and wait to see if it finishes unexpectedly.
// While waiting for Updater, it should kill the collector and we should never execute any code past running it
func (m othersUpdaterManager) StartAndMonitorUpdater() error {
	initialUpdaterPath := filepath.Join(m.tmpPath, updaterDir, m.updaterName)
	updaterPath, err := copyExecutable(m.logger.Named("copy-executable"), initialUpdaterPath, m.cwd)
	if err != nil {
		return fmt.Errorf("failed to copy updater to cwd: %w", err)
	}

	//#nosec G204 -- paths are not determined via user input
	cmd := exec.Command(updaterPath)

	// We need to set the processor group id to something different so that at least on mac, when the
	// collector dies the updater won't die as well
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
	// Start does not block
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("updater had an issue while starting: %w", err)
	}

	// See if we're still alive after waiting for the timeout to pass
	time.Sleep(m.shutdownWaitTimeout)

	// Updater might already be killed
	if err := cmd.Process.Kill(); err != nil {
		m.logger.Debug("Failed to kill failed Updater", zap.Error(err))
	}

	// Ideally we should not get here as we will be killed by the updater.
	// Updater should either exit before us with error or we die before it does.
	return errors.New("updater failed to update collector")
}
