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

// Package service provides a service wrapper around the collector regardless of managed or standalone mode.
package service

import (
	"context"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/internal/measurements"
	"github.com/observiq/bindplane-otel-collector/internal/topology"

	"github.com/observiq/bindplane-otel-collector/collector"
	"github.com/observiq/bindplane-otel-collector/internal/version"
	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/observiq/bindplane-otel-collector/opamp/observiq"
	"go.uber.org/zap"
)

// ManagedCollectorService is a RunnableService that runs the collector being managed by an OpAmp enabled platform
type ManagedCollectorService struct {
	logger *zap.Logger
	client opamp.Client

	// Config paths
	managerConfigPath   string
	collectorConfigPath string
	loggerConfigPath    string
}

// NewManagedCollectorService creates a new ManagedCollectorService
func NewManagedCollectorService(col collector.Collector, logger *zap.Logger, managerConfigPath, collectorConfigPath, loggerConfigPath string) (*ManagedCollectorService, error) {
	opampConfig, err := opamp.ParseConfig(managerConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manager config: %w", err)
	}

	// Create client Args
	clientArgs := &observiq.NewClientArgs{
		DefaultLogger:        logger,
		Config:               *opampConfig,
		Collector:            col,
		Version:              version.Version(),
		TmpPath:              "./tmp",
		ManagerConfigPath:    managerConfigPath,
		CollectorConfigPath:  collectorConfigPath,
		LoggerConfigPath:     loggerConfigPath,
		MeasurementsReporter: measurements.BindplaneAgentThroughputMeasurementsRegistry,
		TopologyReporter:     topology.BindplaneAgentTopologyRegistry,
	}

	// Create new client
	client, err := observiq.NewClient(clientArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create observIQ client: %w", err)
	}

	return &ManagedCollectorService{
		client:              client,
		logger:              logger,
		managerConfigPath:   managerConfigPath,
		collectorConfigPath: collectorConfigPath,
		loggerConfigPath:    loggerConfigPath,
	}, nil
}

// Start initiates the OpAmp connection and starts the collector
func (m *ManagedCollectorService) Start(ctx context.Context) error {
	m.logger.Info("Starting in managed mode")

	// Connect to manager platform
	if err := m.client.Connect(ctx); err != nil {
		return fmt.Errorf("error during OpAmp connection: %w", err)
	}

	return nil
}

// Stop stops the collector and disconnects from the platform
func (m *ManagedCollectorService) Stop(ctx context.Context) error {
	m.logger.Info("Shutting down collector")
	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("error during client disconnect: %w", err)
	}
	return nil
}

// Error returns an empty error channel. This will never send errors.
func (m *ManagedCollectorService) Error() <-chan error {
	// send new channel that's never used
	return make(<-chan error)
}
