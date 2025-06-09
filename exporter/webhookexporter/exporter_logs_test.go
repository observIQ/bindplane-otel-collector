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
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestNewLogsExporter(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         *SignalConfig
		expectError bool
	}{
		{
			name: "valid config",
			cfg: &SignalConfig{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "http://localhost:8080",
					Headers:  map[string]configopaque.String{"X-Test": configopaque.String("test-value")},
				},
				Verb:        POST,
				ContentType: "application/json",
			},
			expectError: false,
		},
		{
			name:        "nil logs config",
			cfg:         nil,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exp, err := newLogsExporter(context.Background(), tc.cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, exp)
				require.Contains(t, err.Error(), "logs config is required")
			} else {
				require.NoError(t, err)
				require.NotNil(t, exp)
				require.Equal(t, tc.cfg, exp.cfg)
				require.NotNil(t, exp.logger)
			}
		})
	}
}

func TestLogsExporterCapabilities(t *testing.T) {
	exp := &logsExporter{}
	caps := exp.Capabilities()
	require.False(t, caps.MutatesData)
}

func TestLogsExporterStartShutdown(t *testing.T) {
	exp := &logsExporter{
		cfg: &SignalConfig{
			ClientConfig: confighttp.ClientConfig{
				Endpoint: "http://localhost:8080",
			},
		},
		logger:   zap.NewNop(),
		settings: component.TelemetrySettings{},
	}
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
			serverResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: "failed to send request: 500 Internal Server Error",
		},
		{
			name: "connection error",
			serverResponse: func(w http.ResponseWriter, _ *http.Request) {
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
			cfg := &SignalConfig{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: server.URL,
					Headers:  map[string]configopaque.String{"X-Test": configopaque.String("test-value")},
				},
				Verb:        POST,
				ContentType: "application/json",
			}

			exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
			exp.start(context.Background(), componenttest.NewNopHost())
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
	testCases := []struct {
		name           string
		expectedFormat string
	}{
		{
			name:           "default json array format",
			expectedFormat: "json_array",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			cfg := &SignalConfig{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: server.URL,
					Headers:  map[string]configopaque.String{"X-Test": configopaque.String("test-value")},
				},
				Verb:        POST,
				ContentType: "application/json",
			}

			exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
			exp.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			// Create test logs with specific content
			logs := plog.NewLogs()
			resourceLogs := logs.ResourceLogs().AppendEmpty()
			scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
			logRecord := scopeLogs.LogRecords().AppendEmpty()
			logRecord.Body().SetStr("test log message")
			logRecord2 := scopeLogs.LogRecords().AppendEmpty()
			logRecord2.Body().SetStr("test log message 2")

			// Push logs
			err = exp.logsDataPusher(context.Background(), logs)
			require.NoError(t, err)

			// Get the received body
			received := <-receivedBody

			// Verify the format
			var jsonArray []string
			err = json.Unmarshal(received, &jsonArray)
			require.NoError(t, err)
			require.Len(t, jsonArray, 2)
			require.Equal(t, "test log message", jsonArray[0])
			require.Equal(t, "test log message 2", jsonArray[1])
		})
	}
}

