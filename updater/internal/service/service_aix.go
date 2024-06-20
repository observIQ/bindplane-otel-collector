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

//go:build aix

package service

import (
	"fmt"
	"os/exec"

	"go.uber.org/zap"
)

const aixUnixServiceIdentifier = "oiqcollector"
const aixUnixServiceName = "observiq-otel-collector"

// Option is an extra option for creating a Service
type Option func(aixUnixSvc *aixUnixService)

// WithServiceFile returns an option setting the service file to use when updating using the service
func WithServiceFile(svcFilePath string) Option {
	return func(aixUnixSvc *aixUnixService) {
		// Do nothing
	}
}

// NewService returns an instance of the Service interface for managing the observiq-otel-collector service on the current OS.
func NewService(logger *zap.Logger, installDir string, opts ...Option) Service {
	aixUnixSvc := &aixUnixService{
		serviceName:       aixUnixServiceName,
		serviceIdentifier: aixUnixServiceIdentifier,
		installDir:        installDir,
		logger:            logger.Named("aixUnix-service"),
	}

	for _, opt := range opts {
		opt(aixUnixSvc)
	}

	return aixUnixSvc
}

type aixUnixService struct {
	// newServiceFilePath a useless stub to please service_action.go
	newServiceFilePath string
	// serviceName is the name of the service
	serviceName string
	// serviceName is the name of the service
	serviceIdentifier string
	installDir        string
	logger            *zap.Logger
}

// Start the service
func (l aixUnixService) Start() error {
	// startsrc -s observiq-otel-collector -a start -e "$(cat /opt/observiq-otel-collector/observiq-otel-collector.env)"
	//#nosec G204 -- serviceName is not determined by user input
	cmd := exec.Command("startsrc", "-s", l.serviceName, "-a start -e \"$(cat /opt/observiq-otel-collector/observiq-otel-collector.env)\"")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running service failed: %w", err)
	}
	return nil
}

// Stop the service
func (l aixUnixService) Stop() error {
	// stopsrc -s observiq-otel-collector
	//#nosec G204 -- serviceName is not determined by user input
	cmd := exec.Command("stopsrc", "-s", l.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running service failed: %w", err)
	}

	return nil
}

// installs the service
func (l aixUnixService) install() error {
	// mkssys -s observiq-otel-collector -p /opt/observiq-otel-collector/observiq-otel-collector -u $(id -u observiq-otel-collector) -S -n15 -f9 -a '--config config.yaml --manager manager.yaml --logging logging.yaml'
	//#nosec G204 -- serviceName is not determined by user input
	cmd := exec.Command("mkssys", "-s", l.serviceName, "-p /opt/observiq-otel-collector/observiq-otel-collector -u $(id -u observiq-otel-collector) -S -n15 -f9 -a '--config config.yaml --manager manager.yaml --logging logging.yaml'")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enabling service file failed: %w", err)
	}
	// mkitab 'oiqcollector:23456789:respawn:startsrc -s observiq-otel-collector -a start -e "$(cat /opt/observiq-otel-collector/observiq-otel-collector.env)"'
	//#nosec G204 -- serviceName is not determined by user input
	cmd = exec.Command("mkitab", "'oiqcollector:23456789:respawn:startsrc -s", l.serviceName, "-a start -e \"$(cat /opt/observiq-otel-collector/observiq-otel-collector.env)\"'")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enabling service file failed: %w", err)
	}

	return nil
}

// uninstalls the service
func (l aixUnixService) uninstall() error {
	// stopsrc -s observiq-otel-collector
	//#nosec G204 -- serviceName is not determined by user input
	cmd := exec.Command("stopsrc", "-s", l.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running service failed: %w", err)
	}

	// rmitab oiqcollector
	//#nosec G204 -- serviceIdentifier is not determined by user input
	cmd = exec.Command("rmitab", l.serviceIdentifier)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reloading service failed: %w", err)
	}

	return nil
}

func (l aixUnixService) Update() error {
	if err := l.uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall old service: %w", err)
	}

	if err := l.install(); err != nil {
		return fmt.Errorf("failed to install new service: %w", err)
	}

	return nil
}

func (l aixUnixService) Backup() error {
	return nil
}
