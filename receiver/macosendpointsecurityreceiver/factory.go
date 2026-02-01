// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

// NewFactory creates a factory for the macOS Endpoint Security receiver
func NewFactory() receiver.Factory {
	return newFactoryAdapter()
}

// createDefaultConfig creates a config with default values
func createDefaultConfig() component.Config {
	// Default to common process lifecycle events
	return &Config{
		MaxPollInterval: 30 * time.Second,
		EventTypes: []EventType{
			EventTypeExec,
			EventTypeFork,
			EventTypeExit,
		},
	}
}
