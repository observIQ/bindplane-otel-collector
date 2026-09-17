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

package collector

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/confmap/provider/aesprovider"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"

	"go.uber.org/zap"
)

const buildDescription = "Bindplane's distribution of the OpenTelemetry collector"

// buildCommand identifies the distribution. It is deliberately a constant
// rather than os.Args[0]: BuildInfo.Command is how a collector names its
// distribution to the outside world -- it becomes service.name on the
// collector's own telemetry, and is what `--version` and `components` report --
// so it should not vary with the path the binary happens to be invoked by.
// The value matches the agent type Bindplane uses for this distribution
// (model.BindplaneOTelCollectorV2AgentType), so a collector names itself the
// same way in its telemetry as it does to the platform managing it.
const buildCommand = "bindplane-otel-collector"

// NewSettings returns new settings for the collector with default values.
func NewSettings(configPaths []string, version string, loggingOpts []zap.Option, factories otelcol.Factories) (*otelcol.CollectorSettings, error) {
	buildInfo := component.BuildInfo{
		Command:     buildCommand,
		Description: buildDescription,
		Version:     version,
	}

	configProviderSettings := otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: configPaths,
			ProviderFactories: []confmap.ProviderFactory{
				fileprovider.NewFactory(),
				envprovider.NewFactory(),
				yamlprovider.NewFactory(),
				httpsprovider.NewFactory(),
				aesprovider.NewFactory(),
			},
			ConverterFactories: []confmap.ConverterFactory{},
			DefaultScheme:      "env",
		},
	}

	return &otelcol.CollectorSettings{
		Factories:               func() (otelcol.Factories, error) { return factories, nil },
		BuildInfo:               buildInfo,
		LoggingOptions:          loggingOpts,
		ConfigProviderSettings:  configProviderSettings,
		DisableGracefulShutdown: true,
	}, nil
}
