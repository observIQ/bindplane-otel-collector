// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package webhookexporter implements an OpenTelemetry Logs exporter that sends logs to a webhook.
package webhookexporter

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/observiq/bindplane-otel-collector/exporter/webhookexporter/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory creates a new Webhook exporter factory
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		LogsConfig: &SignalConfig{
			ClientConfig: confighttp.ClientConfig{
				Endpoint: "https://localhost",
			},
			Verb:        POST,
			ContentType: "application/json",
		},
	}
}

func createLogsExporter(ctx context.Context, params exporter.Settings, config component.Config) (exporter.Logs, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("invalid configuration: expected *Config for Webhook exporter, but got a different type")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	e, err := newLogsExporter(ctx, cfg.LogsConfig, params)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogs(
		ctx,
		params,
		cfg,
		e.logsDataPusher,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(e.Capabilities()),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: e.cfg.ClientConfig.Timeout}),
		exporterhelper.WithQueueBatch(
			e.cfg.QueueBatchConfig,
			exporterhelper.QueueBatchSettings{
				Encoding: &logsEncoding{},
			},
		),
		exporterhelper.WithRetry(e.cfg.BackOffConfig),
	)
}

// logsEncoding implements QueueBatchEncoding for logs
type logsEncoding struct{}

func (e *logsEncoding) Marshal(req exporterhelper.Request) ([]byte, error) {
	return json.Marshal(req)
}

func (e *logsEncoding) Unmarshal(data []byte) (exporterhelper.Request, error) {
	var req exporterhelper.Request
	err := json.Unmarshal(data, &req)
	return req, err
}
