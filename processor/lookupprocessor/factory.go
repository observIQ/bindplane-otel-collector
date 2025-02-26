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

package lookupprocessor

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("lookup")

const (
	stability = component.StabilityLevelAlpha
)

var (
	consumerCapabilities = consumer.Capabilities{MutatesData: true}
	errInvalidConfigType = errors.New("config is not of type lookupprocessor.Config")
)

// NewFactory creates a new factory with the default configuration
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, stability),
		processor.WithLogs(createLogsProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
	)
}

// createDefaultConfig creates the default configuration for the processor
func createDefaultConfig() component.Config {
	return &Config{}
}

// createTracesProcessor creates a trace processor
func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	lookupCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}

	processor := newLookupProcessor(lookupCfg, set.Logger)
	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processTraces,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(processor.start),
		processorhelper.WithShutdown(processor.shutdown),
	)
}

// createLogsProcessor creates a log processor
func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	lookupCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}

	processor := newLookupProcessor(lookupCfg, set.Logger)
	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(processor.start),
		processorhelper.WithShutdown(processor.shutdown),
	)
}

// createMetricsProcessor creates a metric processor
func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	lookupCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}

	processor := newLookupProcessor(lookupCfg, set.Logger)
	return processorhelper.NewMetrics(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(processor.start),
		processorhelper.WithShutdown(processor.shutdown),
	)
}
