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

package pluginreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const emitterTypeStr = "plugin_output"

// Emitter is a struct used to emit data from an internal pipeline to an external consumer.
// The emitter operates as a singleton exporter within an internal pipeline.
type Emitter struct {
	consumer.Logs
	consumer.Metrics
	consumer.Traces
}

// Start is a no-op that fulfills the component.Component interface
func (e *Emitter) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown is a no-op that fulfills the component.Component interface
func (e *Emitter) Shutdown(_ context.Context) error {
	return nil
}

// Capabilities returns the capabilities of the emitter
func (e *Emitter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

// defaultEmitterConfig returns a default config for the plugin's emitter
func defaultEmitterConfig() config.Exporter {
	componentID := config.NewComponentID(emitterTypeStr)
	defaultConfig := config.NewExporterSettings(componentID)
	return &defaultConfig
}

// createLogEmitterFactory creates a log emitter factory.
// The resulting factory will create an exporter that can emit logs from an internal pipeline to an external consumer.
func createLogEmitterFactory(consumer consumer.Logs) component.ExporterFactory {
	createExporter := func(_ context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.LogsExporter, error) {
		return &Emitter{Logs: consumer}, nil
	}

	return exporterhelper.NewFactory(
		emitterTypeStr,
		defaultEmitterConfig,
		exporterhelper.WithLogs(createExporter),
	)
}

// createLogEmitterFactory creates a metric emitter factory.
// The resulting factory will create an exporter that can emit metrics from an internal pipeline to an external consumer.
func createMetricEmitterFactory(consumer consumer.Metrics) component.ExporterFactory {
	createExporter := func(_ context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.MetricsExporter, error) {
		return &Emitter{Metrics: consumer}, nil
	}

	return exporterhelper.NewFactory(
		emitterTypeStr,
		defaultEmitterConfig,
		exporterhelper.WithMetrics(createExporter),
	)
}

// createLogEmitterFactory creates a trace emitter factory.
// The resulting factory will create an exporter that can emit traces from an internal pipeline to an external consumer.
func createTraceEmitterFactory(consumer consumer.Traces) component.ExporterFactory {
	createExporter := func(_ context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.TracesExporter, error) {
		return &Emitter{Traces: consumer}, nil
	}

	return exporterhelper.NewFactory(
		emitterTypeStr,
		defaultEmitterConfig,
		exporterhelper.WithTraces(createExporter),
	)
}
