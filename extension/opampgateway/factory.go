package opampgateway

import (
	"context"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/gateway"
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
			Endpoint: "0.0.0.0:0",
		},
	}
}

func createOpAMPGateway(ctx context.Context, cs extension.Settings, cfg component.Config) (extension.Extension, error) {
	t, err := metadata.NewTelemetryBuilder(cs.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("create telemetry builder: %w", err)
	}

	oCfg := cfg.(*Config)

	tlsCfg, err := oCfg.OpAMPServer.TLS.LoadTLSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load TLS config: %w", err)
	}

	settings := gateway.Settings{
		UpstreamOpAMPAddress: oCfg.UpstreamOpAMPAddress,
		SecretKey:            oCfg.SecretKey,
		UpstreamConnections:  oCfg.UpstreamConnections,
		ServerEndpoint:       oCfg.OpAMPServer.Endpoint,
		ServerTLS:            tlsCfg,
	}

	gw := gateway.New(cs.Logger, settings, t)
	return &OpAMPGateway{gateway: gw}, nil
}
