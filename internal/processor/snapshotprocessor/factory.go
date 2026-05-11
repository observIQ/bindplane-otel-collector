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

const stability = component.StabilityLevelAlpha

var consumerCapabilities = consumer.Capabilities{MutatesData: false}

// NewFactory creates a new ProcessorFactory with default configuration.
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
	return &Config{Enabled: true}
}

// createOrGetProcessor returns the snapshotProcessor instance for the
// given Settings.ID, creating one if it does not yet exist. Sharing a
// single instance across the traces/logs/metrics signals (and across
// multiple pipelines wiring the same processor) ensures Start runs
// once and the OpAMP custom capability is registered exactly once per
// processor ID.
func createOrGetProcessor(set processor.Settings, cfg *Config) *snapshotProcessor {
	processorsMux.Lock()
	defer processorsMux.Unlock()

	if p, ok := processors[set.ID]; ok {
		return p
	}
	p := newSnapshotProcessor(set.Logger, cfg, set.ID)
	processors[set.ID] = p
	return p
}

// unregisterProcessor removes the snapshotProcessor for id from the
// shared map. Called by stop().
func unregisterProcessor(id component.ID) {
	processorsMux.Lock()
	defer processorsMux.Unlock()
	delete(processors, id)
}

// processors holds the live snapshotProcessor instances keyed by
// component ID. Mirrors the contrib v2 pattern so a single instance
// services traces, logs, and metrics for the same processor ID.
var (
	processors    = map[component.ID]*snapshotProcessor{}
	processorsMux = sync.Mutex{}
)

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	sp := createOrGetProcessor(set, cfg.(*Config))
	return processorhelper.NewTraces(
		ctx, set, cfg, nextConsumer,
		sp.processTraces,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(sp.start),
		processorhelper.WithShutdown(sp.stop),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	sp := createOrGetProcessor(set, cfg.(*Config))
	return processorhelper.NewLogs(
		ctx, set, cfg, nextConsumer,
		sp.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(sp.start),
		processorhelper.WithShutdown(sp.stop),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	sp := createOrGetProcessor(set, cfg.(*Config))
	return processorhelper.NewMetrics(
		ctx, set, cfg, nextConsumer,
		sp.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(sp.start),
		processorhelper.WithShutdown(sp.stop),
	)
}
