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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"context"
	"errors"

	"github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory creates a factory for Google Cloud Storage Exporter
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
		TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
		QueueConfig:   exporterhelper.NewDefaultQueueConfig(),
		BackOffConfig: configretry.NewDefaultBackOffConfig(),
		BucketName: "",
		ProjectID:  "",
		Location:   "",
		StorageClass: "",
		FolderName: "",
		BlobPrefix: "",
		// Partition:       minute,
	}
}

func createMetricsExporter(ctx context.Context, params exporter.Settings, config component.Config) (exporter.Metrics, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("not a Google Cloud Storage config")
	}
	exp, err := newExporter(cfg, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetrics(
		ctx,
		params,
		cfg,
		exp.metricsDataPusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithRetry(cfg.BackOffConfig),
	)
}

func createLogsExporter(ctx context.Context, params exporter.Settings, config component.Config) (exporter.Logs, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("not a Google Cloud Storage config")
	}
	exp, err := newExporter(cfg, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewLogs(
		ctx,
		params,
		cfg,
		exp.logsDataPusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithRetry(cfg.BackOffConfig),
	)
}

func createTracesExporter(ctx context.Context, params exporter.Settings, config component.Config) (exporter.Traces, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("not a Google Cloud Storage config")
	}
	exp, err := newExporter(cfg, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTraces(
		ctx,
		params,
		cfg,
		exp.traceDataPusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithRetry(cfg.BackOffConfig),
	)
}
