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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
}

// newExporter creates a new Microsoft Sentinel exporter
func newExporter(cfg *Config, params exporter.Settings) (*microsoftSentinelExporter, error) {
	logger := params.Logger

	// Create Azure credential
	var cred *azidentity.DefaultAzureCredential
	var err error

	if cfg.CredentialPath != "" {
		// If credential path is provided, set the environment variable for DefaultAzureCredential
		os.Setenv("AZURE_AUTH_LOCATION", cfg.CredentialPath)
	}

	cred, err = azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create Azure logs client
	client, err := azlogs.NewClient(cfg.Endpoint, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Microsoft Sentinel client: %w", err)
	}

	return &microsoftSentinelExporter{
		cfg:        cfg,
		logger:     logger,
		client:     client,
		ruleID:     cfg.RuleID,
		streamName: cfg.StreamName,
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
	jsonLogs, err := e.logsToJSON(ld)
	if err != nil {
		return fmt.Errorf("failed to convert logs to JSON: %w", err)
	}

	// Upload logs to Microsoft Sentinel
	_, err = e.client.Upload(ctx, e.ruleID, e.streamName, jsonLogs, nil)
	if err != nil {
		return fmt.Errorf("failed to upload logs to Microsoft Sentinel: %w", err)
	}

	e.logger.Debug("Successfully sent logs to Microsoft Sentinel",
		zap.Int("count", logsCount),
	)

	return nil
}

// logsToJSON converts OpenTelemetry logs to JSON format for Microsoft Sentinel
func (e *microsoftSentinelExporter) logsToJSON(ld plog.Logs) ([]byte, error) {
	logs := make([]map[string]interface{}, 0, ld.LogRecordCount())

	resourceLogs := ld.ResourceLogs()
	for i := range resourceLogs.Len() {
		rl := resourceLogs.At(i)
		resource := rl.Resource()

		scopeLogs := rl.ScopeLogs()
		for j := range scopeLogs.Len() {
			sl := scopeLogs.At(j)
			scope := sl.Scope()

			logRecords := sl.LogRecords()
			for k := range logRecords.Len() {
				lr := logRecords.At(k)

				// Create a log entry with all necessary fields
				logEntry := make(map[string]interface{})

				// Add standard fields
				logEntry["timestamp"] = lr.Timestamp().AsTime().Format(time.RFC3339Nano)
				logEntry["severity"] = lr.SeverityText()
				logEntry["severityNumber"] = lr.SeverityNumber()

				if lr.Body().Type() != pcommon.ValueTypeEmpty {
					logEntry["body"] = lr.Body().AsString()
				}

				// Add attributes from the log record
				attrs := make(map[string]interface{})
				lr.Attributes().Range(func(k string, v pcommon.Value) bool {
					attrs[k] = v.AsString()
					return true
				})
				logEntry["attributes"] = attrs

				// Add resource information
				resourceAttrs := make(map[string]interface{})
				resource.Attributes().Range(func(k string, v pcommon.Value) bool {
					resourceAttrs[k] = v.AsString()
					return true
				})
				logEntry["resource"] = resourceAttrs

				// Add instrumentation scope information
				logEntry["instrumentationScope"] = map[string]string{
					"name":    scope.Name(),
					"version": scope.Version(),
				}

				logs = append(logs, logEntry)
			}
		}
	}

	return json.Marshal(logs)
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
