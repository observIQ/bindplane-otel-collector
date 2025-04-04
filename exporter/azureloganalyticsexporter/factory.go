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

package azureloganalyticsexporter // import "github.com/observiq/bindplane-otel-collector/exporter/azureloganalyticsexporter"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Define type to replace metadata.Type temporarily
var typeStr = component.MustNewType("azureloganalytics")

// NewFactory creates a factory for Azure Log Analytics Exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsExporter(ctx context.Context, params exporter.Settings, config component.Config) (exporter.Logs, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("not a Azure Log Analytics config")
	}

	exp, err := newExporter(cfg, params)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogs(ctx,
		params,
		cfg,
		exp.logsDataPusher,
		exporterhelper.WithCapabilities(exp.Capabilities()),
	)
}
