// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
)

// NewFactory creates a factory for macOS Unified Logging receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

// createDefaultConfig creates a config with default values
func createDefaultConfig() component.Config {
	return &Config{
		TraceV3Paths: []string{
			"/var/db/diagnostics/tracev3/Persist/*",
			"/var/db/diagnostics/tracev3/Special/*",
			"/var/db/diagnostics/tracev3/HighVolume/*",
			"/var/db/diagnostics/tracev3/SignPost/*",
		},
		TimesyncPaths: []string{"/var/db/diagnostics/timesync/*"},
		BaseConfig: adapter.BaseConfig{
			Operators: []operator.Config{},
		},
		Config: *fileconsumer.NewConfig(),
	}
}

// createLogsReceiver creates a logs receiver based on provided config
func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	oCfg := cfg.(*Config)

	if err := oCfg.Validate(); err != nil {
		return nil, err
	}

	// Create a custom receiver that handles traceV3 files with encoding extensions
	// The encoding extension will be loaded in the Start method when component.Host is available
	return newMacOSUnifiedLogReceiver(oCfg, set, consumer)
}
