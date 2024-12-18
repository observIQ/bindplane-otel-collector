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

package splunksearchapireceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr = component.MustNewType("splunksearchapi")
)

func createDefaultConfig() component.Config {
	return &Config{
		ClientConfig:    confighttp.NewDefaultClientConfig(),
		JobPollInterval: 5 * time.Second,
	}
}

func createLogsReceiver(_ context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	ssapirConfig := cfg.(*Config)
	ssapir := &splunksearchapireceiver{
		logger:           params.Logger,
		logsConsumer:     consumer,
		config:           ssapirConfig,
		id:               params.ID,
		settings:         params.TelemetrySettings,
		checkpointRecord: &EventRecord{},
	}
	return ssapir, nil
}

// NewFactory creates a factory for Splunk Search API receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(typeStr, createDefaultConfig, receiver.WithLogs(createLogsReceiver, component.StabilityLevelAlpha))
}
