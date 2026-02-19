// Copyright  observIQ, Inc.
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

package statustestexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("statustest_exporter")

const stability = component.StabilityLevelDevelopment

// NewFactory creates a new factory for the statustest exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		componentType,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, stability),
		exporter.WithMetrics(createMetricsExporter, stability),
		exporter.WithTraces(createTracesExporter, stability),
	)
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	e := newStatusTestExporter(cfg.(*Config), set.Logger)
	return exporterhelper.NewLogs(
		ctx, set, cfg,
		func(_ context.Context, _ plog.Logs) error { return nil },
		exporterhelper.WithStart(e.Start),
		exporterhelper.WithShutdown(e.Shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

func createMetricsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	e := newStatusTestExporter(cfg.(*Config), set.Logger)
	return exporterhelper.NewMetrics(
		ctx, set, cfg,
		func(_ context.Context, _ pmetric.Metrics) error { return nil },
		exporterhelper.WithStart(e.Start),
		exporterhelper.WithShutdown(e.Shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	e := newStatusTestExporter(cfg.(*Config), set.Logger)
	return exporterhelper.NewTraces(
		ctx, set, cfg,
		func(_ context.Context, _ ptrace.Traces) error { return nil },
		exporterhelper.WithStart(e.Start),
		exporterhelper.WithShutdown(e.Shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}
