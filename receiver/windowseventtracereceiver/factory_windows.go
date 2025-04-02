//go:build windows

package windowseventtracereceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

func createLogsReceiver(ctx context.Context, params receiver.Settings, cfg component.Config, nextConsumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("not a valid microsoft event trace config")
	}

	lr, err := newLogsReceiver(cfg.(*Config), nextConsumer, params.Logger)
	if err != nil {
		return nil, err
	}
	return lr, nil
}
