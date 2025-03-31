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
	"time"

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
	sentinelLogs, err := e.transformLogsToSentinelFormat(ld)
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

func (e *microsoftSentinelExporter) transformLogsToSentinelFormat(ld plog.Logs) ([]byte, error) {
	var sentinelLogs []map[string]interface{}

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)

		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)

			for k := 0; k < sl.LogRecords().Len(); k++ {
				logRecord := sl.LogRecords().At(k)

				// Convert the timestamp to RFC3339 format for TimeGenerated
				timeGenerated := logRecord.Timestamp().AsTime().Format(time.RFC3339)

				// Create a new single-record logs object to hold just this log
				singleLogRecord := plog.NewLogs()
				resourceLog := singleLogRecord.ResourceLogs().AppendEmpty()

				// Copy resource
				rl.Resource().CopyTo(resourceLog.Resource())

				// Copy scope
				scopeLog := resourceLog.ScopeLogs().AppendEmpty()
				sl.Scope().CopyTo(scopeLog.Scope())

				// Copy log record
				logRecordCopy := scopeLog.LogRecords().AppendEmpty()
				logRecord.CopyTo(logRecordCopy)

				// Use the JSONMarshaler to convert to JSON
				marshaler := plog.JSONMarshaler{}
				logJSON, err := marshaler.MarshalLogs(singleLogRecord)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal log to JSON: %w", err)
				}

				// Unmarshal the JSON to a map so we can access the inner resourceLogs
				var logMap map[string]interface{}
				if err := json.Unmarshal(logJSON, &logMap); err != nil {
					return nil, fmt.Errorf("failed to unmarshal log JSON: %w", err)
				}

				// Extract the inner resourceLogs array directly
				innerResourceLogs, ok := logMap["resourceLogs"]
				if !ok {
					return nil, fmt.Errorf("unexpected structure: resourceLogs not found")
				}

				// Create the Sentinel format with TimeGenerated and the inner resourceLogs
				sentinelLog := map[string]interface{}{
					"TimeGenerated": timeGenerated,
					"resourceLogs":  innerResourceLogs, // Use the inner content directly
				}

				sentinelLogs = append(sentinelLogs, sentinelLog)
			}
		}
	}

	jsonLogs, err := json.Marshal(sentinelLogs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert logs to JSON: %w", err)
	}
	return jsonLogs, nil
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
