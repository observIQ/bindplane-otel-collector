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

package action

import (
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/observiq/bindplane-otel-collector/updater/internal/service"
	"go.uber.org/zap"
)

// ServiceStopAction is an action that records that a service was stopped.
type ServiceStopAction struct {
	svc service.Service
}

var _ RollbackableAction = (*ServiceStopAction)(nil)

// NewServiceStopAction creates a new ServiceStopAction
func NewServiceStopAction(svc service.Service) *ServiceStopAction {
	return &ServiceStopAction{
		svc: svc,
	}
}

// Rollback rolls back the stop action (starts the service)
func (s ServiceStopAction) Rollback() error {
	return s.svc.Start()
}

// ServiceStartAction is an action that records that a service was started.
type ServiceStartAction struct {
	svc service.Service
}

var _ RollbackableAction = (*ServiceStartAction)(nil)

// NewServiceStartAction creates a new ServiceStartAction
func NewServiceStartAction(svc service.Service) *ServiceStartAction {
	return &ServiceStartAction{
		svc: svc,
	}
}

// Rollback rolls back the start action (stops the service)
func (s ServiceStartAction) Rollback() error {
	return s.svc.Stop()
}

// ServiceUpdateAction is an action that records that a service was updated.
type ServiceUpdateAction struct {
	backupSvc service.Service
}

var _ RollbackableAction = (*ServiceUpdateAction)(nil)

// NewServiceUpdateAction creates a new ServiceUpdateAction
func NewServiceUpdateAction(logger *zap.Logger, installDir string) *ServiceUpdateAction {
	namedLogger := logger.Named("service-update-action")
	return &ServiceUpdateAction{
		backupSvc: service.NewService(
			namedLogger,
			installDir,
			service.WithServiceFile(path.BackupServiceFile(installDir)),
		),
	}
}

// Rollback is an action that rolls back the service configuration to the one saved in the backup directory.
func (s ServiceUpdateAction) Rollback() error {
	return s.backupSvc.Update()
}
