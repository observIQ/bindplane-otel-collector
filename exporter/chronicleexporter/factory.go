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

package chronicleexporter

import (
	"context"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory creates a new Chronicle exporter factory.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability))
}

const (
	defaultEndpoint                  = "malachiteingestion-pa.googleapis.com"
	defaultBatchRequestSizeLimitGRPC = 1048576
	defaultBatchRequestSizeLimitHTTP = 1048576
)

// createDefaultConfig creates the default configuration for the exporter.
func createDefaultConfig() component.Config {
	return &Config{
		Protocol:                  protocolGRPC,
		TimeoutConfig:             exporterhelper.NewDefaultTimeoutConfig(),
		QueueBatchConfig:          exporterhelper.NewDefaultQueueConfig(),
		BackOffConfig:             configretry.NewDefaultBackOffConfig(),
		OverrideLogType:           true,
		Compression:               noCompression,
		CollectAgentMetrics:       true,
		Endpoint:                  defaultEndpoint,
		BatchRequestSizeLimitGRPC: defaultBatchRequestSizeLimitGRPC,
		BatchRequestSizeLimitHTTP: defaultBatchRequestSizeLimitHTTP,
	}
}

// createLogsExporter creates a new log exporter based on this config.
func createLogsExporter(
	ctx context.Context,
	params exporter.Settings,
	cfg component.Config,
) (exp exporter.Logs, err error) {
	t, err := metadata.NewTelemetryBuilder(params.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("create telemetry builder: %w", err)
	}

	c := cfg.(*Config)
	if c.Protocol == protocolHTTPS {
		return newHTTPExporter(ctx, c, params, t)
	}
	return newGRPCExporter(ctx, c, params, t)
}
