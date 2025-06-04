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
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logsExporter struct {
	cfg    *Config
	logger *zap.Logger
	client *http.Client
}

func newLogsExporter(
	ctx context.Context,
	cfg *Config,
	params exporter.Settings,
) (*logsExporter, error) {
	if cfg.LogsConfig == nil {
		return nil, fmt.Errorf("logs config is required")
	}

	client := &http.Client{
		Timeout: cfg.LogsConfig.TimeoutConfig.Timeout,
	}

	return &logsExporter{
		cfg:    cfg,
		logger: params.Logger,
		client: client,
	}, nil
}

func (le *logsExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (le *logsExporter) start(ctx context.Context, _ component.Host) error {
	return nil
}

func (le *logsExporter) shutdown(_ context.Context) error {
	return nil
}

func (le *logsExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	if le.cfg.LogsConfig == nil {
		return fmt.Errorf("logs config is required")
	}

	le.logger.Debug("begin webhook logsDataPusher")

	limit := le.cfg.LogsConfig.Limit
	if limit <= 0 {
		// If no limit is set, send all logs in one request
		return le.sendLogs(ctx, ld)
	}

	// Create a new logs object for the current batch
	currentBatch := plog.NewLogs()
	batches := []plog.Logs{currentBatch}
	currentBatchSize := 0

	// Process each resource log
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		newResourceLog := currentBatch.ResourceLogs().AppendEmpty()
		resourceLog.Resource().CopyTo(newResourceLog.Resource())

		// Process each scope log
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			newScopeLog := newResourceLog.ScopeLogs().AppendEmpty()
			scopeLog.Scope().CopyTo(newScopeLog.Scope())

			// Process each log record
			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				// If we've reached the limit, create a new batch
				if currentBatchSize >= limit {
					currentBatch = plog.NewLogs()
					batches = append(batches, currentBatch)
					newResourceLog = currentBatch.ResourceLogs().AppendEmpty()
					resourceLog.Resource().CopyTo(newResourceLog.Resource())
					newScopeLog = newResourceLog.ScopeLogs().AppendEmpty()
					scopeLog.Scope().CopyTo(newScopeLog.Scope())
					currentBatchSize = 0
				}

				logRecord := scopeLog.LogRecords().At(k)
				newLogRecord := newScopeLog.LogRecords().AppendEmpty()
				logRecord.CopyTo(newLogRecord)
				currentBatchSize++
			}
		}
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

func (le *logsExporter) sendLogs(ctx context.Context, logs plog.Logs) error {
	body, err := (&plog.JSONMarshaler{}).MarshalLogs(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, string(le.cfg.LogsConfig.Verb), string(le.cfg.LogsConfig.Endpoint), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range le.cfg.LogsConfig.Headers {
		request.Header.Set(key, value)
	}

	request.Header.Set("Content-Type", le.cfg.LogsConfig.ContentType)

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
