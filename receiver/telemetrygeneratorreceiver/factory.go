// Copyright observIQ, Inc.
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

package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"context"
	"errors"

	"github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// errImproperCfgType error for when an invalid config type is passed to receiver creation funcs
var errImproperCfgType = errors.New("improper config type")

// NewFactory creates a new receiver factory
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
	)
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() component.Config {
	return &Config{
		PayloadsPerSecond: 1,
	}
}

// createMetricsReceiver creates a metrics receiver
func createMetricsReceiver(ctx context.Context, params receiver.CreateSettings, conf component.Config, nextConsumer consumer.Metrics) (receiver.Metrics, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errImproperCfgType
	}

	return newMetricsReceiver(ctx, params.Logger, cfg, nextConsumer), nil
}

// createLogsReceiver creates a logs receiver
func createLogsReceiver(ctx context.Context, params receiver.CreateSettings, conf component.Config, nextConsumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errImproperCfgType
	}

	return newLogsReceiver(ctx, params.Logger, cfg, nextConsumer), nil
}

// createTracesReceiver creates a traces receiver
func createTracesReceiver(ctx context.Context, params receiver.CreateSettings, conf component.Config, nextConsumer consumer.Traces) (receiver.Traces, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errImproperCfgType
	}

	return newTracesReceiver(ctx, params.Logger, cfg, nextConsumer), nil
}
