// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		makeCfg     func(t *testing.T) *Config
		expectedErr string
	}{
		{
			desc: "valid config with event types",
			makeCfg: func(_ *testing.T) *Config {
				return &Config{
					EventTypes:      []EventType{EventTypeExec, EventTypeFork, EventTypeExit},
					MaxPollInterval: 50 * time.Second,
				}
			},
		},
		{
			desc: "invalid config - no event types",
			makeCfg: func(_ *testing.T) *Config {
				return &Config{
					MaxPollInterval: 30 * time.Second,
				}
			},
			expectedErr: "at least one event_type must be specified",
		},
		{
			desc: "invalid config - invalid event type",
			makeCfg: func(_ *testing.T) *Config {
				return &Config{
					EventTypes: []EventType{EventTypeExec, EventType("invalid_event")},
				}
			},
			expectedErr: "invalid event type",
		},
		{
			desc: "valid config with select paths",
			makeCfg: func(_ *testing.T) *Config {
				return &Config{
					EventTypes: []EventType{EventTypeExec},
					SelectPaths: []string{"/bin/zsh", "/usr/bin"},
				}
			},
		},
		{
			desc: "invalid select path - empty",
			makeCfg: func(_ *testing.T) *Config {
				return &Config{
					EventTypes: []EventType{EventTypeExec},
					SelectPaths: []string{""},
				}
			},
			expectedErr: "select_path cannot be empty",
		},
		{
			desc: "invalid select path - relative path",
			makeCfg: func(_ *testing.T) *Config {
				return &Config{
					EventTypes: []EventType{EventTypeExec},
					SelectPaths: []string{"relative/path"},
				}
			},
			expectedErr: "select_path must be an absolute path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := tc.makeCfg(t)
			err := cfg.Validate()
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	testCases := []struct {
		name            string
		configKey       string
		expectedTypes   []EventType
		expectedPaths   []string
		expectedPoll    time.Duration
	}{
		{
			name:          "basic config",
			configKey:     "basic_config",
			expectedTypes: []EventType{EventTypeExec, EventTypeFork, EventTypeExit},
			expectedPoll:  30 * time.Second,
		},
		{
			name:          "with path filtering",
			configKey:     "with_path_filtering",
			expectedTypes: []EventType{EventTypeExec, EventTypeFork, EventTypeExit, EventTypeOpen, EventTypeClose},
			expectedPaths: []string{"/bin/zsh", "/usr/bin"},
			expectedPoll:  30 * time.Second,
		},
		{
			name:          "minimal config",
			configKey:     "minimal_config",
			expectedTypes: []EventType{EventTypeExec},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load the config from YAML file
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", "test_config.yaml"))
			require.NoError(t, err)

			// Get the specific config section
			sub, err := cm.Sub(tc.configKey)
			require.NoError(t, err)

			// Unmarshal into Config struct
			cfg := &Config{}
			err = sub.Unmarshal(cfg)
			require.NoError(t, err)

			// Verify the config values were parsed correctly
			require.Equal(t, len(tc.expectedTypes), len(cfg.EventTypes), "event_types count mismatch")
			for i, expectedType := range tc.expectedTypes {
				require.Equal(t, expectedType, cfg.EventTypes[i], "event_type mismatch at index %d", i)
			}

			if len(tc.expectedPaths) > 0 {
				require.Equal(t, tc.expectedPaths, cfg.SelectPaths, "select_paths mismatch")
			}

			if tc.expectedPoll > 0 {
				require.Equal(t, tc.expectedPoll, cfg.MaxPollInterval, "max_poll_interval mismatch")
			}

			// Validate the config (should pass for valid configs)
			err = cfg.Validate()
			require.NoError(t, err)
		})
	}
}

func TestEventTypeValidation(t *testing.T) {
	testCases := []struct {
		name      string
		eventType EventType
		expected  bool
	}{
		{
			name:      "valid event type - exec",
			eventType: EventTypeExec,
			expected:  true,
		},
		{
			name:      "valid event type - fork",
			eventType: EventTypeFork,
			expected:  true,
		},
		{
			name:      "invalid event type",
			eventType: EventType("invalid"),
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.eventType.IsValid()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestParseEventType(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    EventType
		expectError bool
	}{
		{
			name:        "valid event type",
			input:       "exec",
			expected:    EventTypeExec,
			expectError: false,
		},
		{
			name:        "invalid event type",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseEventType(tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestValidateSelectPath(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "valid")
	require.NoError(t, os.MkdirAll(validPath, 0o755))

	testCases := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid absolute path",
			path:        validPath,
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
		},
		{
			name:        "relative path",
			path:        "relative/path",
			expectError: true,
		},
		{
			name:        "absolute path that doesn't exist",
			path:        "/nonexistent/path",
			expectError: false, // We don't fail if path doesn't exist, just warn
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSelectPath(tc.path)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
