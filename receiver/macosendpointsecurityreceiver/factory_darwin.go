// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver/internal/metadata"
)

func newFactoryAdapter() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiverDarwin, metadata.LogsStability),
	)
}

// createLogsReceiverDarwin creates a logs receiver based on provided config
func createLogsReceiverDarwin(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	oCfg := cfg.(*Config)

	if err := oCfg.Validate(); err != nil {
		return nil, err
	}

	return newEndpointSecurityReceiver(oCfg, set.Logger, consumer), nil
}
