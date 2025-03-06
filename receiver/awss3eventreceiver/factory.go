// Copyright observIQ, Inc.
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

package awss3eventreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// errImproperCfgType error for when an invalid config type is passed to receiver creation funcs
var errImproperCfgType = errors.New("improper config type")

// NewFactory creates a new receiver factory
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() component.Config {
	return &Config{
		PollInterval:      20 * time.Second,
		VisibilityTimeout: 300 * time.Second,
		Workers:           5,
		MaxLogSize:        1024 * 1024,
	}
}

// createLogsReceiver creates a logs receiver
func createLogsReceiver(_ context.Context, params receiver.Settings, conf component.Config, con consumer.Logs) (receiver.Logs, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errImproperCfgType
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return newLogsReceiver(params.ID, params.TelemetrySettings, cfg, con)
}
