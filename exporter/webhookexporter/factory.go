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
	"fmt"
	"time"

	"github.com/observiq/bindplane-otel-collector/exporter/webhookexporter/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/internal/version"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"
)

const (
	defaultUserAgent = "bindplane-otel-collector"
)

// NewFactory creates a new Webhook exporter factory
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig(),
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
	)
}

func createDefaultConfig() func() component.Config {
	userAgent := fmt.Sprintf("%s/%s", defaultUserAgent, version.Version())
	return func() component.Config {
		return &Config{
			LogsConfig: &SignalConfig{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "https://localhost",
					Timeout:  30 * time.Second,
					Headers: configopaque.MapList{
						{
							Name:  "User-Agent",
							Value: configopaque.String(userAgent),
						},
					},
				},
				Verb:             POST,
				ContentType:      "application/json",
				QueueBatchConfig: configoptional.Some(exporterhelper.NewDefaultQueueConfig()),
				BackOffConfig:    configretry.NewDefaultBackOffConfig(),
			},
		}
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
		exporterhelper.WithQueue(e.cfg.QueueBatchConfig),
		exporterhelper.WithRetry(e.cfg.BackOffConfig),
	)
}

// logsEncoding implements QueueBatchEncoding for logs
type logsEncoding struct{}

func (e *logsEncoding) Marshal(req xexporterhelper.Request) ([]byte, error) {
	return json.Marshal(req)
}

func (e *logsEncoding) Unmarshal(data []byte) (xexporterhelper.Request, error) {
	var req xexporterhelper.Request
	err := json.Unmarshal(data, &req)
	return req, err
}
