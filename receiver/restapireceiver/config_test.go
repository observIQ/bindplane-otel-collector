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
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "missing URL",
			config: &Config{
				URL:      "",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "url is required",
		},
		{
			name: "valid config with no auth",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeNone,
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "invalid auth mode",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: AuthMode("invalid"),
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "invalid auth mode: invalid, must be one of: none, apikey, bearer, basic, oauth2, akamai_edgegrid",
		},
		{
			name: "apikey auth missing header name",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "header_name is required when auth_mode is apikey",
		},
		{
			name: "apikey auth missing value",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "value is required when auth_mode is apikey",
		},
		{
			name: "valid apikey auth",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "bearer auth missing token",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeBearer,
				BearerConfig: BearerConfig{
					Token: "",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "token is required when auth_mode is bearer",
		},
		{
			name: "valid bearer auth",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeBearer,
				BearerConfig: BearerConfig{
					Token: "test-token",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "basic auth missing username",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeBasic,
				BasicConfig: BasicConfig{
					Username: "",
					Password: "test-password",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "username is required when auth_mode is basic",
		},
		{
			name: "basic auth missing password",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeBasic,
				BasicConfig: BasicConfig{
					Username: "test-user",
					Password: "",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "password is required when auth_mode is basic",
		},
		{
			name: "valid basic auth",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeBasic,
				BasicConfig: BasicConfig{
					Username: "test-user",
					Password: "test-password",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "oauth2 auth missing client_id",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeOAuth2,
				OAuth2Config: OAuth2Config{
					ClientID:     "",
					ClientSecret: "test-client-secret",
					TokenURL:     "https://oauth.example.com/token",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "client_id is required when auth_mode is oauth2",
		},
		{
			name: "oauth2 auth missing client_secret",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeOAuth2,
				OAuth2Config: OAuth2Config{
					ClientID:     "test-client-id",
					ClientSecret: "",
					TokenURL:     "https://oauth.example.com/token",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "client_secret is required when auth_mode is oauth2",
		},
		{
			name: "oauth2 auth missing token_url",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeOAuth2,
				OAuth2Config: OAuth2Config{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					TokenURL:     "",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "token_url is required when auth_mode is oauth2",
		},
		{
			name: "valid oauth2 auth",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeOAuth2,
				OAuth2Config: OAuth2Config{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					TokenURL:     "https://oauth.example.com/token",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "valid oauth2 auth with scopes",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeOAuth2,
				OAuth2Config: OAuth2Config{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					TokenURL:     "https://oauth.example.com/token",
					Scopes:       []string{"read", "write"},
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "valid oauth2 auth with endpoint params",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeOAuth2,
				OAuth2Config: OAuth2Config{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					TokenURL:     "https://oauth.example.com/token",
					EndpointParams: map[string]string{
						"audience": "https://api.example.com",
					},
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "akamai edgegrid auth missing access token",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAkamaiEdgeGrid,
				AkamaiEdgeGridConfig: AkamaiEdgeGridConfig{
					AccessToken:  "",
					ClientToken:  "test-client-token",
					ClientSecret: "test-client-secret",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "access_token is required when auth_mode is akamai_edgegrid",
		},
		{
			name: "akamai edgegrid auth missing client token",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAkamaiEdgeGrid,
				AkamaiEdgeGridConfig: AkamaiEdgeGridConfig{
					AccessToken:  "test-access-token",
					ClientSecret: "test-client-secret",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "client_token is required when auth_mode is akamai_edgegrid",
		},
		{
			name: "akamai edgegrid auth missing client secret",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAkamaiEdgeGrid,
				AkamaiEdgeGridConfig: AkamaiEdgeGridConfig{
					AccessToken: "test-access-token",
					ClientToken: "test-client-token",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "client_secret is required when auth_mode is akamai_edgegrid",
		},
		{
			name: "valid akamai edgegrid auth",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAkamaiEdgeGrid,
				AkamaiEdgeGridConfig: AkamaiEdgeGridConfig{
					AccessToken:  "test-access-token",
					ClientToken:  "test-client-token",
					ClientSecret: "test-client-secret",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
			},
			expectedErr: "",
		},
		{
			name: "invalid pagination mode",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: PaginationMode("invalid"),
				},
			},
			expectedErr: "invalid pagination mode: invalid, must be one of: none, offset_limit, page_size, timestamp",
		},
		{
			name: "offset_limit pagination missing offset field name",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeOffsetLimit,
					OffsetLimit: OffsetLimitPagination{
						OffsetFieldName: "",
						LimitFieldName:  "limit",
					},
				},
			},
			expectedErr: "offset_field_name is required when pagination.mode is offset_limit",
		},
		{
			name: "offset_limit pagination missing limit field name",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeOffsetLimit,
					OffsetLimit: OffsetLimitPagination{
						OffsetFieldName: "offset",
						LimitFieldName:  "",
					},
				},
			},
			expectedErr: "limit_field_name is required when pagination.mode is offset_limit",
		},
		{
			name: "valid offset_limit pagination",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
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
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModePageSize,
					PageSize: PageSizePagination{
						PageNumFieldName:  "",
						PageSizeFieldName: "page_size",
					},
				},
			},
			expectedErr: "page_num_field_name is required when pagination.mode is page_size",
		},
		{
			name: "page_size pagination missing page size field name",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModePageSize,
					PageSize: PageSizePagination{
						PageNumFieldName:  "page",
						PageSizeFieldName: "",
					},
				},
			},
			expectedErr: "page_size_field_name is required when pagination.mode is page_size",
		},
		{
			name: "valid page_size pagination",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
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
			name: "timestamp pagination missing param name",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeTimestamp,
					Timestamp: TimestampPagination{
						ParamName:          "",
						TimestampFieldName: "ts",
						PageSizeFieldName:  "perPage",
						PageSize:           100,
					},
				},
			},
			expectedErr: "param_name is required when pagination.mode is timestamp",
		},
		{
			name: "timestamp pagination missing timestamp field name",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeTimestamp,
					Timestamp: TimestampPagination{
						ParamName:          "t0",
						TimestampFieldName: "",
						PageSizeFieldName:  "perPage",
						PageSize:           100,
					},
				},
			},
			expectedErr: "timestamp_field_name is required when pagination.mode is timestamp",
		},
		{
			name: "valid timestamp pagination",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeTimestamp,
					Timestamp: TimestampPagination{
						ParamName:          "t0",
						TimestampFieldName: "ts",
						PageSizeFieldName:  "perPage",
						PageSize:           200,
						InitialTimestamp:   time.Now().Format(time.RFC3339),
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "valid timestamp pagination with custom format",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeTimestamp,
					Timestamp: TimestampPagination{
						ParamName:          "min-date",
						TimestampFieldName: "timestamp",
						TimestampFormat:    "2006-01-02 15:04:05",
						InitialTimestamp:   "2025-01-01 00:00:00",
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "invalid initial_timestamp format",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeTimestamp,
					Timestamp: TimestampPagination{
						ParamName:          "min-date",
						TimestampFieldName: "timestamp",
						TimestampFormat:    "2006-01-02 15:04:05",
						InitialTimestamp:   "invalid-timestamp",
					},
				},
			},
			expectedErr: `initial_timestamp "invalid-timestamp" could not be parsed`,
		},
		{
			name: "negative min_poll_interval",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
				MinPollInterval: -1 * time.Second,
			},
			expectedErr: "min_poll_interval must be greater than or equal to 0",
		},
		{
			name: "negative max_poll_interval",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
				MaxPollInterval: -1 * time.Minute,
			},
			expectedErr: "max_poll_interval must be greater than or equal to 0",
		},
		{
			name: "min_poll_interval greater than max_poll_interval",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
				MinPollInterval: 10 * time.Minute,
				MaxPollInterval: 5 * time.Minute,
			},
			expectedErr: "min_poll_interval (10m0s) must be less than or equal to max_poll_interval (5m0s)",
		},
		{
			name: "valid min_poll_interval",
			config: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				Pagination: PaginationConfig{
					Mode: paginationModeNone,
				},
				MinPollInterval: 30 * time.Second,
				MaxPollInterval: 5 * time.Minute,
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

	require.Equal(t, authModeNone, cfg.AuthMode)
	require.Equal(t, paginationModeNone, cfg.Pagination.Mode)
	require.Equal(t, 10*time.Second, cfg.MinPollInterval)
	require.Equal(t, 5*time.Minute, cfg.MaxPollInterval)
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
	require.Equal(t, 10*time.Second, restapiCfg.MinPollInterval)
	require.Equal(t, 5*time.Minute, restapiCfg.MaxPollInterval)
	require.Equal(t, authModeAPIKey, restapiCfg.AuthMode)
	require.Equal(t, "test-key", restapiCfg.APIKeyConfig.Value)
	require.Equal(t, "X-API-Key", restapiCfg.APIKeyConfig.HeaderName)
}
