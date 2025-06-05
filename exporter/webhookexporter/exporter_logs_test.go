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

package webhookexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestNewLogsExporter(t *testing.T) {
	cfg := &Config{
		LogsConfig: &SignalConfig{
			Endpoint:    Endpoint("http://localhost:8080"),
			Verb:        POST,
			Headers:     map[string]string{"X-Test": "test-value"},
			ContentType: "application/json",
		},
	}

	exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
	require.NoError(t, err)
	require.NotNil(t, exp)
	require.Equal(t, cfg, exp.cfg)
	require.NotNil(t, exp.logger)
	require.NotNil(t, exp.client)
}

func TestLogsExporterCapabilities(t *testing.T) {
	exp := &logsExporter{}
	caps := exp.Capabilities()
	require.False(t, caps.MutatesData)
}

func TestLogsExporterStartShutdown(t *testing.T) {
	exp := &logsExporter{}
	err := exp.start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	err = exp.shutdown(context.Background())
	require.NoError(t, err)
}

func TestLogsDataPusher(t *testing.T) {
	testCases := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
		expectedBody   string
	}{
		{
			name: "successful push",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				require.Equal(t, "test-value", r.Header.Get("X-Test"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NotEmpty(t, body)

				w.WriteHeader(http.StatusOK)
			},
			expectedError: "",
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: "failed to send request: 500 Internal Server Error",
		},
		{
			name: "connection error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Simulate connection error by closing the connection
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
			},
			expectedError: "failed to send request:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tc.serverResponse))
			defer server.Close()

			// Create exporter with test server URL
			cfg := &Config{
				LogsConfig: &SignalConfig{
					Endpoint:    Endpoint(server.URL),
					Verb:        POST,
					Headers:     map[string]string{"X-Test": "test-value"},
					ContentType: "application/json",
				},
			}

			exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
			require.NoError(t, err)

			// Create test logs
			logs := plog.NewLogs()
			resourceLogs := logs.ResourceLogs().AppendEmpty()
			scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
			logRecord := scopeLogs.LogRecords().AppendEmpty()
			logRecord.Body().SetStr("test log message")

			// Push logs
			err = exp.logsDataPusher(context.Background(), logs)
			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Integration test that verifies the actual data being sent matches what's received
func TestLogsDataPusherIntegration(t *testing.T) {
	// Create a channel to receive the request body
	receivedBody := make(chan []byte, 1)

	// Create test server that captures the request body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBody <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create exporter
	cfg := &Config{
		LogsConfig: &SignalConfig{
			Endpoint:    Endpoint(server.URL),
			Verb:        POST,
			Headers:     map[string]string{"X-Test": "test-value"},
			ContentType: "application/json",
		},
	}

	exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
	require.NoError(t, err)

	// Create test logs with specific content
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("test log message")

	// Push logs
	err = exp.logsDataPusher(context.Background(), logs)
	require.NoError(t, err)

	// Get the received body
	received := <-receivedBody

	// Unmarshal the received body
	var receivedLogs []map[string]interface{}
	err = json.Unmarshal(received, &receivedLogs)
	require.NoError(t, err)

	// Verify the content
	require.Len(t, receivedLogs, 1)
	require.Contains(t, receivedLogs[0], "resourceLog")
}

func TestLogsDataPusherWithBatching(t *testing.T) {
	testCases := []struct {
		name            string
		limit           int
		numLogs         int
		expectedBatches int
	}{
		{
			name:            "no batching with zero limit",
			limit:           0,
			numLogs:         30,
			expectedBatches: 1,
		},
		{
			name:            "no batching with negative limit",
			limit:           -1,
			numLogs:         30,
			expectedBatches: 1,
		},
		{
			name:            "batching with limit of 10",
			limit:           10,
			numLogs:         25,
			expectedBatches: 3,
		},
		{
			name:            "exact batch size",
			limit:           10,
			numLogs:         10,
			expectedBatches: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a channel to receive request bodies
			receivedBodies := make(chan []byte, tc.expectedBatches)

			// Create test server that captures request bodies
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				receivedBodies <- body
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create exporter with test configuration
			cfg := &Config{
				LogsConfig: &SignalConfig{
					Endpoint:    Endpoint(server.URL),
					Verb:        POST,
					ContentType: "application/json",
					Limit:       tc.limit,
				},
			}

			exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
			require.NoError(t, err)

			// Create test logs
			logs := plog.NewLogs()
			resourceLogs := logs.ResourceLogs().AppendEmpty()
			scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

			// Add the specified number of log records
			for i := 0; i < tc.numLogs; i++ {
				logRecord := scopeLogs.LogRecords().AppendEmpty()
				logRecord.Body().SetStr(fmt.Sprintf("test log message %d", i))
			}

			// Push logs
			err = exp.logsDataPusher(context.Background(), logs)
			require.NoError(t, err)

			// Verify the number of batches
			receivedCount := 0
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			for i := 0; i < tc.expectedBatches; i++ {
				select {
				case body := <-receivedBodies:
					receivedCount++
					var receivedLogs map[string]interface{}
					err = json.Unmarshal(body, &receivedLogs)
					require.NoError(t, err)

					// Verify the structure contains resourceLogs
					require.Contains(t, receivedLogs, "resourceLogs")
					resourceLogs, ok := receivedLogs["resourceLogs"].([]interface{})
					require.True(t, ok)

					// Count the number of log records in this batch
					var logCount int
					for _, rl := range resourceLogs {
						rlMap, ok := rl.(map[string]interface{})
						require.True(t, ok)
						scopeLogs, ok := rlMap["scopeLogs"].([]interface{})
						require.True(t, ok)
						for _, sl := range scopeLogs {
							slMap, ok := sl.(map[string]interface{})
							require.True(t, ok)
							logRecords, ok := slMap["logRecords"].([]interface{})
							require.True(t, ok)
							logCount += len(logRecords)
						}
					}

					// Verify batch size
					if tc.limit > 0 {
						require.LessOrEqual(t, logCount, tc.limit)
					}
				case <-ctx.Done():
					t.Fatalf("Timeout waiting for batch %d", i+1)
				}
			}
			require.Equal(t, tc.expectedBatches, receivedCount)
		})
	}
}
