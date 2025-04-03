// Copyright observIQ, Inc.
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

package microsoftsentinelexporter // import "github.com/observiq/bindplane-otel-collector/exporter/microsoftsentinelexporter"

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// microsoftSentinelExporter exports logs to Microsoft Sentinel's Log Analytics API
type microsoftSentinelExporter struct {
	cfg        *Config
	logger     *zap.Logger
	client     *azlogs.Client
	ruleID     string
	streamName string
	marshaler  *sentinelMarshaler
}

// newExporter creates a new Microsoft Sentinel exporter
func newExporter(cfg *Config, params exporter.Settings) (*microsoftSentinelExporter, error) {
	logger := params.Logger

	// Create Azure credential
	cred, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to verify Azure credential: %w", err)
	}

	// Create Azure logs client
	client, err := azlogs.NewClient(cfg.Endpoint, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Microsoft Sentinel client: %w", err)
	}

	marshaler := newMarshaler(cfg, params.TelemetrySettings)

	return &microsoftSentinelExporter{
		cfg:        cfg,
		logger:     logger,
		client:     client,
		ruleID:     cfg.RuleID,
		streamName: cfg.StreamName,
		marshaler:  marshaler,
	}, nil
}

// Capabilities returns the capabilities of the exporter
func (e *microsoftSentinelExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// logsDataPusher pushes log data to Microsoft Sentinel
func (e *microsoftSentinelExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	logsCount := ld.LogRecordCount()
	if logsCount == 0 {
		return nil
	}

	e.logger.Debug("Microsoft Sentinel exporter sending logs", zap.Int("count", logsCount))

	// Convert logs to JSON format expected by Sentinel
	sentinelLogs, err := e.marshaler.transformLogsToSentinelFormat(ctx, ld)
	if err != nil {
		return fmt.Errorf("failed to convert logs to Sentinel format: %w", err)
	}

	_, err = e.client.Upload(ctx, e.ruleID, e.streamName, sentinelLogs, nil)
	if err != nil {
		return fmt.Errorf("failed to upload logs to Microsoft Sentinel: %w", err)
	}

	e.logger.Debug("Successfully sent logs to Microsoft Sentinel",
		zap.Int("count", logsCount),
	)

	return nil
}

// Start starts the exporter
func (e *microsoftSentinelExporter) Start(ctx context.Context, host component.Host) error {
	e.logger.Info("Starting Microsoft Sentinel exporter")
	return nil
}

// Shutdown will shutdown the exporter
func (e *microsoftSentinelExporter) Shutdown(ctx context.Context) error {
	e.logger.Info("Shutting down Microsoft Sentinel exporter")
	return nil
}
