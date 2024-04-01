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
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("resourceattributetransposer")

const (
	stability = component.StabilityLevelStable
)

// NewFactory returns a new factory for the resourceattributetransposer processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, stability),
		processor.WithLogs(createLogsProcessor, stability),
	)
}

// createDefaultConfig returns the default config for the resourceattributetransposer processor.
func createDefaultConfig() component.Config {
	return &Config{}
}

// createMetricsProcessor creates the resourceattributetransposer processor.
func createMetricsProcessor(_ context.Context, params processor.CreateSettings, cfg component.Config, nextConsumer consumer.Metrics) (processor.Metrics, error) {
	processorCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("config was not of correct type for the processor: %+v", cfg)
	}

	return newMetricsProcessor(params.Logger, nextConsumer, processorCfg), nil
}

func createLogsProcessor(_ context.Context, params processor.CreateSettings, cfg component.Config, nextConsumer consumer.Logs) (processor.Logs, error) {
	processorCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("config was not of correct type for the processor: %+v", cfg)
	}

	return newLogsProcessor(params.Logger, nextConsumer, processorCfg), nil
}
