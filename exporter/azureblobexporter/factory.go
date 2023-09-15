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

package azureblobexporter // import "github.com/observiq/bindplane-agent/exporter/azureblobexporter"

import (
	"context"
	"errors"

	"github.com/observiq/bindplane-agent/exporter/azureblobexporter/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory creates a factory for Azure Blob Exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutSettings: exporterhelper.NewDefaultTimeoutSettings(),
		QueueSettings:   exporterhelper.NewDefaultQueueSettings(),
		RetrySettings:   exporterhelper.NewDefaultRetrySettings(),
		Partition:       minutePartition,
		Compression:     noCompression,
	}
}

func createMetricsExporter(ctx context.Context, params exporter.CreateSettings, config component.Config) (exporter.Metrics, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("not an Azure Blob config")
	}

	exp, err := newExporter(cfg, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetricsExporter(ctx,
		params,
		config,
		exp.metricsDataPusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
		exporterhelper.WithTimeout(cfg.TimeoutSettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithRetry(cfg.RetrySettings),
	)
}

func createLogsExporter(ctx context.Context, params exporter.CreateSettings, config component.Config) (exporter.Logs, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("not an Azure Blob config")
	}

	exp, err := newExporter(cfg, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewLogsExporter(ctx,
		params,
		config,
		exp.logsDataPusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
		exporterhelper.WithTimeout(cfg.TimeoutSettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithRetry(cfg.RetrySettings),
	)
}

func createTracesExporter(ctx context.Context, params exporter.CreateSettings, config component.Config) (exporter.Traces, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("not an Azure Blob config")
	}

	exp, err := newExporter(cfg, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTracesExporter(ctx,
		params,
		config,
		exp.tracesDataPusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
		exporterhelper.WithTimeout(cfg.TimeoutSettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithRetry(cfg.RetrySettings),
	)
}
