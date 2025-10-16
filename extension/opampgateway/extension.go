package opampgateway

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type OpAMPGateway struct {
	logger *zap.Logger
	cfg    *Config
}

func newOpAMPGateway(logger *zap.Logger, cfg *Config) *OpAMPGateway {
	return &OpAMPGateway{
		logger: logger,
		cfg:    cfg,
	}
}

func (o *OpAMPGateway) Start(_ context.Context, host component.Host) error {
	return nil
}

func (o *OpAMPGateway) Shutdown(ctx context.Context) error {
	return nil
}
