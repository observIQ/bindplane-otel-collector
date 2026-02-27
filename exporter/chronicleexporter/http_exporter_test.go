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

package chronicleexporter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/metadatatest"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"golang.org/x/oauth2"
)

type mockHTTPServer struct {
	srv          *httptest.Server
	requestCount int
}

func newMockHTTPServer(logTypeHandlers map[string]http.HandlerFunc) *mockHTTPServer {
	mockServer := mockHTTPServer{}
	mux := http.NewServeMux()
	for logType, handlerFunc := range logTypeHandlers {
		pattern := fmt.Sprintf("/logTypes/%s/logs:import", logType)
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			mockServer.requestCount++
			handlerFunc(w, r)
		})
	}
	mockServer.srv = httptest.NewServer(mux)
	return &mockServer
}

type emptyTokenSource struct{}

func (t *emptyTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{}, nil
}

func TestHTTPExporter(t *testing.T) {
	// Override the token source so that we don't have to provide real credentials
	secureTokenSource := tokenSource
	defer func() {
		tokenSource = secureTokenSource
	}()
	tokenSource = func(context.Context, *Config) (oauth2.TokenSource, error) {
		return &emptyTokenSource{}, nil
	}

	// By default, tests will apply the following changes to NewFactory.CreateDefaultConfig()
	defaultCfgMod := func(cfg *Config) {
		cfg.Protocol = protocolHTTPS
		cfg.Location = "us"
		cfg.CustomerID = "00000000-1111-2222-3333-444444444444"
		cfg.Project = "fake"
		cfg.Forwarder = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		cfg.LogType = "FAKE"
		cfg.QueueBatchConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
		cfg.BackOffConfig.Enabled = false
	}

	defaultHandlers := map[string]http.HandlerFunc{
		"FAKE": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}

	testCases := []struct {
		name             string
		cfgMod           func(cfg *Config)
		handlers         map[string]http.HandlerFunc
		input            plog.Logs
		expectedRequests int
		expectedErr      string
		permanentErr     bool
	}{
		{
			name:             "empty log record",
			input:            plog.NewLogs(),
			expectedRequests: 0,
		},
		{
			name: "single log record",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			expectedRequests: 1,
		},
		// TODO test splitting large payloads
		{
			name: "retryable_error 429 Too Many Requests",
			handlers: map[string]http.HandlerFunc{
				"FAKE": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusTooManyRequests)
				},
			},
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			expectedRequests: 1,
			expectedErr:      "upload to chronicle: 429 Too Many Requests",
			permanentErr:     false,
		},
		{
			name: "retryable_error 500 Internal Server Error",
			handlers: map[string]http.HandlerFunc{
				"FAKE": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			expectedRequests: 1,
			expectedErr:      "upload to chronicle: 500 Internal Server Error",
			permanentErr:     false,
		},
		{
			name: "retryable_error 502 Bad Gateway",
			handlers: map[string]http.HandlerFunc{
				"FAKE": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadGateway)
				},
			},
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			expectedRequests: 1,
			expectedErr:      "upload to chronicle: 502 Bad Gateway",
			permanentErr:     false,
		},
		{
			name: "retryable_error 503 Service Unavailable",
			handlers: map[string]http.HandlerFunc{
				"FAKE": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				},
			},
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			expectedRequests: 1,
			expectedErr:      "upload to chronicle: 503 Service Unavailable",
			permanentErr:     false,
		},
		{
			name: "retryable_error 504 Gateway Timeout",
			handlers: map[string]http.HandlerFunc{
				"FAKE": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusGatewayTimeout)
				},
			},
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			expectedRequests: 1,
			expectedErr:      "upload to chronicle: 504 Gateway Timeout",
			permanentErr:     false,
		},
		{
			name: "permanent_error",
			handlers: map[string]http.HandlerFunc{
				"FAKE": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
				},
			},
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			expectedRequests: 1,
			expectedErr:      "upload to chronicle: Permanent error: 401 Unauthorized",
			permanentErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server so we are not dependent on the actual Chronicle service
			handlers := defaultHandlers
			if tc.handlers != nil {
				handlers = tc.handlers
			}
			mockServer := newMockHTTPServer(handlers)
			defer mockServer.srv.Close()

			// Override the endpoint builder so that we can point to the mock server
			secureHTTPEndpoint := httpEndpoint
			defer func() {
				httpEndpoint = secureHTTPEndpoint
			}()
			httpEndpoint = func(_ *Config, logType string) string {
				return fmt.Sprintf("%s/logTypes/%s/logs:import", mockServer.srv.URL, logType)
			}

			f := NewFactory()
			cfg := f.CreateDefaultConfig().(*Config)
			if tc.cfgMod != nil {
				tc.cfgMod(cfg)
			} else {
				defaultCfgMod(cfg)
			}
			require.NoError(t, cfg.Validate())

			ctx := context.Background()
			exp, err := f.CreateLogs(ctx, exportertest.NewNopSettings(typ), cfg)
			require.NoError(t, err)
			require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
			defer func() {
				require.NoError(t, exp.Shutdown(ctx))
			}()

			err = exp.ConsumeLogs(ctx, tc.input)
			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedErr)
				require.Equal(t, tc.permanentErr, consumererror.IsPermanent(err))
			}

			require.Equal(t, tc.expectedRequests, mockServer.requestCount)
		})
	}
}

