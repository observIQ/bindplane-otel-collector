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
	"os"

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

const buildDescription = "observIQ's opentelemetry-collector distribution"

// NewSettings returns new settings for the collector with default values.
func NewSettings(configPaths []string, version string, loggingOpts []zap.Option, factories otelcol.Factories) (*otelcol.CollectorSettings, error) {
	buildInfo := component.BuildInfo{
		Command:     os.Args[0],
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
