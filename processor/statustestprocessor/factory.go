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

package statustestprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("statustest_processor")

const stability = component.StabilityLevelDevelopment

// NewFactory creates a new factory for the statustest processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
		processor.WithTraces(createTracesProcessor, stability),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	p := newStatusTestProcessor(cfg.(*Config), set.Logger)
	return processorhelper.NewLogs(
		ctx, set, cfg, nextConsumer,
		p.processLogs,
		processorhelper.WithStart(p.start),
		processorhelper.WithShutdown(p.stop),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	p := newStatusTestProcessor(cfg.(*Config), set.Logger)
	return processorhelper.NewMetrics(
		ctx, set, cfg, nextConsumer,
		p.processMetrics,
		processorhelper.WithStart(p.start),
		processorhelper.WithShutdown(p.stop),
	)
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	p := newStatusTestProcessor(cfg.(*Config), set.Logger)
	return processorhelper.NewTraces(
		ctx, set, cfg, nextConsumer,
		p.processTraces,
		processorhelper.WithStart(p.start),
		processorhelper.WithShutdown(p.stop),
	)
}
