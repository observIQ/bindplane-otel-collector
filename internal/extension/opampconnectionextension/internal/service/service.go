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

package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

const (
	startTimeout = 10 * time.Second
	stopTimeout  = 10 * time.Second
)

// RunnableService may be run as a service.
//
//go:generate mockery --name RunnableService --filename mock_runnable_service.go --structname MockRunnableService
type RunnableService interface {
	// Start asynchronously starts the underlying service. The service may not necessarily be "ready"
	// once this returns, but could be asynchronously starting up.
	Start(ctx context.Context) error
	// Stop synchronously shuts down the service. After this function returns, the underlying service should be completely stopped.
	Stop(ctx context.Context) error
	// Error returns an error channel that should emit an error when the service must unexpectedly quit.
	Error() <-chan error
}

// runServiceInteractive runs the service in an "interactive" mode (responds to SIGINT and SIGTERM).
// This mode is always used in linux, and is used in Windows when the collector
// is not running as a service.
func runServiceInteractive(ctx context.Context, logger *zap.Logger, svc RunnableService) error {
	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	var svcErr error
	// Service is started; Wait for a stop signal.
	select {
	case <-ctx.Done():
	case svcErr = <-svc.Error():
		logger.Error("Unexpected error while running service", zap.Error(svcErr))
	}

	stopTimeoutCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
	defer stopCancel()

	if err := svc.Stop(stopTimeoutCtx); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return svcErr
}
