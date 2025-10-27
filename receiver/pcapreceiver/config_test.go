// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pcapreceiver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				Interface:   "eth0",
				SnapLength:  1500,
				BPFFilter:   "tcp port 80",
				Promiscuous: true,
			},
			expectedErr: "",
		},
		{
			name: "valid config with defaults",
			config: &Config{
				Interface:   "",
				SnapLength:  65535,
				BPFFilter:   "",
				Promiscuous: true,
			},
			expectedErr: "",
		},
		{
			name: "valid config with empty BPF filter",
			config: &Config{
				Interface:   "lo",
				SnapLength:  65535,
				BPFFilter:   "",
				Promiscuous: false,
			},
			expectedErr: "",
		},
		{
			name: "invalid: negative snap length",
			config: &Config{
				Interface:   "eth0",
				SnapLength:  -1,
				BPFFilter:   "",
				Promiscuous: true,
			},
			expectedErr: "snap_length must be greater than 0",
		},
		{
			name: "invalid: zero snap length",
			config: &Config{
				Interface:   "eth0",
				SnapLength:  0,
				BPFFilter:   "",
				Promiscuous: true,
			},
			expectedErr: "snap_length must be greater than 0",
		},
		{
			name: "invalid: malformed BPF filter - unclosed parenthesis",
			config: &Config{
				Interface:   "eth0",
				SnapLength:  65535,
				BPFFilter:   "tcp port 80 and (host",
				Promiscuous: true,
			},
			expectedErr: "invalid bpf_filter",
		},
		{
			name: "invalid: malformed BPF filter - invalid syntax",
			config: &Config{
				Interface:   "eth0",
				SnapLength:  65535,
				BPFFilter:   "invalid syntax here!!!",
				Promiscuous: true,
			},
			expectedErr: "invalid bpf_filter",
		},
		{
			name: "valid: complex BPF filter",
			config: &Config{
				Interface:   "eth0",
				SnapLength:  65535,
				BPFFilter:   "tcp port 80 or tcp port 443",
				Promiscuous: true,
			},
			expectedErr: "",
		},
		{
			name: "valid: BPF filter with host",
			config: &Config{
				Interface:   "eth0",
				SnapLength:  65535,
				BPFFilter:   "host 192.168.1.1",
				Promiscuous: true,
			},
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectedErr == "" {
				require.NoError(t, err, "expected validation to pass")
			} else {
				require.Error(t, err, "expected validation to fail")
				require.Contains(t, err.Error(), tt.expectedErr, "error message should contain expected text")
			}
		})
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	// Load the config from YAML file
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "test_config.yaml"))
	require.NoError(t, err)

	t.Run("basic config", func(t *testing.T) {
		sub, err := cm.Sub("basic")
		require.NoError(t, err)

		cfg := &Config{}
		err = sub.Unmarshal(cfg)
		require.NoError(t, err)

		require.Equal(t, "eth0", cfg.Interface)
		require.Equal(t, int32(65535), cfg.SnapLength)
		require.Equal(t, true, cfg.Promiscuous)
		require.Equal(t, "", cfg.BPFFilter)

		// Validate should pass
		require.NoError(t, cfg.Validate())
	})

	t.Run("config with filter", func(t *testing.T) {
		sub, err := cm.Sub("with_filter")
		require.NoError(t, err)

		cfg := &Config{}
		err = sub.Unmarshal(cfg)
		require.NoError(t, err)

		require.Equal(t, "lo", cfg.Interface)
		require.Equal(t, int32(1500), cfg.SnapLength)
		require.Equal(t, "tcp port 80", cfg.BPFFilter)

		// Validate should pass
		require.NoError(t, cfg.Validate())
	})

	t.Run("auto-detect config", func(t *testing.T) {
		sub, err := cm.Sub("auto_detect")
		require.NoError(t, err)

		cfg := &Config{}
		err = sub.Unmarshal(cfg)
		require.NoError(t, err)

		require.Equal(t, "", cfg.Interface) // Auto-detect
		require.Equal(t, int32(65535), cfg.SnapLength)
		require.Equal(t, true, cfg.Promiscuous)

		// Validate should pass
		require.NoError(t, cfg.Validate())
	})
}