func TestExtractLogBodies(t *testing.T) {
	tests := []struct {
		name     string
		logs     plog.Logs
		expected []any
	}{
		{
			name:     "empty logs",
			logs:     plog.NewLogs(),
			expected: []any{},
		},
		{
			name: "single log",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetStr("test log")
				return logs
			}(),
			expected: []any{"test log"},
		},
		{
			name: "multiple logs with different bodies",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()

				// Add first log
				lr1 := sl.LogRecords().AppendEmpty()
				lr1.Body().SetStr("first log")

				// Add second log
				lr2 := sl.LogRecords().AppendEmpty()
				lr2.Body().SetStr("second log")

				return logs
			}(),
			expected: []any{"first log", "second log"},
		},
		{
			name: "nested structure with multiple resource and scope logs",
			logs: func() plog.Logs {
				logs := plog.NewLogs()

				// First resource logs
				rl1 := logs.ResourceLogs().AppendEmpty()
				sl1 := rl1.ScopeLogs().AppendEmpty()
				lr1 := sl1.LogRecords().AppendEmpty()
				lr1.Body().SetStr("resource1 log")

				// Second resource logs
				rl2 := logs.ResourceLogs().AppendEmpty()
				sl2 := rl2.ScopeLogs().AppendEmpty()
				lr2 := sl2.LogRecords().AppendEmpty()
				lr2.Body().SetStr("resource2 log")

				return logs
			}(),
			expected: []any{"resource1 log", "resource2 log"},
		},
		{
			name: "log with map body",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				lr := sl.LogRecords().AppendEmpty()

				// Create a map body
				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("key1", "value1")
				bodyMap.PutInt("key2", 42)

				return logs
			}(),
			expected: []any{map[string]any{
				"key1": "value1",
				"key2": float64(42), // JSON numbers are unmarshaled as float64
			}},
		},
		{
			name: "log with JSON string",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetStr(`{"message": "test", "value": 42}`)
				return logs
			}(),
			expected: []any{map[string]any{
				"message": "test",
				"value":   float64(42),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLogBodies(tt.logs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractLogsFromLogRecords(t *testing.T) {
	tests := []struct {
		name     string
		records  plog.LogRecordSlice
		expected []any
	}{
		{
			name:     "empty records",
			records:  plog.NewLogRecordSlice(),
			expected: []any{},
		},
		{
			name: "single record",
			records: func() plog.LogRecordSlice {
				slice := plog.NewLogRecordSlice()
				lr := slice.AppendEmpty()
				lr.Body().SetStr("test log")
				return slice
			}(),
			expected: []any{"test log"},
		},
		{
			name: "multiple records",
			records: func() plog.LogRecordSlice {
				slice := plog.NewLogRecordSlice()

				lr1 := slice.AppendEmpty()
				lr1.Body().SetStr("first log")

				lr2 := slice.AppendEmpty()
				lr2.Body().SetStr("second log")

				return slice
			}(),
			expected: []any{"first log", "second log"},
		},
		{
			name: "record with JSON string",
			records: func() plog.LogRecordSlice {
				slice := plog.NewLogRecordSlice()
				lr := slice.AppendEmpty()
				lr.Body().SetStr(`{"message": "test"}`)
				return slice
			}(),
			expected: []any{map[string]any{
				"message": "test",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLogsFromLogRecords(tt.records)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractLogsFromScopeLogs(t *testing.T) {
	tests := []struct {
		name      string
		scopeLogs plog.ScopeLogsSlice
		expected  []any
	}{
		{
			name:      "empty scope logs",
			scopeLogs: plog.NewScopeLogsSlice(),
			expected:  []any{},
		},
		{
			name: "single scope log with single record",
			scopeLogs: func() plog.ScopeLogsSlice {
				slice := plog.NewScopeLogsSlice()
				sl := slice.AppendEmpty()
				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetStr("test log")
				return slice
			}(),
			expected: []any{"test log"},
		},
		{
			name: "multiple scope logs with multiple records",
			scopeLogs: func() plog.ScopeLogsSlice {
				slice := plog.NewScopeLogsSlice()

				// First scope log
				sl1 := slice.AppendEmpty()
				lr1 := sl1.LogRecords().AppendEmpty()
				lr1.Body().SetStr("scope1 log1")
				lr2 := sl1.LogRecords().AppendEmpty()
				lr2.Body().SetStr("scope1 log2")

				// Second scope log
				sl2 := slice.AppendEmpty()
				lr3 := sl2.LogRecords().AppendEmpty()
				lr3.Body().SetStr("scope2 log1")

				return slice
			}(),
			expected: []any{"scope1 log1", "scope1 log2", "scope2 log1"},
		},
		{
			name: "scope log with JSON string",
			scopeLogs: func() plog.ScopeLogsSlice {
				slice := plog.NewScopeLogsSlice()
				sl := slice.AppendEmpty()
				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetStr(`{"message": "test"}`)
				return slice
			}(),
			expected: []any{map[string]any{
				"message": "test",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLogsFromScopeLogs(tt.scopeLogs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractLogsFromResourceLogs(t *testing.T) {
	tests := []struct {
		name         string
		resourceLogs plog.ResourceLogsSlice
		expected     []any
	}{
		{
			name:         "empty resource logs",
			resourceLogs: plog.NewResourceLogsSlice(),
			expected:     []any{},
		},
		{
			name: "single resource log with single record",
			resourceLogs: func() plog.ResourceLogsSlice {
				slice := plog.NewResourceLogsSlice()
				rl := slice.AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetStr("test log")
				return slice
			}(),
			expected: []any{"test log"},
		},
		{
			name: "multiple resource logs with multiple records",
			resourceLogs: func() plog.ResourceLogsSlice {
				slice := plog.NewResourceLogsSlice()

				// First resource log
				rl1 := slice.AppendEmpty()
				sl1 := rl1.ScopeLogs().AppendEmpty()
				lr1 := sl1.LogRecords().AppendEmpty()
				lr1.Body().SetStr("resource1 log1")
				lr2 := sl1.LogRecords().AppendEmpty()
				lr2.Body().SetStr("resource1 log2")

				// Second resource log
				rl2 := slice.AppendEmpty()
				sl2 := rl2.ScopeLogs().AppendEmpty()
				lr3 := sl2.LogRecords().AppendEmpty()
				lr3.Body().SetStr("resource2 log1")

				return slice
			}(),
			expected: []any{"resource1 log1", "resource1 log2", "resource2 log1"},
		},
		{
			name: "resource log with JSON string",
			resourceLogs: func() plog.ResourceLogsSlice {
				slice := plog.NewResourceLogsSlice()
				rl := slice.AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetStr(`{"message": "test"}`)
				return slice
			}(),
			expected: []any{map[string]any{
				"message": "test",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLogsFromResourceLogs(tt.resourceLogs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestLogsDataPusherWithQueueSettings(t *testing.T) {
	testCases := []struct {
		name            string
		queueSettings   exporterhelper.QueueBatchConfig
		numLogs         int
		expectedBatches int
	}{
		{
			name: "default queue settings",
			queueSettings: exporterhelper.QueueBatchConfig{
				Enabled:      true,
				QueueSize:    5000,
				NumConsumers: 10,
			},
			numLogs:         100,
			expectedBatches: 1, // Default settings will batch all logs together
		},
		{
			name: "disabled queue",
			queueSettings: exporterhelper.QueueBatchConfig{
				Enabled: false,
			},
			numLogs:         50,
			expectedBatches: 1, // No batching when queue is disabled
		},
		{
			name: "small queue size",
			queueSettings: exporterhelper.QueueBatchConfig{
				Enabled:      true,
				QueueSize:    10,
				NumConsumers: 1,
			},
			numLogs:         25,
			expectedBatches: 3, // Should batch into smaller chunks due to queue size
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
			cfg := &SignalConfig{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: server.URL,
				},
				Verb:             POST,
				ContentType:      "application/json",
				QueueBatchConfig: tc.queueSettings,
			}
			exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
			exp.start(context.Background(), componenttest.NewNopHost())
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for i := 0; i < tc.expectedBatches; i++ {
				select {
				case body := <-receivedBodies:
					receivedCount++
					var receivedLogs []string
					err = json.Unmarshal(body, &receivedLogs)
					require.NoError(t, err)

					// Verify we received logs
					require.Greater(t, len(receivedLogs), 0)

					// For small queue size test, verify batch sizes
					if tc.queueSettings.QueueSize > 0 && int(tc.queueSettings.QueueSize) < tc.numLogs {
						require.LessOrEqual(t, len(receivedLogs), int(tc.queueSettings.QueueSize))
					}
				case <-ctx.Done():
					t.Fatalf("Timeout waiting for batch %d", i+1)
				}
			}
			require.Equal(t, tc.expectedBatches, receivedCount)
		})
	}
}

func TestLogsDataPusherWithQueueSettingsAndErrors(t *testing.T) {
	testCases := []struct {
		name          string
		queueSettings exporterhelper.QueueBatchConfig
		serverError   bool
		expectedError string
	}{
		{
			name: "queue enabled with server error",
			queueSettings: exporterhelper.QueueBatchConfig{
				Enabled:      true,
				QueueSize:    100,
				NumConsumers: 1,
			},
			serverError:   true,
			expectedError: "failed to send request",
		},
		{
			name: "queue disabled with server error",
			queueSettings: exporterhelper.QueueBatchConfig{
				Enabled: false,
			},
			serverError:   true,
			expectedError: "failed to send request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server that returns error if configured
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.serverError {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create exporter with test configuration
			cfg := &SignalConfig{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: server.URL,
				},
				Verb:             POST,
				ContentType:      "application/json",
				QueueBatchConfig: tc.queueSettings,
			}

			exp, err := newLogsExporter(context.Background(), cfg, exportertest.NewNopSettings(component.MustNewType("webhook")))
			exp.start(context.Background(), componenttest.NewNopHost())
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
