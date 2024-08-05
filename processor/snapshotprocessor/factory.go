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

package snapshotprocessor

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var componentType = component.MustNewType("snapshotprocessor")

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
		Enabled: true,
		OpAMP:   defaultOpAMPExtensionID,
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	oCfg := cfg.(*Config)
	sp := createOrGetProcessor(set, oCfg)

	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		sp.processTraces,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(sp.start),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)
	sp := createOrGetProcessor(set, oCfg)

	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		sp.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(sp.start),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	oCfg := cfg.(*Config)
	sp := createOrGetProcessor(set, oCfg)

	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		sp.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(sp.start),
	)
}

func createOrGetProcessor(set processor.Settings, cfg *Config) *snapshotProcessor {
	processorsMux.Lock()
	defer processorsMux.Unlock()

	var sp *snapshotProcessor
	if p, ok := processors[set.ID]; ok {
		sp = p
	} else {
		sp = newSnapshotProcessor(set.Logger, cfg, set.ID)
		processors[set.ID] = sp
	}

	return sp
}

func unregisterProcessor(id component.ID) {
	processorsMux.Lock()
	defer processorsMux.Unlock()
	delete(processors, id)
}

// processors is a map of component.ID to an instance of snapshot processor.
// It is used so that only one instance of a particular snapshot processor exists, even if it's included
// across multiple pipelines/signal types.
var processors = map[component.ID]*snapshotProcessor{}
var processorsMux = sync.Mutex{}
