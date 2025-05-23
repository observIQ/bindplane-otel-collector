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

package snowflakeexporter

import (
	"context"
	"errors"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/exporter/snowflakeexporter/internal/database"
	"github.com/observiq/bindplane-otel-collector/exporter/snowflakeexporter/internal/metadata"
	"github.com/snowflakedb/gosnowflake"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory creates a new Snowflake exporter factory
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
	)
}

// createDefaultConfig creates the default configuration for the exporter
func createDefaultConfig() component.Config {
	// need to include default snowflake config, will be overwritten with user values
	sfCfg := buildDefaultSFConfig()
	dsn, _ := gosnowflake.DSN(sfCfg)

	return &Config{
		TimeoutConfig:    exporterhelper.NewDefaultTimeoutConfig(),
		QueueBatchConfig: exporterhelper.NewDefaultQueueConfig(),
		BackOffConfig:    configretry.NewDefaultBackOffConfig(),
		Database:         defaultDatabase,
		Logs: TelemetryConfig{
			Schema: defaultLogsSchema,
			Table:  defaultTable,
		},
		Metrics: TelemetryConfig{
			Schema: defaultMetricsSchema,
			Table:  defaultTable,
		},
		Traces: TelemetryConfig{
			Schema: defaultTracesSchema,
			Table:  defaultTable,
		},
		dsn: dsn,
	}
}

// createLogsExporter creates a new log exporter based on the config
func createLogsExporter(
	ctx context.Context,
	params exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	c, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	e, err := newLogsExporter(ctx, c, params, database.CreateSnowflakeDatabase)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs exporter: %w", err)
	}

	return exporterhelper.NewLogs(
		ctx,
		params,
		c,
		e.logsDataPusher,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(e.Capabilities()),
		exporterhelper.WithTimeout(e.cfg.TimeoutConfig),
		exporterhelper.WithQueue(e.cfg.QueueBatchConfig),
		exporterhelper.WithRetry(e.cfg.BackOffConfig),
	)
}

// createMetricsExporter creates a new metric exporter based on the config
func createMetricsExporter(
	ctx context.Context,
	params exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	c, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	e, err := newMetricsExporter(ctx, c, params, database.CreateSnowflakeDatabase)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	return exporterhelper.NewMetrics(
		ctx,
		params,
		c,
		e.metricsDataPusher,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(e.Capabilities()),
		exporterhelper.WithTimeout(e.cfg.TimeoutConfig),
		exporterhelper.WithQueue(e.cfg.QueueBatchConfig),
		exporterhelper.WithRetry(e.cfg.BackOffConfig),
	)
}

// createTracesExporter creates a new trace exporter based on the config
func createTracesExporter(
	ctx context.Context,
	params exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	c, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	e, err := newTracesExporter(ctx, c, params, database.CreateSnowflakeDatabase)
	if err != nil {
		return nil, fmt.Errorf("failed to create traces exporter: %w", err)
	}

	return exporterhelper.NewTraces(
		ctx,
		params,
		c,
		e.tracesDataPusher,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(e.Capabilities()),
		exporterhelper.WithTimeout(e.cfg.TimeoutConfig),
		exporterhelper.WithQueue(e.cfg.QueueBatchConfig),
		exporterhelper.WithRetry(e.cfg.BackOffConfig),
	)
}
