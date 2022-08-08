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

// Package googlecloudexporter provides a wrapper around the official googlecloudexporter component that exposes some quality of life improvements in configuration
package googlecloudexporter

import (
	"context"
	"fmt"

	gcp "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/batchprocessor"
)

// gcpFactory is the factory used to create the underlying gcp exporter
var gcpFactory = gcp.NewFactory()

const (
	// typeStr is the type of the google cloud exporter
	typeStr = "googlecloud"

	// The stability level of the exporter. Matches the current exporter in contrib
	stability = component.StabilityLevelBeta
)

// NewFactory creates a factory for the googlecloud exporter
func NewFactory() component.ExporterFactory {
	return component.NewExporterFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsExporterAndStabilityLevel(createMetricsExporter, stability),
		component.WithLogsExporterAndStabilityLevel(createLogsExporter, stability),
		component.WithTracesExporterAndStabilityLevel(createTracesExporter, stability),
	)
}

// createMetricsExporter creates a metrics exporter based on this config.
func createMetricsExporter(ctx context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.MetricsExporter, error) {
	exporterConfig := cfg.(*Config)
	exporterConfig.setClientOptions()

	gcpExporter, err := gcpFactory.CreateMetricsExporter(ctx, set, exporterConfig.GCPConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	processors := []component.MetricsProcessor{}
	processorConfigs := []config.Processor{
		exporterConfig.BatchConfig,
	}

	processorFactories := []component.ProcessorFactory{
		batchprocessor.NewFactory(),
	}

	processorSettings := component.ProcessorCreateSettings{
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	var consumer consumer.Metrics = gcpExporter
	for i, processorConfig := range processorConfigs {
		processorConfig.SetIDName(exporterConfig.ID().String())
		factory := processorFactories[i]
		processor, err := factory.CreateMetricsProcessor(ctx, processorSettings, processorConfig, consumer)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics processor %s: %w", processorConfig.ID().String(), err)
		}
		processors = append(processors, processor)
		consumer = processor
	}

	return &exporter{
		appendHost:        exporterConfig.AppendHost,
		metricsProcessors: processors,
		metricsExporter:   gcpExporter,
		metricsConsumer:   consumer,
	}, nil
}

// createLogExporter creates a logs exporter based on this config.
func createLogsExporter(ctx context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.LogsExporter, error) {
	exporterConfig := cfg.(*Config)
	exporterConfig.setClientOptions()

	gcpExporter, err := gcpFactory.CreateLogsExporter(ctx, set, exporterConfig.GCPConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs exporter: %w", err)
	}

	processors := []component.LogsProcessor{}
	processorConfigs := []config.Processor{
		exporterConfig.BatchConfig,
	}

	processorFactories := []component.ProcessorFactory{
		batchprocessor.NewFactory(),
	}

	processorSettings := component.ProcessorCreateSettings{
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	var consumer consumer.Logs = gcpExporter
	for i, processorConfig := range processorConfigs {
		processorConfig.SetIDName(exporterConfig.ID().String())
		factory := processorFactories[i]
		processor, err := factory.CreateLogsProcessor(ctx, processorSettings, processorConfig, consumer)
		if err != nil {
			return nil, fmt.Errorf("failed to create logs processor %s: %w", processorConfig.ID().String(), err)
		}
		processors = append(processors, processor)
		consumer = processor
	}

	return &exporter{
		appendHost:     exporterConfig.AppendHost,
		logsProcessors: processors,
		logsExporter:   gcpExporter,
		logsConsumer:   consumer,
	}, nil
}

// createTracesExporter creates a traces exporter based on this config.
func createTracesExporter(ctx context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.TracesExporter, error) {
	exporterConfig := cfg.(*Config)
	exporterConfig.setClientOptions()

	gcpExporter, err := gcpFactory.CreateTracesExporter(ctx, set, exporterConfig.GCPConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create traces exporter: %w", err)
	}

	processors := []component.TracesProcessor{}
	processorConfigs := []config.Processor{
		exporterConfig.BatchConfig,
	}

	processorFactories := []component.ProcessorFactory{
		batchprocessor.NewFactory(),
	}

	processorSettings := component.ProcessorCreateSettings{
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	var consumer consumer.Traces = gcpExporter
	for i, processorConfig := range processorConfigs {
		processorConfig.SetIDName(exporterConfig.ID().String())
		factory := processorFactories[i]
		processor, err := factory.CreateTracesProcessor(ctx, processorSettings, processorConfig, consumer)
		if err != nil {
			return nil, fmt.Errorf("failed to create traces processor %s: %w", processorConfig.ID().String(), err)
		}
		processors = append(processors, processor)
		consumer = processor
	}

	return &exporter{
		appendHost:       exporterConfig.AppendHost,
		tracesProcessors: processors,
		tracesExporter:   gcpExporter,
		tracesConsumer:   consumer,
	}, nil
}