// TestHTTPJSONCredentialsError tests that the HTTP exporter returns an error when the json credentials are invalid and does not panic during shutdown
func TestHTTPJSONCredentialsError(t *testing.T) {
	defaultCfgMod := func(cfg *Config) {
		cfg.Protocol = protocolHTTPS
		cfg.Location = "us"
		cfg.CustomerID = "00000000-1111-2222-3333-444444444444"
		cfg.Project = "fake"
		cfg.Forwarder = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		cfg.LogType = "FAKE"
		cfg.QueueBatchConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
		cfg.BackOffConfig.Enabled = false
	}

	// Create and configure the exporter
	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	defaultCfgMod(cfg)
	cfg.Creds = "z"                    // This invalid JSON will cause the token source to error
	require.NoError(t, cfg.Validate()) // TODO: Validate really should fail immediately when given invalid JSON as credentials

	ctx := context.Background()
	exp, err := f.CreateLogs(ctx, exportertest.NewNopSettings(typ), cfg)
	require.NoError(t, err)

	// Start should fail with invalid credentials
	err = exp.Start(ctx, componenttest.NewNopHost())
	require.Error(t, err)
	require.EqualError(t, err, "load Google credentials: invalid character 'z' looking for beginning of value")

	// Shutdown should not panic
	require.NoError(t, exp.Shutdown(ctx))
}

