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

package maskprocessor

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	typeStr   = "mask"
	stability = component.StabilityLevelAlpha
)

var (
	consumerCapabilities = consumer.Capabilities{MutatesData: true}
	errInvalidConfigType = errors.New("config is not of type maskprocessor.Config")
	defaultRules         = map[string]string{
		"email":       `\b[a-z0-9._%\+\-—|]+@[a-z0-9.\-—|]+\.[a-z|]{2,6}\b`,
		"ssn":         `\b\d{3}[- ]\d{2}[- ]\d{4}\b`,
		"credit_card": `\b(?:(?:(?:\d{4}[- ]?){3}\d{4}|\d{15,16}))\b`,
		"phone":       `\b((\+|\b)[1l][\-\. ])?\(?\b[\dOlZSB]{3,5}([\-\. ]|\) ?)[\dOlZSB]{3}[\-\. ][\dOlZSB]{4}\b`,
	}
)

// NewFactory creates a new ProcessorFactory with default configuration
func NewFactory() component.ProcessorFactory {
	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithTracesProcessor(createTracesProcessor, stability),
		component.WithLogsProcessor(createLogsProcessor, stability),
		component.WithMetricsProcessor(createMetricsProcessor, stability),
	)
}

// createDefaultConfig creates a default config for the mask processor.
func createDefaultConfig() component.Config {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(component.NewID(typeStr)),
		Rules:             defaultRules,
	}
}

// createTracesProcessor creates a mask processor for traces.
func createTracesProcessor(
	ctx context.Context,
	set component.ProcessorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (component.TracesProcessor, error) {
	maskCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}

	processor := newProcessor(set.Logger, maskCfg)
	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processTraces,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(processor.start))
}

// createLogsProcessor creates a mask processor for logs.
func createLogsProcessor(
	ctx context.Context,
	set component.ProcessorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (component.LogsProcessor, error) {
	maskCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}

	processor := newProcessor(set.Logger, maskCfg)
	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(processor.start))
}

// createMetricsProcessor creates a mask processor for metrics.
func createMetricsProcessor(
	ctx context.Context,
	set component.ProcessorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (component.MetricsProcessor, error) {
	maskCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}

	processor := newProcessor(set.Logger, maskCfg)
	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(processor.start))
}
