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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "valid config with no auth",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "missing URL",
			config: &Config{
				URL: "",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "url is required",
		},
		{
			name: "invalid auth mode",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthMode("invalid"),
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "invalid auth mode: invalid, must be one of: none, apikey, bearer, basic",
		},
		{
			name: "apikey auth missing header name",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeAPIKey,
					APIKey: APIKeyAuth{
						HeaderName: "",
						Value:      "test-key",
					},
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "auth.apikey.header_name is required when auth.mode is apikey",
		},
		{
			name: "apikey auth missing value",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeAPIKey,
					APIKey: APIKeyAuth{
						HeaderName: "X-API-Key",
						Value:      "",
					},
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "auth.apikey.value is required when auth.mode is apikey",
		},
		{
			name: "valid apikey auth",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeAPIKey,
					APIKey: APIKeyAuth{
						HeaderName: "X-API-Key",
						Value:      "test-key",
					},
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "bearer auth missing token",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode:        AuthModeBearer,
					BearerToken: "",
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "auth.bearer_token is required when auth.mode is bearer",
		},
		{
			name: "valid bearer auth",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode:        AuthModeBearer,
					BearerToken: "test-token",
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "basic auth missing username",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeBasic,
					BasicAuth: BasicAuth{
						Username: "",
						Password: "test-password",
					},
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "auth.basic.username is required when auth.mode is basic",
		},
		{
			name: "basic auth missing password",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeBasic,
					BasicAuth: BasicAuth{
						Username: "test-user",
						Password: "",
					},
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "auth.basic.password is required when auth.mode is basic",
		},
		{
			name: "valid basic auth",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeBasic,
					BasicAuth: BasicAuth{
						Username: "test-user",
						Password: "test-password",
					},
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "invalid pagination mode",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationMode("invalid"),
				},
			},
			expectedErr: "invalid pagination mode: invalid, must be one of: none, offset_limit, page_size",
		},
		{
			name: "offset_limit pagination missing offset field name",
			config: &Config{
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeOffsetLimit,
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
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeOffsetLimit,
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
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeOffsetLimit,
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
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModePageSize,
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
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModePageSize,
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
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModePageSize,
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
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
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
				URL: "https://api.example.com/data",
				Auth: AuthConfig{
					Mode: AuthModeNone,
				},
				Pagination: PaginationConfig{
					Mode: PaginationModeNone,
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

	require.Equal(t, AuthModeNone, cfg.Auth.Mode)
	require.Equal(t, PaginationModeNone, cfg.Pagination.Mode)
	require.Equal(t, 5*time.Minute, cfg.PollInterval)
	require.False(t, cfg.TimeBasedOffset.Enabled)
	require.Equal(t, 0, cfg.Pagination.PageLimit)
	require.False(t, cfg.Pagination.ZeroBasedIndex)
}
