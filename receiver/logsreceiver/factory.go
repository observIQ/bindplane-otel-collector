// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logsreceiver

import (
	"context"

	"github.com/open-telemetry/opentelemetry-log-collection/agent"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/plugin"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/multierr"
)

const typeStr = "stanza"

// NewFactory creates a factory for a Stanza-based receiver
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithLogs(createLogsReceiver),
	)
}

func createDefaultConfig() config.Receiver {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewComponentID(typeStr)),
		Pipeline:         OperatorConfigs{},
		Converter: ConverterConfig{
			MaxFlushCount: DefaultMaxFlushCount,
			FlushInterval: DefaultFlushInterval,
		},
	}
}

func createLogsReceiver(
	_ context.Context,
	params component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Logs,
) (component.LogsReceiver, error) {
	stanzaCfg := cfg.(*Config)

	// TODO agent.Builder should accept generic []map[string]interface{}
	// and handle this internally
	if stanzaCfg.PluginDir != "" {
		errs := plugin.RegisterPlugins(stanzaCfg.PluginDir, operator.DefaultRegistry)
		if len(errs) != 0 {
			return nil, multierr.Combine(errs...)
		}
	}

	pipeline, err := stanzaCfg.decodeOperatorConfigs()
	if err != nil {
		return nil, err
	}

	emitter := NewLogEmitter(
		LogEmitterWithLogger(params.Logger.Sugar()),
		LogEmitterWithFlushInterval(stanzaCfg.Converter.FlushInterval),
		LogEmitterWithMaxBatchSize(stanzaCfg.Converter.MaxFlushCount),
	)

	logAgent, err := agent.NewBuilder(params.Logger.Sugar()).
		WithConfig(&agent.Config{Pipeline: pipeline}).
		WithDefaultOutput(emitter).
		Build()
	if err != nil {
		return nil, err
	}

	pluginIDToConfig := map[string]map[string]interface{}{}
	for _, conf := range stanzaCfg.Pipeline {
		if id, ok := conf["id"]; ok {
			if idStr, ok := id.(string); ok {
				pluginIDToConfig[idStr] = conf
			}
		}
	}

	opts := []ConverterOption{
		WithLogger(params.Logger),
		WithIDToPipelineConfigMap(pluginIDToConfig),
	}
	if stanzaCfg.Converter.WorkerCount > 0 {
		opts = append(opts, WithWorkerCount(stanzaCfg.Converter.WorkerCount))
	}
	converter := NewConverter(opts...)

	return &receiver{
		id:        cfg.ID(),
		agent:     logAgent,
		emitter:   emitter,
		consumer:  nextConsumer,
		logger:    params.Logger,
		converter: converter,
	}, nil
}