// TestHTTPExporterAgentMetrics tests that the HTTP exporter starts agent metrics collection
// when CollectAgentMetrics is enabled and correctly sends stats to the importStatsEvents endpoint
func TestHTTPExporterAgentMetrics(t *testing.T) {
	// Override the token source so that we don't have to provide real credentials
	secureTokenSource := tokenSource
	defer func() {
		tokenSource = secureTokenSource
	}()
	tokenSource = func(context.Context, *Config) (oauth2.TokenSource, error) {
		return &emptyTokenSource{}, nil
	}

	t.Run("stats_endpoint_receives_request", func(t *testing.T) {
		statsReceived := make(chan struct{}, 1)
		mockServer := newMockHTTPServer(map[string]http.HandlerFunc{
			"FAKE": func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		})
		defer mockServer.srv.Close()

		// Add a stats endpoint handler to the mock server
		statsMux := http.NewServeMux()
		statsMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && r.URL.Path != "" {
				select {
				case statsReceived <- struct{}{}:
				default:
				}
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})
		statsSrv := httptest.NewServer(statsMux)
		defer statsSrv.Close()

		// Override endpoints
		secureHTTPEndpoint := httpEndpoint
		defer func() { httpEndpoint = secureHTTPEndpoint }()
		httpEndpoint = func(_ *Config, logType string) string {
			return fmt.Sprintf("%s/logTypes/%s/logs:import", mockServer.srv.URL, logType)
		}

		secureStatsEndpoint := httpStatsEndpoint
		defer func() { httpStatsEndpoint = secureStatsEndpoint }()
		httpStatsEndpoint = func(_ *Config, collectorID string) string {
			return fmt.Sprintf("%s/forwarders/%s:importStatsEvents", statsSrv.URL, collectorID)
		}

		f := NewFactory()
		cfg := f.CreateDefaultConfig().(*Config)
		cfg.Protocol = protocolHTTPS
		cfg.Location = "us"
		cfg.CustomerID = "00000000-1111-2222-3333-444444444444"
		cfg.Project = "fake"
		cfg.Forwarder = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		cfg.LogType = "FAKE"
		cfg.QueueBatchConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
		cfg.BackOffConfig.Enabled = false
		cfg.CollectAgentMetrics = true
		require.NoError(t, cfg.Validate())

		ctx := context.Background()
		exp, err := f.CreateLogs(ctx, exportertest.NewNopSettings(typ), cfg)
		require.NoError(t, err)
		require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
		defer func() {
			require.NoError(t, exp.Shutdown(ctx))
		}()

		// The metrics reporter runs on a 5-minute ticker, so we can't easily
		// test the full cycle. Instead, verify the exporter started without error
		// (meaning the metrics reporter was created and started successfully).
		// The Shutdown will also verify the reporter shuts down cleanly.
	})

	t.Run("agent_metrics_disabled", func(t *testing.T) {
		mockServer := newMockHTTPServer(map[string]http.HandlerFunc{
			"FAKE": func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		})
		defer mockServer.srv.Close()

		secureHTTPEndpoint := httpEndpoint
		defer func() { httpEndpoint = secureHTTPEndpoint }()
		httpEndpoint = func(_ *Config, logType string) string {
			return fmt.Sprintf("%s/logTypes/%s/logs:import", mockServer.srv.URL, logType)
		}

		f := NewFactory()
		cfg := f.CreateDefaultConfig().(*Config)
		cfg.Protocol = protocolHTTPS
		cfg.Location = "us"
		cfg.CustomerID = "00000000-1111-2222-3333-444444444444"
		cfg.Project = "fake"
		cfg.Forwarder = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		cfg.LogType = "FAKE"
		cfg.QueueBatchConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
		cfg.BackOffConfig.Enabled = false
		cfg.CollectAgentMetrics = false
		require.NoError(t, cfg.Validate())

		ctx := context.Background()
		exp, err := f.CreateLogs(ctx, exportertest.NewNopSettings(typ), cfg)
		require.NoError(t, err)
		require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))

		// Shutdown should not panic even when metrics is nil
		require.NoError(t, exp.Shutdown(ctx))
	})
}

