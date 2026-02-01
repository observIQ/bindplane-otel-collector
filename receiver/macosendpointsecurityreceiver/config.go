// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Validate checks the Config is valid
func (cfg *Config) Validate() error {
	// Validate that at least one event type is specified
	if len(cfg.EventTypes) == 0 {
		return errors.New("at least one event_type must be specified")
	}

	// Validate each event type
	for _, eventType := range cfg.EventTypes {
		if !eventType.IsValid() {
			return fmt.Errorf("invalid event type: %s", eventType)
		}
	}

	// Validate select paths if specified
	for _, path := range cfg.SelectPaths {
		if err := validateSelectPath(path); err != nil {
			return fmt.Errorf("invalid select_path %q: %w", path, err)
		}
	}

	return nil
}

// validateSelectPath checks if a select path is valid
func validateSelectPath(path string) error {
	if path == "" {
		return errors.New("select_path cannot be empty")
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Check if it's an absolute path
	if !filepath.IsAbs(cleaned) {
		return errors.New("select_path must be an absolute path")
	}

	// Check if the path exists (optional - just warn if it doesn't)
	// We don't fail validation if the path doesn't exist, as it might be valid
	// but the process might not be running yet
	if _, err := os.Stat(cleaned); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot access select_path: %w", err)
	}

	return nil
}
