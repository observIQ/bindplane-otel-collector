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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/metadatatest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
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
		cfg.QueueBatchConfig.Enabled = false
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
		expectedBytes    int
		expectedErr      string
		permanentErr     bool
	}{
		{
			name:             "empty log record",
			input:            plog.NewLogs(),
			expectedRequests: 0,
			expectedBytes:    0,
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
			expectedBytes:    56, // JSON: {"attributes":{},"body":"Test","resource_attributes":{}}
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
			expectedRequests: 1,
			// JSON: {"attributes":{"A":"10"},"body":"Test","resource_attributes":{"R":"5"}}
			expectedBytes: 71,
		},
		// TODO test splitting large payloads
		{
			name: "transient_error",
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
			expectedBytes:    0,
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
			expectedBytes:    0,
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
			if tc.cfgMod != nil {
				tc.cfgMod(cfg)
			} else {
				defaultCfgMod(cfg)
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
			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedErr)
				require.Equal(t, tc.permanentErr, consumererror.IsPermanent(err))
			}

			require.Equal(t, tc.expectedRequests, mockServer.requestCount)

			if tc.expectedErr == "" {
				// Test telemetry metrics - check that the metric exists and has the expected value
				metric, err := testTelemetry.GetMetric("otelcol_exporter_raw_bytes")
				require.NoError(t, err)
				require.NotNil(t, metric)

				// For successful cases, verify the metric has the expected value
				sumData, ok := metric.Data.(metricdata.Sum[int64])
				require.True(t, ok, "Expected Sum metric data")
				require.Len(t, sumData.DataPoints, 1, "Expected exactly one data point")
				require.Equal(t, int64(tc.expectedBytes), sumData.DataPoints[0].Value)
			}
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
		cfg.QueueBatchConfig.Enabled = false
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
