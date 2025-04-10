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

package azureloganalyticsexporter // import "github.com/observiq/bindplane-otel-collector/exporter/azureloganalyticsexporter"

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

// azureLogAnalyticsExporter exports logs to Azure Log Analytics
type azureLogAnalyticsExporter struct {
	cfg        *Config
	logger     *zap.Logger
	client     *azlogs.Client
	ruleID     string
	streamName string
	marshaler  *azureLogAnalyticsMarshaler
}

// newExporter creates a new Azure Log Analytics exporter
func newExporter(cfg *Config, params exporter.Settings) (*azureLogAnalyticsExporter, error) {
	logger := params.Logger

	// Create Azure credential
	cred, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to verify Azure credential: %w", err)
	}

	// Create Azure logs client
	client, err := azlogs.NewClient(cfg.Endpoint, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Log Analytics client: %w", err)
	}

	marshaler := newMarshaler(cfg, params.TelemetrySettings)

	return &azureLogAnalyticsExporter{
		cfg:        cfg,
		logger:     logger,
		client:     client,
		ruleID:     cfg.RuleID,
		streamName: cfg.StreamName,
		marshaler:  marshaler,
	}, nil
}

// Capabilities returns the capabilities of the exporter
func (e *azureLogAnalyticsExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// logsDataPusher pushes log data to Azure Log Analytics
func (e *azureLogAnalyticsExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	logsCount := ld.LogRecordCount()
	if logsCount == 0 {
		return nil
	}

	e.logger.Debug("Microsoft Sentinel exporter sending logs", zap.Int("count", logsCount))

	// Convert logs to JSON format expected by Azure Log Analytics
	azureLogAnalyticsLogs, err := e.marshaler.transformLogsToSentinelFormat(ctx, ld)

	fmt.Println(string(azureLogAnalyticsLogs))
	//test, err := json.Marshal("hello")
	if err != nil {
		return fmt.Errorf("failed to convert logs to Azure Log Analytics format: %w", err)
	}

	_, err = e.client.Upload(ctx, e.ruleID, e.streamName, azureLogAnalyticsLogs, nil)
	if err != nil {
		return fmt.Errorf("failed to upload logs to Azure Log Analytics: %w", err)
	}

	e.logger.Debug("Successfully sent logs to Azure Log Analytics",
		zap.Int("count", logsCount),
	)

	return nil
}

// Start starts the exporter
func (e *azureLogAnalyticsExporter) Start(_ context.Context, _ component.Host) error {
	e.logger.Info("Starting Azure Log Analytics exporter")
	return nil
}

// Shutdown will shutdown the exporter
func (e *azureLogAnalyticsExporter) Shutdown(_ context.Context) error {
	e.logger.Info("Shutting down Azure Log Analytics exporter")
	return nil
}
