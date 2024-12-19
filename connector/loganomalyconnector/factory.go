package loganomalyconnector

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

// NewFactory returns a new factory for the Metrics Generation processor.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		component.MustNewType("loganomaly"),
		createDefaultConfig,
		connector.WithLogsToLogs(createLogsToLogs, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		SampleInterval:   1 * time.Minute,
		MaxWindowAge:     1 * time.Hour,
		ZScoreThreshold:  3.0,
		MADThreshold:     3.5,
		EmergencyMaxSize: 1000,
	}
}

func createLogsToLogs(_ context.Context, set connector.Settings, cfg component.Config, nextConsumer consumer.Logs) (connector.Logs, error) {
	connectorConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("configuration parsing error")
	}

	return newDetector(connectorConfig, set.Logger, nextConsumer), nil
}