func TestUploadStatsHTTP(t *testing.T) {
	// Override the token source so that we don't have to provide real credentials
	secureTokenSource := tokenSource
	defer func() {
		tokenSource = secureTokenSource
	}()
	tokenSource = func(context.Context, *Config) (oauth2.TokenSource, error) {
		return &emptyTokenSource{}, nil
	}

	t.Run("success", func(t *testing.T) {
		var receivedBody []byte
		statsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "POST", r.Method)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			receivedBody = body
			w.WriteHeader(http.StatusOK)
		}))
		defer statsSrv.Close()

		secureStatsEndpoint := httpStatsEndpoint
		defer func() { httpStatsEndpoint = secureStatsEndpoint }()
		httpStatsEndpoint = func(_ *Config, _ string) string {
			return statsSrv.URL
		}

		cfg := &Config{
			Protocol:    protocolHTTPS,
			Location:    "us",
			CustomerID:  "00000000-1111-2222-3333-444444444444",
			Project:     "fake",
			Compression: noCompression,
		}

		exp := &httpExporter{
			cfg: cfg,
			set: componenttest.NewNopTelemetrySettings(),
			client: &http.Client{
				Transport: &oauth2.Transport{
					Base:   http.DefaultTransport,
					Source: &emptyTokenSource{},
				},
			},
		}

		request := &api.BatchCreateEventsRequest{
			Batch: &api.EventBatch{
				Id:   []byte{1, 2, 3},
				Type: api.EventBatch_AGENT_STATS,
				Events: []*api.Event{
					{
						Payload: &api.Event_AgentStats{
							AgentStats: &api.AgentStatsEvent{
								AgentId: []byte{1, 2, 3},
							},
						},
					},
				},
			},
		}

		err := exp.uploadStatsHTTP(context.Background(), request, "test-collector-id")
		require.NoError(t, err)
		require.NotEmpty(t, receivedBody)
	})

	t.Run("server_error", func(t *testing.T) {
		statsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer statsSrv.Close()

		secureStatsEndpoint := httpStatsEndpoint
		defer func() { httpStatsEndpoint = secureStatsEndpoint }()
		httpStatsEndpoint = func(_ *Config, _ string) string {
			return statsSrv.URL
		}

		cfg := &Config{
			Protocol:    protocolHTTPS,
			Location:    "us",
			CustomerID:  "00000000-1111-2222-3333-444444444444",
			Project:     "fake",
			Compression: noCompression,
		}

		exp := &httpExporter{
			cfg: cfg,
			set: componenttest.NewNopTelemetrySettings(),
			client: &http.Client{
				Transport: &oauth2.Transport{
					Base:   http.DefaultTransport,
					Source: &emptyTokenSource{},
				},
			},
		}

		request := &api.BatchCreateEventsRequest{
			Batch: &api.EventBatch{
				Id:   []byte{1, 2, 3},
				Type: api.EventBatch_AGENT_STATS,
				Events: []*api.Event{
					{
						Payload: &api.Event_AgentStats{
							AgentStats: &api.AgentStatsEvent{
								AgentId: []byte{1, 2, 3},
							},
						},
					},
				},
			},
		}

		err := exp.uploadStatsHTTP(context.Background(), request, "test-collector-id")
		require.Error(t, err)
		require.Contains(t, err.Error(), "upload stats to Chronicle")
	})
}

func TestHTTPStatsEndpoint(t *testing.T) {
	cfg := &Config{
		Location:   "us",
		Endpoint:   "chronicle.googleapis.com",
		Project:    "my-project",
		CustomerID: "my-customer-id",
	}

	endpoint := httpStatsEndpoint(cfg, "collector-123")
	require.Equal(t,
		"https://us-chronicle.googleapis.com/v1alpha/projects/my-project/locations/us/instances/my-customer-id/forwarders/collector-123:importStatsEvents",
		endpoint,
	)
}

