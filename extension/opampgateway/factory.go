package opampgateway

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

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
		OpAMPServer: &OpAMPServer{
			Endpoint: "0.0.0.0:3001",
			TLS: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func createOpAMPGateway(_ context.Context, cs extension.Settings, cfg component.Config) (extension.Extension, error) {
	t, err := metadata.NewTelemetryBuilder(cs.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("create telemetry builder: %w", err)
	}

	oCfg := cfg.(*Config)
	return newOpAMPGateway(cs.Logger, oCfg, t), nil
}
