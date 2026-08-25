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
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// launchdStderrPath is where com.observiq.collector.plist points launchd's
// StandardErrorPath. launchd opens it O_APPEND and owns the redirect; the
// process only sees it as fd 2.
const launchdStderrPath = "/var/log/observiq_collector.err"

// RunService runs the given service, calling its start and stop functions.
func RunService(logger *zap.Logger, rSvc RunnableService) error {
	// Bound the launchd-managed stderr file so a runaway error stream can't
	// fill the disk.
	capLaunchdStderr()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return runServiceInteractive(ctx, logger, rSvc)
}

// capLaunchdStderr starts the stderr file watchdog when the process's stderr
// is actually the launchd-managed file (i.e. running as the installed
// service). Interactive/dev runs are left alone. The file is capped in place
// (copy + truncate) because launchd holds the fd; there is no handle to swap.
func capLaunchdStderr() {
	pathInfo, err := os.Stat(launchdStderrPath)
	if err != nil {
		return
	}
	stderrInfo, err := os.Stderr.Stat()
	if err != nil || !os.SameFile(pathInfo, stderrInfo) {
		return
	}

	// Archive the previous run's output so the file starts fresh each launch.
	// This must copy+truncate (threshold 1) rather than rename: launchd's
	// already-open fd would follow a rename and keep growing the backup.
	// Running it synchronously also bounds crash-looping processes that die
	// before the watchdog's first tick.
	_ = capStderrFile(launchdStderrPath, 1)

	go watchStderrFile(launchdStderrPath)
}
