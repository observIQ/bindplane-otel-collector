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

package throughputmeasurementprocessor

import (
	"context"

	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	typeStr = "throughputmeasurement"

	stability = component.StabilityLevelAlpha
)

var (
	consumerCapabilities = consumer.Capabilities{MutatesData: false}
)

// RegisterMetricViews unregisters old metric views if they exist and registers new ones
func RegisterMetricViews() error {
	views := metricViews()
	view.Unregister(views...)
	return view.Register(views...)
}

// NewFactory creates a new ProcessorFactory with default configuration
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, stability),
		processor.WithLogs(createLogsProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Enabled:       true,
		SamplingRatio: 0.5,
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	oCfg := cfg.(*Config)
	tmp := newThroughputMeasurementProcessor(set.Logger, oCfg, set.ID.String())

	return processorhelper.NewTracesProcessor(ctx, set, cfg, nextConsumer, tmp.processTraces, processorhelper.WithCapabilities(consumerCapabilities))
}

func createLogsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)
	tmp := newThroughputMeasurementProcessor(set.Logger, oCfg, set.ID.String())

	return processorhelper.NewLogsProcessor(ctx, set, cfg, nextConsumer, tmp.processLogs, processorhelper.WithCapabilities(consumerCapabilities))
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	oCfg := cfg.(*Config)
	tmp := newThroughputMeasurementProcessor(set.Logger, oCfg, set.ID.String())

	return processorhelper.NewMetricsProcessor(ctx, set, cfg, nextConsumer, tmp.processMetrics, processorhelper.WithCapabilities(consumerCapabilities))
}
