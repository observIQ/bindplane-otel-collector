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

package routereceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

const (
	// typeStr is the value of the "type" key in configuration.
	typeStr = "route"

	// stability is the current state of the receiver.
	stability = component.StabilityLevelAlpha
)

// NewFactory creates a new factory for the receiver.
func NewFactory() component.ReceiverFactory {
	return component.NewReceiverFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsReceiver(createMetricsReceiver, stability),
		component.WithLogsReceiver(createLogsReceiver, stability),
		component.WithTracesReceiver(createTracesReceiver, stability),
	)
}

// createMetricsReceiver creates a metric receiver.
func createMetricsReceiver(_ context.Context, _ component.ReceiverCreateSettings, cfg component.ReceiverConfig, consumer consumer.Metrics) (component.MetricsReceiver, error) {
	receiver := createOrGetRoute(cfg.ID().Name())
	receiver.registerMetricConsumer(consumer)
	return receiver, nil
}

// createLogsReceiver creates a log receiver.
func createLogsReceiver(_ context.Context, _ component.ReceiverCreateSettings, cfg component.ReceiverConfig, consumer consumer.Logs) (component.LogsReceiver, error) {
	receiver := createOrGetRoute(cfg.ID().Name())
	receiver.registerLogConsumer(consumer)
	return receiver, nil
}

// createTracesReceiver creates a trace receiver.
func createTracesReceiver(_ context.Context, _ component.ReceiverCreateSettings, cfg component.ReceiverConfig, consumer consumer.Traces) (component.TracesReceiver, error) {
	receiver := createOrGetRoute(cfg.ID().Name())
	receiver.registerTraceConsumer(consumer)
	return receiver, nil
}
