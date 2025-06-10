// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhookexporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logsExporter struct {
	cfg      *SignalConfig
	logger   *zap.Logger
	settings component.TelemetrySettings
	client   *http.Client
}

func newLogsExporter(
	_ context.Context,
	cfg *SignalConfig,
	params exporter.Settings,
) (*logsExporter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("logs config is required")
	}

	return &logsExporter{
		cfg:      cfg,
		logger:   params.Logger,
		settings: params.TelemetrySettings,
	}, nil
}

func (le *logsExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (le *logsExporter) start(_ context.Context, host component.Host) error {
	le.logger.Info("starting webhook logs exporter")
	client, err := le.cfg.ClientConfig.ToClient(context.Background(), host, le.settings)
	if err != nil {
		return fmt.Errorf("failed to create http client: %w", err)
	}
	le.logger.Debug("created http client", zap.String("endpoint", le.cfg.ClientConfig.Endpoint))
	le.client = client
	return nil
}

func (le *logsExporter) shutdown(_ context.Context) error {
	le.logger.Info("shutting down webhook logs exporter")
	if le.client != nil {
		le.client.CloseIdleConnections()
	}
	le.logger.Info("webhook logs exporter shutdown complete")
	return nil
}

func (le *logsExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	le.logger.Debug("begin webhook logsDataPusher")

	limit := int(le.cfg.QueueBatchConfig.QueueSize)
	if limit <= 0 {
		// If no limit is set, send all logs in one request
		return le.sendLogs(ctx, extractLogBodies(ld))
	}

	// Create a new logs object for the current batch
	logs := extractLogBodies(ld)

	batches := make([][]any, 0)
	currentBatch := make([]any, 0)
	currentBatchSize := 0
	// split logs into batches
	for _, log := range logs {
		if currentBatchSize >= limit {
			batches = append(batches, currentBatch)
			currentBatch = make([]any, 0)
			currentBatchSize = 0
		}
		currentBatch = append(currentBatch, log)
		currentBatchSize++
	}
	// Append the last batch if it's not empty
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	le.logger.Debug("created log batches", zap.Int("num_batches", len(batches)), zap.Int("batch_size", limit))

	// Send each batch
	for i, batch := range batches {
		if err := le.sendLogs(ctx, batch); err != nil {
			return fmt.Errorf("failed to send batch %d: %w", i+1, err)
		}
		le.logger.Debug("sent log batch", zap.Int("batch_number", i+1), zap.Int("total_batches", len(batches)))
	}

	return nil
}

func extractLogBodies(ld plog.Logs) []any {
	return extractLogsFromResourceLogs(ld.ResourceLogs())
}

func extractLogsFromResourceLogs(resourceLogs plog.ResourceLogsSlice) []any {
	logs := make([]any, 0)
	for i := 0; i < resourceLogs.Len(); i++ {
		logs = append(logs, extractLogsFromScopeLogs(resourceLogs.At(i).ScopeLogs())...)
	}
	return logs
}

func extractLogsFromScopeLogs(scopeLogs plog.ScopeLogsSlice) []any {
	logs := make([]any, 0)
	for i := 0; i < scopeLogs.Len(); i++ {
		logs = append(logs, extractLogsFromLogRecords(scopeLogs.At(i).LogRecords())...)
	}
	return logs
}

func extractLogsFromLogRecords(logRecords plog.LogRecordSlice) []any {
	logs := make([]any, 0, logRecords.Len())
	for i := 0; i < logRecords.Len(); i++ {
		logStr := logRecords.At(i).Body().AsString()
		var parsedLog any
		if err := json.Unmarshal([]byte(logStr), &parsedLog); err != nil {
			// If the log isn't valid JSON, keep it as a string
			logs = append(logs, logStr)
			continue
		}
		logs = append(logs, parsedLog)
	}
	return logs
}

func (le *logsExporter) sendLogs(ctx context.Context, logs []any) error {
	body, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, string(le.cfg.Verb), le.cfg.ClientConfig.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range le.cfg.ClientConfig.Headers {
		request.Header.Set(key, string(value))
	}

	request.Header.Set("Content-Type", le.cfg.ContentType)

	response, err := le.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("failed to send request: %s", response.Status)
	}

	return nil
}
