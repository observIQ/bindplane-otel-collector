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

package removeemptyvaluesprocessor

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("removeemptyvalues")

const (
	stability = component.StabilityLevelAlpha
)

var (
	errInvalidConfigType = errors.New("config is not of type removeemptyvaluesprocessor.Config")
	consumerCapabilities = consumer.Capabilities{MutatesData: true}
)

// NewFactory creates a new ProcessorFactory with default configuration
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, stability),
		processor.WithLogs(createLogsProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		RemoveNulls:       true,
		RemoveEmptyLists:  false,
		RemoveEmptyMaps:   false,
		EmptyStringValues: []string{},
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	oCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}
	evp := newEmptyValueProcessor(set.Logger, *oCfg)

	return processorhelper.NewTracesProcessor(ctx, set, cfg, nextConsumer, evp.processTraces, processorhelper.WithCapabilities(consumerCapabilities))
}

func createLogsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}
	evp := newEmptyValueProcessor(set.Logger, *oCfg)

	return processorhelper.NewLogsProcessor(ctx, set, cfg, nextConsumer, evp.processLogs, processorhelper.WithCapabilities(consumerCapabilities))
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	oCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}
	evp := newEmptyValueProcessor(set.Logger, *oCfg)

	return processorhelper.NewMetricsProcessor(ctx, set, cfg, nextConsumer, evp.processMetrics, processorhelper.WithCapabilities(consumerCapabilities))
}
