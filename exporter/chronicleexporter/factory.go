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

package chronicleexporter

import (
	"context"
	"errors"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory creates a new Chronicle exporter factory.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability))
}

const (
	defaultEndpoint                  = "malachiteingestion-pa.googleapis.com"
	defaultBatchLogCountLimitGRPC    = 1000
	defaultBatchRequestSizeLimitGRPC = 1048576
	defaultBatchLogCountLimitHTTP    = 1000
	defaultBatchRequestSizeLimitHTTP = 1048576
)

// createDefaultConfig creates the default configuration for the exporter.
func createDefaultConfig() component.Config {
	return &Config{
		Protocol:                  protocolGRPC,
		TimeoutConfig:             exporterhelper.NewDefaultTimeoutConfig(),
		QueueConfig:               exporterhelper.NewDefaultQueueConfig(),
		BackOffConfig:             configretry.NewDefaultBackOffConfig(),
		OverrideLogType:           true,
		Compression:               noCompression,
		CollectAgentMetrics:       true,
		Endpoint:                  defaultEndpoint,
		BatchLogCountLimitGRPC:    defaultBatchLogCountLimitGRPC,
		BatchRequestSizeLimitGRPC: defaultBatchRequestSizeLimitGRPC,
		BatchLogCountLimitHTTP:    defaultBatchLogCountLimitHTTP,
		BatchRequestSizeLimitHTTP: defaultBatchRequestSizeLimitHTTP,
	}
}

// createLogsExporter creates a new log exporter based on this config.
func createLogsExporter(
	ctx context.Context,
	params exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	chronicleCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	exp, err := newExporter(chronicleCfg, params, params.ID.String())
	if err != nil {
		return nil, err
	}

	pusher := exp.logsDataPusher
	if chronicleCfg.Protocol == protocolHTTPS {
		pusher = exp.logsHTTPDataPusher
	}
	return exporterhelper.NewLogs(
		ctx,
		params,
		chronicleCfg,
		pusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
		exporterhelper.WithTimeout(chronicleCfg.TimeoutConfig),
		exporterhelper.WithQueue(chronicleCfg.QueueConfig),
		exporterhelper.WithRetry(chronicleCfg.BackOffConfig),
		exporterhelper.WithStart(exp.Start),
		exporterhelper.WithShutdown(exp.Shutdown),
	)
}
