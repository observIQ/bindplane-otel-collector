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

package restapireceiver // import "github.com/observiq/bindplane-otel-collector/receiver/restapireceiver"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/observiq/bindplane-otel-collector/receiver/restapireceiver/internal/metadata"
)

// NewFactory creates a factory for the REST API receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		URL:      "",
		AuthMode: authModeNone,
		Pagination: PaginationConfig{
			Mode:           paginationModeNone,
			PageLimit:      0,
			ZeroBasedIndex: false,
		},
		MinPollInterval:   10 * time.Second,
		MaxPollInterval:   5 * time.Minute,
		BackoffMultiplier: 2.0,
		ClientConfig: confighttp.ClientConfig{
			Timeout: 10 * time.Second,
		},
	}
}

func createLogsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	cfg := rConf.(*Config)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	rcvr, err := newRESTAPILogsReceiver(params, cfg, consumer)
	if err != nil {
		return nil, fmt.Errorf("unable to create REST API logs receiver: %w", err)
	}
	return rcvr, nil
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := rConf.(*Config)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if cfg.Metrics.NameField == "" {
		return nil, fmt.Errorf("metrics.name_field is required")
	}

	rcvr, err := newRESTAPIMetricsReceiver(params, cfg, consumer)
	if err != nil {
		return nil, fmt.Errorf("unable to create REST API metrics receiver: %w", err)
	}
	return rcvr, nil
}
