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

package resourceattributetransposerprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const (
	typeStr = "resourceattributetransposer"

	stability = component.StabilityLevelStable
)

// NewFactory returns a new factory for the resourceattributetransposer processor.
func NewFactory() component.ProcessorFactory {
	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsProcessor(createMetricsProcessor, stability),
		component.WithLogsProcessor(createLogsProcessor, stability),
	)
}

// createDefaultConfig returns the default config for the resourceattributetransposer processor.
func createDefaultConfig() component.ProcessorConfig {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(component.NewID(typeStr)),
	}
}

// createMetricsProcessor creates the resourceattributetransposer processor.
func createMetricsProcessor(_ context.Context, params component.ProcessorCreateSettings, cfg component.ProcessorConfig, nextConsumer consumer.Metrics) (component.MetricsProcessor, error) {
	processorCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("config was not of correct type for the processor: %+v", cfg)
	}

	return newMetricsProcessor(params.Logger, nextConsumer, processorCfg), nil
}

func createLogsProcessor(_ context.Context, params component.ProcessorCreateSettings, cfg component.ProcessorConfig, nextConsumer consumer.Logs) (component.LogsProcessor, error) {
	processorCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("config was not of correct type for the processor: %+v", cfg)
	}

	return newLogsProcessor(params.Logger, nextConsumer, processorCfg), nil
}
