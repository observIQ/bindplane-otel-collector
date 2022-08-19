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

package pluginreceiver

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const (
	typeStr = "plugin"

	stability = component.StabilityLevelBeta
)

// Config is the configuration of a plugin receiver
type Config struct {
	config.ReceiverSettings `mapstructure:",squash"`
	Path                    string         `mapstructure:"path"`
	Parameters              map[string]any `mapstructure:"parameters"`
}

// createDefaultConfig creates a default config for a plugin receiver
func createDefaultConfig() config.Receiver {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewComponentID(typeStr)),
		Parameters:       make(map[string]any),
	}
}

// NewFactory creates a factory for a plugin receiver
func NewFactory() component.ReceiverFactory {
	return component.NewReceiverFactory(typeStr,
		createDefaultConfig,
		component.WithLogsReceiver(createLogsReceiver, stability),
		component.WithMetricsReceiver(createMetricsReceiver, stability),
		component.WithTracesReceiver(createTracesReceiver, stability),
	)
}

// createLogsReceiver creates a plugin receiver with a logs consumer
func createLogsReceiver(_ context.Context, set component.ReceiverCreateSettings, cfg config.Receiver, consumer consumer.Logs) (component.LogsReceiver, error) {
	emitterFactory := createLogEmitterFactory(consumer)
	return createReceiver(cfg, set, emitterFactory)
}

// createMetricsReceiver creates a plugin receiver with a metrics consumer
func createMetricsReceiver(_ context.Context, set component.ReceiverCreateSettings, cfg config.Receiver, consumer consumer.Metrics) (component.MetricsReceiver, error) {
	emitterFactory := createMetricEmitterFactory(consumer)
	return createReceiver(cfg, set, emitterFactory)
}

// createTracesReceiver creates a plugin receiver with a traces consumer
func createTracesReceiver(_ context.Context, set component.ReceiverCreateSettings, cfg config.Receiver, consumer consumer.Traces) (component.TracesReceiver, error) {
	emitterFactory := createTraceEmitterFactory(consumer)
	return createReceiver(cfg, set, emitterFactory)
}

// createReceiver creates a plugin receiver with the supplied emitter
func createReceiver(cfg config.Receiver, set component.ReceiverCreateSettings, emitterFactory component.ExporterFactory) (*Receiver, error) {
	receiverConfig, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("config is not a plugin receiver config")
	}

	plugin, err := LoadPlugin(receiverConfig.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	if err := plugin.CheckParameters(receiverConfig.Parameters); err != nil {
		return nil, fmt.Errorf("invalid plugin parameter: %w", err)
	}

	renderedCfg, err := plugin.Render(receiverConfig.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to render plugin: %w", err)
	}

	return &Receiver{
		plugin:         plugin,
		renderedCfg:    renderedCfg,
		emitterFactory: emitterFactory,
		logger:         set.Logger,
		createService:  createService,
	}, nil
}
