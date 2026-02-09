package opampgateway

import (
	"context"

	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/gateway"
	"go.opentelemetry.io/collector/component"
)

// OpAMPGateway is the OTel extension wrapper around the internal gateway implementation.
type OpAMPGateway struct {
	gateway *gateway.Gateway
}

func (o *OpAMPGateway) Start(ctx context.Context, host component.Host) error {
	return o.gateway.Start(ctx)
}

func (o *OpAMPGateway) Shutdown(ctx context.Context) error {
	return o.gateway.Shutdown(ctx)
}
