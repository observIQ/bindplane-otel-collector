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

package statustestreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("statustest_receiver")

const stability = component.StabilityLevelDevelopment

// NewFactory creates a new factory for the statustest receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		componentType,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, stability),
		receiver.WithMetrics(createMetricsReceiver, stability),
		receiver.WithTraces(createTracesReceiver, stability),
	)
}

func createLogsReceiver(_ context.Context, set receiver.Settings, cfg component.Config, _ consumer.Logs) (receiver.Logs, error) {
	rCfg := cfg.(*Config)
	return newStatusTestReceiver(rCfg, set.Logger), nil
}

func createMetricsReceiver(_ context.Context, set receiver.Settings, cfg component.Config, _ consumer.Metrics) (receiver.Metrics, error) {
	rCfg := cfg.(*Config)
	return newStatusTestReceiver(rCfg, set.Logger), nil
}

func createTracesReceiver(_ context.Context, set receiver.Settings, cfg component.Config, _ consumer.Traces) (receiver.Traces, error) {
	rCfg := cfg.(*Config)
	return newStatusTestReceiver(rCfg, set.Logger), nil
}
