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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_Success(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				Interface:   "en0",
				Filter:      "tcp port 443",
				SnapLen:     65535,
				Promiscuous: true,
			},
		},
		{
			name: "valid config with minimal fields",
			config: &Config{
				Interface: "eth0",
			},
		},
		{
			name: "valid interface with hyphen",
			config: &Config{
				Interface: "eth-0",
			},
		},
		{
			name: "valid interface with underscore",
			config: &Config{
				Interface: "eth_0",
			},
		},
		{
			name: "valid complex BPF filter",
			config: &Config{
				Interface: "en0",
				Filter:    "tcp port 443 or udp port 53",
			},
		},
		{
			name: "valid filter with parentheses",
			config: &Config{
				Interface: "en0",
				Filter:    "(tcp port 80 or tcp port 443) and not src 192.168.1.1",
			},
		},
		{
			name: "minimum valid snaplen",
			config: &Config{
				Interface: "en0",
				SnapLen:   64,
			},
		},
		{
			name: "maximum valid snaplen",
			config: &Config{
				Interface: "en0",
				SnapLen:   65535,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			require.NoError(t, err)
		})
	}
}

func TestConfig_Validate_InterfaceErrors(t *testing.T) {
	tests := []struct {
		name      string
		iface     string
		wantError string
	}{
		{
			name:      "empty interface",
			iface:     "",
			wantError: "interface must be specified",
		},
		{
			name:      "interface with semicolon",
			iface:     "eth0; rm -rf /",
			wantError: "invalid character",
		},
		{
			name:      "interface with pipe",
			iface:     "eth0|cat /etc/passwd",
			wantError: "invalid character",
		},
		{
			name:      "interface with dollar sign",
			iface:     "eth0$USER",
			wantError: "invalid character",
		},
		{
			name:      "interface with backtick",
			iface:     "eth0`whoami`",
			wantError: "invalid character",
		},
		{
			name:      "interface with newline",
			iface:     "eth0\nwhoami",
			wantError: "invalid character",
		},
		{
			name:      "interface with ampersand",
			iface:     "eth0 && whoami",
			wantError: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Interface: tt.iface,
			}
			err := cfg.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestConfig_Validate_FilterErrors(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		wantError string
	}{
		{
			name:      "filter with semicolon",
			filter:    "tcp port 80; rm -rf /",
			wantError: "invalid character",
		},
		{
			name:      "filter with pipe",
			filter:    "tcp port 80 | cat /etc/passwd",
			wantError: "invalid character",
		},
		{
			name:      "filter with dollar sign",
			filter:    "tcp port 80 $USER",
			wantError: "invalid character",
		},
		{
			name:      "filter with backtick",
			filter:    "tcp port 80 `whoami`",
			wantError: "invalid character",
		},
		{
			name:      "filter with newline",
			filter:    "tcp port 80\nwhoami",
			wantError: "invalid character",
		},
		{
			name:      "filter with double ampersand",
			filter:    "tcp port 80 && whoami",
			wantError: "invalid character",
		},
		{
			name:      "filter with double pipe",
			filter:    "tcp port 80 || whoami",
			wantError: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Interface: "en0",
				Filter:    tt.filter,
			}
			err := cfg.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestConfig_Validate_SnapLenErrors(t *testing.T) {
	tests := []struct {
		name      string
		snaplen   int
		wantError string
	}{
		{
			name:      "negative snaplen",
			snaplen:   -1,
			wantError: "snaplen must be between",
		},
		{
			name:      "snaplen too small",
			snaplen:   63,
			wantError: "snaplen must be between",
		},
		{
			name:      "snaplen too large",
			snaplen:   65536,
			wantError: "snaplen must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Interface: "en0",
				SnapLen:   tt.snaplen,
			}
			err := cfg.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}

