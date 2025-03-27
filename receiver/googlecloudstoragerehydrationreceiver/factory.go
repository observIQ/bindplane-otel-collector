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

package googlecloudstoragerehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver"

import (
	"context"
	"errors"

	"github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// errImproperCfgType error for when an invalid config type is passed to receiver creation funcs
var errImproperCfgType = errors.New("improper config type")

const (
	defaultBatchSize = 30
)

// NewFactory creates a new factory for the Google Cloud Storage rehydration receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() component.Config {
	return &Config{
		DeleteOnRead: false,
		BatchSize:    defaultBatchSize,
	}
}

// createMetricsReceiver creates a metrics receiver
func createMetricsReceiver(_ context.Context, params receiver.Settings, conf component.Config, con consumer.Metrics) (receiver.Metrics, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errImproperCfgType
	}

	return newMetricsReceiver(params.ID, params.Logger, cfg, con)
}

// createLogsReceiver creates a logs receiver
func createLogsReceiver(_ context.Context, params receiver.Settings, conf component.Config, con consumer.Logs) (receiver.Logs, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errImproperCfgType
	}

	return newLogsReceiver(params.ID, params.Logger, cfg, con)
}

// createTracesReceiver creates a traces receiver
func createTracesReceiver(_ context.Context, params receiver.Settings, conf component.Config, con consumer.Traces) (receiver.Traces, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errImproperCfgType
	}

	return newTracesReceiver(params.ID, params.Logger, cfg, con)
}
