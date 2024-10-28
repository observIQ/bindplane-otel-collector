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
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("throughputmeasurement")

const (
	stability = component.StabilityLevelAlpha
)

var (
	consumerCapabilities = consumer.Capabilities{MutatesData: false}
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
		Enabled:       true,
		SamplingRatio: 0.5,
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	oCfg := cfg.(*Config)
	tmp, err := createOrGetProcessor(set, oCfg)
	if err != nil {
		return nil, fmt.Errorf("create throughputmeasurementprocessor: %w", err)
	}

	return processorhelper.NewTraces(
		ctx, set, cfg, nextConsumer, tmp.processTraces,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(tmp.start),
		processorhelper.WithShutdown(tmp.shutdown),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)
	tmp, err := createOrGetProcessor(set, oCfg)
	if err != nil {
		return nil, fmt.Errorf("create throughputmeasurementprocessor: %w", err)
	}

	return processorhelper.NewLogs(
		ctx, set, cfg, nextConsumer, tmp.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(tmp.start),
		processorhelper.WithShutdown(tmp.shutdown),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	oCfg := cfg.(*Config)
	tmp, err := createOrGetProcessor(set, oCfg)
	if err != nil {
		return nil, fmt.Errorf("create throughputmeasurementprocessor: %w", err)
	}

	return processorhelper.NewMetrics(
		ctx, set, cfg, nextConsumer, tmp.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(tmp.start),
		processorhelper.WithShutdown(tmp.shutdown),
	)
}

func createOrGetProcessor(set processor.Settings, cfg *Config) (*throughputMeasurementProcessor, error) {
	processorsMux.Lock()
	defer processorsMux.Unlock()

	var tmp *throughputMeasurementProcessor
	if p, ok := processors[set.ID]; ok {
		tmp = p
	} else {
		var err error
		tmp, err = newThroughputMeasurementProcessor(set.Logger, set.MeterProvider, cfg, set.ID)
		if err != nil {
			return nil, err
		}

		processors[set.ID] = tmp
	}

	return tmp, nil
}

func unregisterProcessor(id component.ID) {
	processorsMux.Lock()
	defer processorsMux.Unlock()
	delete(processors, id)
}

// processors is a map of component.ID to an instance of throughput processor.
// It is used so that only one instance of a particular throughput processor exists, even if it's included
// across multiple pipelines/signal types.
var processors = map[component.ID]*throughputMeasurementProcessor{}
var processorsMux = sync.Mutex{}
