// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/confmap"
)

// Config defines configuration for the macOS Endpoint Security receiver
// Separated into a common file that isn't platform specific so that factory_others.go can reference it
type Config struct {
	// EventTypes is a list of event types to subscribe to
	// Example: ["exec", "fork", "exit", "open", "close"]
	EventTypes []EventType `mapstructure:"event_types"`

	// SelectPaths is a list of program path prefixes for filtering events
	// Events will only be captured for processes matching these paths
	// Example: ["/bin/zsh", "/usr/bin"]
	SelectPaths []string `mapstructure:"select_paths"`

	// MaxPollInterval specifies the maximum interval for health checks/reconnection attempts
	// Note: eslogger streams continuously, so this is mainly for error recovery
	MaxPollInterval time.Duration `mapstructure:"max_poll_interval"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Unmarshal implements confmap.Unmarshaler to convert string slices to EventType slices
func (cfg *Config) Unmarshal(componentParser *confmap.Conf) error {
	if err := componentParser.Unmarshal(cfg); err != nil {
		return err
	}

	// Convert event_types from strings to EventType enums
	eventTypesRaw := componentParser.Get("event_types")
	if eventTypesRaw != nil {
		if eventTypesSlice, ok := eventTypesRaw.([]interface{}); ok {
			cfg.EventTypes = make([]EventType, 0, len(eventTypesSlice))
			for _, et := range eventTypesSlice {
				if etStr, ok := et.(string); ok {
					eventType, err := ParseEventType(etStr)
					if err != nil {
						return fmt.Errorf("invalid event_type %q: %w", etStr, err)
					}
					cfg.EventTypes = append(cfg.EventTypes, eventType)
				} else {
					return fmt.Errorf("event_types must be a list of strings, got %T", et)
				}
			}
		}
	}

	return nil
}
