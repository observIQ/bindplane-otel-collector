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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
)

func TestNewRESTAPIClient(t *testing.T) {
	testCases := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config with apikey auth",
			cfg: &Config{
				URL:      "https://api.example.com/data",
				AuthMode: authModeAPIKey,
				APIKeyConfig: APIKeyConfig{
					HeaderName: "X-API-Key",
					Value:      "test-key",
				},
				ClientConfig: confighttp.ClientConfig{},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			host := componenttest.NewNopHost()
			settings := componenttest.NewNopTelemetrySettings()

			client, err := newRESTAPIClient(ctx, settings, tc.cfg, host)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
			}
		})
	}
}

func TestRESTAPIClient_GetJSON_NoAuth(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header is set
		require.Equal(t, "test-key", r.Header.Get("X-API-Key"))

		// Return JSON array
		response := []map[string]any{
			{"id": "1", "name": "test1"},
			{"id": "2", "name": "test2"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-key",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 2)
	require.Equal(t, "1", data[0]["id"])
	require.Equal(t, "test1", data[0]["name"])
	require.Equal(t, "2", data[1]["id"])
	require.Equal(t, "test2", data[1]["name"])
}

func TestRESTAPIClient_GetJSON_NoAuthMode(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no Authorization header is set
		require.Empty(t, r.Header.Get("Authorization"))

		// Return JSON array
		response := []map[string]any{
			{"id": "1", "name": "public-data"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:          server.URL,
		AuthMode:     authModeNone,
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
	require.Equal(t, "1", data[0]["id"])
	require.Equal(t, "public-data", data[0]["name"])
}

func TestRESTAPIClient_GetJSON_APIKeyAuth(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		require.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))

		response := []map[string]any{
			{"id": "1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-api-key",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
}

func TestRESTAPIClient_GetJSON_BearerAuth(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify bearer token
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		response := []map[string]any{
			{"id": "1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeBearer,
		BearerConfig: BearerConfig{
			Token: "test-token",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
}

func TestRESTAPIClient_GetJSON_BasicAuth(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify basic auth
		username, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "testuser", username)
		require.Equal(t, "testpass", password)

		response := []map[string]any{
			{"id": "1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeBasic,
		BasicConfig: BasicConfig{
			Username: "testuser",
			Password: "testpass",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
}

func TestRESTAPIClient_GetJSON_OAuth2Auth(t *testing.T) {
	// Create a mock OAuth2 token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a token request
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "/token", r.URL.Path)

		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err)

		// Verify grant type (client credentials can be sent in Authorization header or form)
		require.Equal(t, "client_credentials", r.Form.Get("grant_type"))

		// Verify client credentials - they can be in form or Authorization header
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		if clientID == "" {
			// Check Authorization header for Basic auth
			username, password, ok := r.BasicAuth()
			require.True(t, ok)
			clientID = username
			clientSecret = password
		}
		require.Equal(t, "test-client-id", clientID)
		require.Equal(t, "test-client-secret", clientSecret)

		// Return access token
		response := map[string]any{
			"access_token": "test-oauth2-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer tokenServer.Close()

	// Create API test server
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify OAuth2 bearer token
		require.Equal(t, "Bearer test-oauth2-token", r.Header.Get("Authorization"))

		response := []map[string]any{
			{"id": "1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer apiServer.Close()

	cfg := &Config{
		URL:      apiServer.URL,
		AuthMode: authModeOAuth2,
		OAuth2Config: OAuth2Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			TokenURL:     tokenServer.URL + "/token",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, apiServer.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
}

func TestRESTAPIClient_GetJSON_OAuth2Auth_WithScopes(t *testing.T) {
	// Create a mock OAuth2 token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err)

		// Verify scopes are included
		require.Equal(t, "read write", r.Form.Get("scope"))

		// Return access token
		response := map[string]any{
			"access_token": "test-oauth2-token-with-scopes",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer tokenServer.Close()

	// Create API test server
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify OAuth2 bearer token
		require.Equal(t, "Bearer test-oauth2-token-with-scopes", r.Header.Get("Authorization"))

		response := []map[string]any{
			{"id": "1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer apiServer.Close()

	cfg := &Config{
		URL:      apiServer.URL,
		AuthMode: authModeOAuth2,
		OAuth2Config: OAuth2Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			TokenURL:     tokenServer.URL + "/token",
			Scopes:       []string{"read", "write"},
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, apiServer.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
}

func TestRESTAPIClient_GetJSON_AkamaiEdgeGridAuth(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Akamai EdgeGrid auth header format
		authHeader := r.Header.Get("Authorization")
		require.NotEmpty(t, authHeader)
		require.Contains(t, authHeader, "EG1-HMAC-SHA256")
		require.Contains(t, authHeader, "client_token=test-client-token")
		require.Contains(t, authHeader, "access_token=test-access-token")
		require.Contains(t, authHeader, "timestamp=")
		require.Contains(t, authHeader, "nonce=")
		require.Contains(t, authHeader, "signature=")

		response := []map[string]any{
			{"id": "1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeAkamaiEdgeGrid,
		AkamaiEdgeGridConfig: AkamaiEdgeGridConfig{
			AccessToken:  "test-access-token",
			ClientToken:  "test-client-token",
			ClientSecret: "test-client-secret",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
}

func TestRESTAPIClient_GetJSON_WithQueryParams(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		require.Equal(t, "value1", r.URL.Query().Get("param1"))
		require.Equal(t, "value2", r.URL.Query().Get("param2"))

		response := []map[string]any{
			{"id": "1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-key",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	params.Set("param1", "value1")
	params.Set("param2", "value2")
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 1)
}

func TestRESTAPIClient_GetJSON_ResponseField(t *testing.T) {
	// Create a test server that returns nested JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]any{
			"data": []map[string]any{
				{"id": "1", "name": "test1"},
				{"id": "2", "name": "test2"},
			},
			"meta": map[string]any{
				"count": 2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:           server.URL,
		ResponseField: "data",
		AuthMode:      authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-key",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.Len(t, data, 2)
	require.Equal(t, "1", data[0]["id"])
	require.Equal(t, "test1", data[0]["name"])
}

func TestRESTAPIClient_GetJSON_HTTPError(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-key",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.Error(t, err)
	require.Nil(t, data)
	require.Contains(t, err.Error(), "500")
}

func TestRESTAPIClient_GetJSON_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-key",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.Error(t, err)
	require.Nil(t, data)
}

func TestRESTAPIClient_GetJSON_EmptyArray(t *testing.T) {
	// Create a test server that returns empty array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := []map[string]any{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:      server.URL,
		AuthMode: authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-key",
		},
		ClientConfig: confighttp.ClientConfig{},
	}

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	client, err := newRESTAPIClient(ctx, settings, cfg, host)
	require.NoError(t, err)

	params := url.Values{}
	data, err := client.GetJSON(ctx, server.URL, params)
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Len(t, data, 0)
}
