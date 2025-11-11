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

package restapireceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"

	"github.com/observiq/bindplane-otel-collector/receiver/restapireceiver/internal/metadata"
)

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "valid config with apikey auth",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "missing URL",
			config: &Config{
				URL:                  "",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "url is required",
		},
		{
			name: "invalid auth mode",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: string(AuthMode("invalid")),
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "invalid auth mode: invalid, must be one of: apikey, bearer, basic",
		},
		{
			name: "apikey auth missing header name",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "apikey_header_name is required when auth_mode is apikey",
		},
		{
			name: "apikey auth missing value",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "apikey_value is required when auth_mode is apikey",
		},
		{
			name: "valid apikey auth",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "bearer auth missing token",
			config: &Config{
				URL:             "https://api.example.com/data",
				AuthMode:        string(authModeBearer),
				AuthBearerToken: "",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "bearer_token is required when auth_mode is bearer",
		},
		{
			name: "valid bearer auth",
			config: &Config{
				URL:             "https://api.example.com/data",
				AuthMode:        string(authModeBearer),
				AuthBearerToken: "test-token",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "basic auth missing username",
			config: &Config{
				URL:               "https://api.example.com/data",
				AuthMode:          string(authModeBasic),
				AuthBasicUsername: "",
				AuthBasicPassword: "test-password",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "basic_username is required when auth_mode is basic",
		},
		{
			name: "basic auth missing password",
			config: &Config{
				URL:               "https://api.example.com/data",
				AuthMode:          string(authModeBasic),
				AuthBasicUsername: "test-user",
				AuthBasicPassword: "",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "basic_password is required when auth_mode is basic",
		},
		{
			name: "valid basic auth",
			config: &Config{
				URL:               "https://api.example.com/data",
				AuthMode:          string(authModeBasic),
				AuthBasicUsername: "test-user",
				AuthBasicPassword: "test-password",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "invalid pagination mode",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: PaginationMode("invalid"),
				},
			},
			expectedErr: "invalid pagination mode: invalid, must be one of: none, offset_limit, page_size",
		},
		{
			name: "offset_limit pagination missing offset field name",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeOffsetLimit,
					OffsetLimit: OffsetLimitPagination{
						OffsetFieldName: "",
						LimitFieldName:  "limit",
					},
				},
			},
			expectedErr: "pagination.offset_limit.offset_field_name is required when pagination.mode is offset_limit",
		},
		{
			name: "offset_limit pagination missing limit field name",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeOffsetLimit,
					OffsetLimit: OffsetLimitPagination{
						OffsetFieldName: "offset",
						LimitFieldName:  "",
					},
				},
			},
			expectedErr: "pagination.offset_limit.limit_field_name is required when pagination.mode is offset_limit",
		},
		{
			name: "valid offset_limit pagination",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeOffsetLimit,
					OffsetLimit: OffsetLimitPagination{
						OffsetFieldName: "offset",
						LimitFieldName:  "limit",
						StartingOffset:  0,
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "page_size pagination missing page num field name",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModePageSize,
					PageSize: PageSizePagination{
						PageNumFieldName:  "",
						PageSizeFieldName: "page_size",
					},
				},
			},
			expectedErr: "pagination.page_size.page_num_field_name is required when pagination.mode is page_size",
		},
		{
			name: "page_size pagination missing page size field name",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModePageSize,
					PageSize: PageSizePagination{
						PageNumFieldName:  "page",
						PageSizeFieldName: "",
					},
				},
			},
			expectedErr: "pagination.page_size.page_size_field_name is required when pagination.mode is page_size",
		},
		{
			name: "valid page_size pagination",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModePageSize,
					PageSize: PageSizePagination{
						PageNumFieldName:  "page",
						PageSizeFieldName: "page_size",
						StartingPage:      1,
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "time_based_offset enabled missing param name",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
				TimeBasedOffset: TimeBasedOffsetConfig{
					Enabled:         true,
					ParamName:       "",
					OffsetTimestamp: time.Now(),
				},
			},
			expectedErr: "time_based_offset.param_name is required when time_based_offset.enabled is true",
		},
		{
			name: "valid time_based_offset",
			config: &Config{
				URL:                  "https://api.example.com/data",
				AuthMode:             string(authModeAPIKey),
				AuthAPIKeyHeaderName: "X-API-Key",
				AuthAPIKeyValue:      "test-key",
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
				TimeBasedOffset: TimeBasedOffsetConfig{
					Enabled:         true,
					ParamName:       "since",
					OffsetTimestamp: time.Now(),
				},
			},
			expectedErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := xconfmap.Validate(tc.config)
			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)

	require.Equal(t, string(authModeAPIKey), cfg.AuthMode)
	require.Equal(t, paginationModeNone, cfg.Pagination.Mode)
	require.Equal(t, 5*time.Minute, cfg.PollInterval)
	require.False(t, cfg.TimeBasedOffset.Enabled)
	require.Equal(t, 0, cfg.Pagination.PageLimit)
	require.False(t, cfg.Pagination.ZeroBasedIndex)
}

func TestLoadConfigFromYAML(t *testing.T) {
	// Load the YAML config file
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "test-config.yaml"))
	require.NoError(t, err)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	// Get the receivers.restapi section
	receiversMap, err := cm.Sub("receivers")
	require.NoError(t, err)

	sub, err := receiversMap.Sub(component.NewID(metadata.Type).String())
	require.NoError(t, err)

	// Unmarshal the config
	err = sub.Unmarshal(cfg)
	require.NoError(t, err)

	// Validate the config
	restapiCfg := cfg.(*Config)
	err = xconfmap.Validate(restapiCfg)
	require.NoError(t, err)

	// Verify the config values were parsed correctly
	require.Equal(t, "https://api.example.com/data", restapiCfg.URL)
	require.Equal(t, "data", restapiCfg.ResponseField)
	require.Equal(t, 5*time.Minute, restapiCfg.PollInterval)
	require.Equal(t, string(authModeAPIKey), restapiCfg.AuthMode)
	require.Equal(t, "X-API-Key", restapiCfg.AuthAPIKeyHeaderName)
	require.Equal(t, "test-key", restapiCfg.AuthAPIKeyValue)
}
