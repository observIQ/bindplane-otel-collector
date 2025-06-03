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

	logs := make([]map[string]any, 0, ld.ResourceLogs().Len())

	for _, resourceLog := range ld.ResourceLogs().All() {
		logs = append(logs, map[string]any{
			"resourceLog": resourceLog,
		})
	}
	body, err := json.Marshal(logs)
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
