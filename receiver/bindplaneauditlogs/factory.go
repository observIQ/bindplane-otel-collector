package bindplaneauditlogs

import (
	"context"
	"time"

	"github.com/observiq/bindplane-otel-collector/receiver/bindplaneauditlogs/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	// TypeStr is the unique identifier for the BindPlane audit logs receiver
	TypeStr = "bindplaneauditlogs"
)

// NewFactory creates a new factory for the Okta receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

func createLogsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	rCfg := rConf.(*Config)
	return newBindplaneAuditLogsReceiver(rCfg, params.Logger, consumer)
}

func createDefaultConfig() component.Config {
	return &Config{
		PollInterval: 10 * time.Second,
	}
}
