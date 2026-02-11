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

package opampgateway

import (
	"context"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/gateway"
	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/extension"
)

// NewFactory creates a new factory for the OpAMP gateway extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		defaultConfig,
		createOpAMPGateway,
		metadata.ExtensionStability,
	)
}

func defaultConfig() component.Config {
	return &Config{
		UpstreamConnections: 1,
		OpAMPServer:         confighttp.NewDefaultServerConfig(),
	}
}

func createOpAMPGateway(_ context.Context, cs extension.Settings, cfg component.Config) (extension.Extension, error) {
	t, err := metadata.NewTelemetryBuilder(cs.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("create telemetry builder: %w", err)
	}

	oCfg := cfg.(*Config)

	settings := gateway.Settings{
		UpstreamOpAMPAddress: oCfg.UpstreamOpAMPAddress,
		SecretKey:            oCfg.SecretKey,
		UpstreamConnections:  oCfg.UpstreamConnections,
		OpAMPServer:          oCfg.OpAMPServer,
	}

	gw := gateway.New(cs.Logger, settings, t)
	return &OpAMPGateway{
		gateway:           gw,
		telemetrySettings: cs.TelemetrySettings,
	}, nil
}
