// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows

package windowseventtracereceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestConfig() *Config {
	return &Config{
		SessionName:       "TestSession",
		SessionBufferSize: 64,
		Providers: []Provider{
			{Name: "TestProvider", Level: LevelInformational},
		},
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "providers cannot be empty")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		errContains string
	}{
		{
			name:        "valid name-based provider",
			cfg:         createTestConfig(),
			errContains: "",
		},
		{
			name: "valid GUID provider",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: 64,
				Providers:         []Provider{{Name: "{6E6B9D25-A29C-4B5D-B1D6-CAEF4DB4B26D}"}},
			},
			errContains: "",
		},
		{
			name: "empty session name",
			cfg: &Config{
				SessionName:       "",
				SessionBufferSize: 64,
				Providers:         []Provider{{Name: "TestProvider"}},
			},
			errContains: "session_name cannot be empty",
		},
		{
			name: "empty providers",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: 64,
				Providers:         []Provider{},
			},
			errContains: "providers cannot be empty",
		},
		{
			name: "empty provider name",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: 64,
				Providers:         []Provider{{Name: ""}},
			},
			errContains: "provider name cannot be empty",
		},
		{
			name: "malformed GUID missing closing brace",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: 64,
				Providers:         []Provider{{Name: "{6E6B9D25-A29C-4B5D-B1D6-CAEF4DB4B26D"}},
			},
			errContains: "looks like a GUID but is not valid",
		},
		{
			name: "malformed GUID wrong segment lengths",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: 64,
				Providers:         []Provider{{Name: "{6E6B9D25-A29C-4B5D-B1D6}"}},
			},
			errContains: "looks like a GUID but is not valid",
		},
		{
			name: "malformed GUID non-hex characters",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: 64,
				Providers:         []Provider{{Name: "{XXXXXXXX-A29C-4B5D-B1D6-CAEF4DB4B26D}"}},
			},
			errContains: "looks like a GUID but is not valid",
		},
		{
			name: "zero buffer size",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: 0,
				Providers:         []Provider{{Name: "TestProvider"}},
			},
			errContains: "buffer_size must be greater than 0",
		},
		{
			name: "negative buffer size",
			cfg: &Config{
				SessionName:       "TestSession",
				SessionBufferSize: -1,
				Providers:         []Provider{{Name: "TestProvider"}},
			},
			errContains: "buffer_size must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.errContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}
