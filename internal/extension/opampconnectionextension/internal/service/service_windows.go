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
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

// windowsServiceShutdownTimeout is the amount of time to wait for the underlying service to stop before
// forcefully stopping the process.
var windowsServiceShutdownTimeout = 20 * time.Second

// The following constants specify error codes for the service.
// See https://docs.microsoft.com/en-us/windows/win32/debug/system-error-codes--1000-1299-
const (
	statusCodeInvalidServiceCommand = uint32(1052)
	statusCodeServiceException      = uint32(1064)
	statusCodeInvalidServiceName    = uint32(1213)
)

func RunService(logger *zap.Logger, rSvc RunnableService) error {
	isService, err := checkIsService()
	if err != nil {
		return fmt.Errorf("failed checking if running as service: %w", err)
	}

	if isService {
		// Change working directory to executable directory
		ex, err := os.Executable()
		if err != nil {
			logger.Warn("Failed to retrieve executable directory", zap.Error(err))
		} else {
			execDirPath := filepath.Dir(ex)
			if err := os.Chdir(execDirPath); err != nil {
				logger.Warn("Failed to modify current working directory", zap.Error(err))
			}
		}

		// Redirect stderr to file, so we can see panic information
		if err := redirectStderr(); err != nil {
			logger.Error("Failed to redirect stderr", zap.Error(err))
		}

		// Service name doesn't need to be specified when directly run by the service manager.
		return svc.Run("", newWindowsServiceHandler(logger, rSvc))
	} else {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		return runServiceInteractive(ctx, logger, rSvc)
	}
}

// windowsServiceHandler implements svc.Handler
type windowsServiceHandler struct {
	svc    RunnableService
	logger *zap.Logger
}

// newWindowsServiceHandler creates a new windowsServiceHandler, which implements svc.Handler
func newWindowsServiceHandler(logger *zap.Logger, svc RunnableService) *windowsServiceHandler {
	return &windowsServiceHandler{
		svc:    svc,
		logger: logger,
	}
}

// Execute handles the Windows service event loop.
func (sh *windowsServiceHandler) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	if len(args) == 0 {
		// Service name is the first argument, and must be provided to open the event log for service logs.
		return false, statusCodeInvalidServiceName
	}

	s <- svc.Status{State: svc.StartPending}

	err := sh.svc.Start(context.Background())
	if err != nil {
		sh.logger.Error("Failed to start service", zap.Error(err))
		return false, statusCodeServiceException
	}

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				s <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				err := sh.shutdown(s)
				if err != nil {
					sh.logger.Error("Failed during service shutdown", zap.Error(err))
					return false, statusCodeServiceException
				}

				return false, 0
			default:
				sh.logger.Error("Got unexpected service command", zap.Uint32("command", uint32(req.Cmd)))
				err := sh.shutdown(s)
				if err != nil {
					sh.logger.Error("Failed during service shutdown", zap.Error(err))
					return false, statusCodeServiceException
				}

				return false, statusCodeInvalidServiceCommand
			}
		case err := <-sh.svc.Error():
			sh.logger.Error("Got unexpected service error", zap.Error(err))

			sh.shutdown(s)

			if err != nil {
				sh.logger.Error("Failed during service shutdown", zap.Error(err))
			}

			return false, statusCodeServiceException
		}
	}
}

func (sh windowsServiceHandler) shutdown(s chan<- svc.Status) error {
	s <- svc.Status{State: svc.StopPending}

	stopTimeoutCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
	defer stopCancel()

	stopErrChan := make(chan error, 1)
	go func() {
		stopErrChan <- sh.svc.Stop(stopTimeoutCtx)
	}()

	var err error
	select {
	case <-time.After(windowsServiceShutdownTimeout):
		err = fmt.Errorf("the service failed to shut down in a timely manner (timeout: %s)", windowsServiceShutdownTimeout)
	case stopErr := <-stopErrChan:
		err = stopErr
	}

	s <- svc.Status{State: svc.Stopped}

	return err
}

// checkIsService returns whether the current process is running as a Windows service.
func checkIsService() (bool, error) {
	// NO_WINDOWS_SERVICE may be set non-zero to override the service detection logic.
	if value, present := os.LookupEnv("NO_WINDOWS_SERVICE"); present && value != "0" {
		return true, nil
	}

	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		return false, fmt.Errorf("failed to determine if we are running in an windows service: %w", err)
	}

	return isWindowsService, nil
}

// redirectStderr redirects stderr so that panic information is output to $INSTALL_DIR/log/observiq_collector.err,
// instead of it being dropped by Windows services.
// Most output should go through the zap logger instead of to stderr.
func redirectStderr() error {
	homeDir, ok := os.LookupEnv("OIQ_OTEL_COLLECTOR_HOME")
	if !ok {
		return errors.New("OIQ_OTEL_COLLECTOR_HOME environment variable not set")
	}

	path := filepath.Join(homeDir, "log", "observiq_collector.err")
	f, err := os.OpenFile(filepath.Clean(path), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd())); err != nil {
		return fmt.Errorf("failed to set stderr handle: %w (close err: %s)", err, f.Close())
	} else {
		os.Stderr = f
	}

	return nil
}
