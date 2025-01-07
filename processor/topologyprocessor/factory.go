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

package topologyprocessor

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var componentType = component.MustNewType("topology")

const (
	stability = component.StabilityLevelAlpha
)

var consumerCapabilities = consumer.Capabilities{MutatesData: false}

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
		Enabled:  false,
		Interval: defaultInterval,
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	oCfg := cfg.(*Config)
	tp, err := createOrGetProcessor(set, oCfg)
	if err != nil {
		return nil, fmt.Errorf("create topologyprocessor: %w", err)
	}

	return processorhelper.NewTraces(
		ctx, set, cfg, nextConsumer, tp.processTraces,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(tp.start),
		processorhelper.WithShutdown(tp.shutdown),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)
	tp, err := createOrGetProcessor(set, oCfg)
	if err != nil {
		return nil, fmt.Errorf("create topologyprocessor: %w", err)
	}

	return processorhelper.NewLogs(
		ctx, set, cfg, nextConsumer, tp.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(tp.start),
		processorhelper.WithShutdown(tp.shutdown),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	oCfg := cfg.(*Config)
	tp, err := createOrGetProcessor(set, oCfg)
	if err != nil {
		return nil, fmt.Errorf("create topologyprocessor: %w", err)
	}

	return processorhelper.NewMetrics(
		ctx, set, cfg, nextConsumer, tp.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(tp.start),
		processorhelper.WithShutdown(tp.shutdown),
	)
}

func createOrGetProcessor(set processor.Settings, cfg *Config) (*topologyProcessor, error) {
	processorsMux.Lock()
	defer processorsMux.Unlock()

	var tp *topologyProcessor
	if p, ok := processors[set.ID]; ok {
		tp = p
	} else {
		var err error
		tp, err = newTopologyProcessor(set.Logger, cfg, set.ID)
		if err != nil {
			return nil, err
		}

		processors[set.ID] = tp
	}

	return tp, nil
}

func unregisterProcessor(id component.ID) {
	processorsMux.Lock()
	defer processorsMux.Unlock()
	delete(processors, id)
}

// processors is a map of component.ID to an instance of topology processor.
// It is used so that only one instance of a particular topology processor exists, even if it's included
// across multiple pipelines/signal types.
var processors = map[component.ID]*topologyProcessor{}
var processorsMux = sync.Mutex{}