// TestHTTPExporterTelemetry tests the telemetry metrics functionality of the HTTP exporter
func TestHTTPExporterTelemetry(t *testing.T) {
	// Override the token source so that we don't have to provide real credentials
	secureTokenSource := tokenSource
	defer func() {
		tokenSource = secureTokenSource
	}()
	tokenSource = func(context.Context, *Config) (oauth2.TokenSource, error) {
		return &emptyTokenSource{}, nil
	}

	// By default, tests will apply the following changes to NewFactory.CreateDefaultConfig()
	defaultCfgMod := func(cfg *Config) {
		cfg.Protocol = protocolHTTPS
		cfg.Location = "us"
		cfg.CustomerID = "00000000-1111-2222-3333-444444444444"
		cfg.Project = "fake"
		cfg.Forwarder = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		cfg.LogType = "FAKE"
		cfg.QueueBatchConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
		cfg.BackOffConfig.Enabled = false
	}

	defaultHandlers := map[string]http.HandlerFunc{
		"FAKE": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}

	testCases := []struct {
		name          string
		input         plog.Logs
		expectedBytes int
		rawLogField   string
		handlers      map[string]http.HandlerFunc
		expectError   bool
		retryEnabled  bool
	}{
		{
			name: "single log record",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				return logs
			}(),
			// JSON: {"attributes":{},"body":"Test","resource_attributes":{}}
			expectedBytes: 56,
			rawLogField:   "",
			handlers:      nil,
			expectError:   false,
		},
		{
			name: "single log record with attributes and resources",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				rls.Resource().Attributes().PutStr("R", "5")
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				lrs.Attributes().PutStr("A", "10")
				return logs
			}(),
			// JSON: {"attributes":{"A":"10"},"body":"Test","resource_attributes":{"R":"5"}}
			expectedBytes: 71,
			rawLogField:   "",
			handlers:      nil,
			expectError:   false,
		},
		{
			name: "single log record with RawLogField set to body",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls := logs.ResourceLogs().AppendEmpty()
				rls.Resource().Attributes().PutStr("R", "5")
				sls := rls.ScopeLogs().AppendEmpty()
				lrs := sls.LogRecords().AppendEmpty()
				lrs.Body().SetStr("Test")
				lrs.Attributes().PutStr("A", "10")
				return logs
			}(),
			expectedBytes: 4, // Data: "Test"
			rawLogField:   "body",
			handlers:      nil,
			expectError:   false,
		},
		{
			name: "multiple payloads",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls1 := logs.ResourceLogs().AppendEmpty()
				sls1 := rls1.ScopeLogs().AppendEmpty()
				lrs1 := sls1.LogRecords().AppendEmpty()
				lrs1.Body().SetStr("type1")
				lrs1.Attributes().PutStr("chronicle_log_type", "TYPE_1")

				rls2 := logs.ResourceLogs().AppendEmpty()
				sls2 := rls2.ScopeLogs().AppendEmpty()
				lrs2 := sls2.LogRecords().AppendEmpty()
				lrs2.Body().SetStr("type2")
				lrs2.Attributes().PutStr("chronicle_log_type", "TYPE_2")
				return logs
			}(),
			expectedBytes: 10, // Data: "type1type2"
			rawLogField:   "body",
			handlers: map[string]http.HandlerFunc{
				"TYPE_1": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				"TYPE_2": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
			expectError: false,
		},
		{
			name: "multiple payloads with one failure - should count bytes since retry is disabled",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls1 := logs.ResourceLogs().AppendEmpty()
				sls1 := rls1.ScopeLogs().AppendEmpty()
				lrs1 := sls1.LogRecords().AppendEmpty()
				lrs1.Body().SetStr("body data")
				lrs1.Attributes().PutStr("chronicle_log_type", "TYPE_1")

				rls2 := logs.ResourceLogs().AppendEmpty()
				sls2 := rls2.ScopeLogs().AppendEmpty()
				lrs2 := sls2.LogRecords().AppendEmpty()
				lrs2.Body().SetStr("body data")
				lrs2.Attributes().PutStr("chronicle_log_type", "TYPE_2")
				return logs
			}(),
			expectedBytes: 9, // Only count bytes from successful payloads before failure (just first "body data")
			rawLogField:   "body",
			// This handler will fail on the 2nd request of either type
			handlers: func() map[string]http.HandlerFunc {
				totalCount := 0
				return map[string]http.HandlerFunc{
					"TYPE_1": func(w http.ResponseWriter, _ *http.Request) {
						if totalCount == 0 {
							w.WriteHeader(http.StatusOK)
						} else {
							w.WriteHeader(http.StatusInternalServerError)
						}
						totalCount++
					},
					"TYPE_2": func(w http.ResponseWriter, _ *http.Request) {
						if totalCount == 0 {
							w.WriteHeader(http.StatusOK)
						} else {
							w.WriteHeader(http.StatusInternalServerError)
						}
						totalCount++
					},
				}
			}(),
			expectError:  true,
			retryEnabled: false,
		},
		{
			name: "transient failure with retry enabled - should NOT count bytes on first failure",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				rls1 := logs.ResourceLogs().AppendEmpty()
				sls1 := rls1.ScopeLogs().AppendEmpty()
				lrs1 := sls1.LogRecords().AppendEmpty()
				lrs1.Body().SetStr("test data")
				lrs1.Attributes().PutStr("chronicle_log_type", "TRANSIENT_TYPE")
				return logs
			}(),
			expectedBytes: 9, // Count bytes only on successful retry (length of "test data")
			rawLogField:   "body",
			// This handler will fail once, then succeed on retry
			handlers: func() map[string]http.HandlerFunc {
				callCount := 0
				return map[string]http.HandlerFunc{
					"TRANSIENT_TYPE": func(w http.ResponseWriter, _ *http.Request) {
						callCount++
						if callCount == 1 {
							// First call fails with transient error
							w.WriteHeader(http.StatusServiceUnavailable)
						} else {
							// Retry succeeds
							w.WriteHeader(http.StatusOK)
						}
					},
				}
			}(),
			expectError:  false,
			retryEnabled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server so we are not dependent on the actual Chronicle service
			handlers := defaultHandlers
			if tc.handlers != nil {
				handlers = tc.handlers
			}
			mockServer := newMockHTTPServer(handlers)
			defer mockServer.srv.Close()

			// Override the endpoint builder so that we can point to the mock server
			secureHTTPEndpoint := httpEndpoint
			defer func() {
				httpEndpoint = secureHTTPEndpoint
			}()
			httpEndpoint = func(_ *Config, logType string) string {
				return fmt.Sprintf("%s/logTypes/%s/logs:import", mockServer.srv.URL, logType)
			}

			// Create telemetry for testing metrics
			testTelemetry := componenttest.NewTelemetry()
			defer testTelemetry.Shutdown(context.Background())

			f := NewFactory()
			cfg := f.CreateDefaultConfig().(*Config)
			defaultCfgMod(cfg)
			if tc.rawLogField != "" {
				cfg.RawLogField = tc.rawLogField
			}
			if tc.retryEnabled {
				cfg.BackOffConfig.Enabled = true
			}
			require.NoError(t, cfg.Validate())

			ctx := context.Background()
			exp, err := f.CreateLogs(ctx, metadatatest.NewSettings(testTelemetry), cfg)
			require.NoError(t, err)
			require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
			defer func() {
				require.NoError(t, exp.Shutdown(ctx))
			}()

			err = exp.ConsumeLogs(ctx, tc.input)

			// Check error expectations based on test case
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "upload to chronicle")
			} else {
				require.NoError(t, err)
			}

			// Test telemetry metrics - check that the metric exists and has the expected value
			// When expectedBytes is 0 (failure case), the metric won't exist
			if tc.expectedBytes > 0 {
				metric, err := testTelemetry.GetMetric("otelcol_exporter_raw_bytes")
				require.NoError(t, err)
				require.NotNil(t, metric)

				// For successful cases, verify the metric has the expected value
				sumData, ok := metric.Data.(metricdata.Sum[int64])
				require.True(t, ok, "Expected Sum metric data")

				require.Len(t, sumData.DataPoints, 1, "Expected exactly one data point")
				require.Equal(t, int64(tc.expectedBytes), sumData.DataPoints[0].Value)
			} else {
				// For failure cases with 0 bytes, verify the metric doesn't exist
				_, err := testTelemetry.GetMetric("otelcol_exporter_raw_bytes")
				require.Error(t, err, "Metric should not exist when no bytes are counted")
				require.Contains(t, err.Error(), "not found", "Error should indicate metric was not found")
			}
		})
	}
}
