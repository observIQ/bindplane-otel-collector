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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/observiq/bindplane-otel-collector/receiver/restapireceiver/internal/metadata"
)

// TestIntegration_EndToEnd_Logs tests a complete end-to-end scenario for logs collection.
func TestIntegration_EndToEnd_Logs(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		response := map[string]any{
			"logs": []map[string]any{
				{"id": "1", "level": "info", "message": "test log 1", "timestamp": time.Now().Format(time.RFC3339)},
				{"id": "2", "level": "error", "message": "test log 2", "timestamp": time.Now().Format(time.RFC3339)},
			},
			"total": 2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:           server.URL,
		ResponseField: "logs",
		AuthMode:      authModeAPIKey,
		APIKeyConfig: APIKeyConfig{
			HeaderName: "X-API-Key",
			Value:      "test-key",
		},
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		MaxPollInterval: 100 * time.Millisecond,
		ClientConfig:    confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait for multiple poll cycles
	time.Sleep(300 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify data was collected
	allLogs := sink.AllLogs()
	require.Greater(t, len(allLogs), 0)

	// Verify multiple requests were made
	require.Greater(t, requestCount, 1)
}

// TestIntegration_EndToEnd_Metrics tests a complete end-to-end scenario for metrics collection.
func TestIntegration_EndToEnd_Metrics(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		response := []map[string]any{
			{"metric": "cpu_usage", "value": 75.5, "timestamp": time.Now().Format(time.RFC3339)},
			{"metric": "memory_usage", "value": 60.2, "timestamp": time.Now().Format(time.RFC3339)},
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
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		MaxPollInterval: 100 * time.Millisecond,
		ClientConfig:    confighttp.ClientConfig{},
		Metrics: MetricsConfig{
			NameField: "metric",
		},
	}

	sink := new(consumertest.MetricsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPIMetricsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait for multiple poll cycles
	time.Sleep(300 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify data was collected
	allMetrics := sink.AllMetrics()
	require.Greater(t, len(allMetrics), 0)

	// Verify multiple requests were made
	require.Greater(t, requestCount, 1)
}

// TestIntegration_WithPaginationAndAuth tests a complete scenario with pagination and authentication.
func TestIntegration_WithPaginationAndAuth(t *testing.T) {
	pageCount := 0
	expectedAuthHeader := "Bearer test-token-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication
		authHeader := r.Header.Get("Authorization")
		require.Equal(t, expectedAuthHeader, authHeader)

		offset := r.URL.Query().Get("offset")
		_ = r.URL.Query().Get("limit") // limit parameter

		var response map[string]any
		if offset == "0" || offset == "" {
			response = map[string]any{
				"data": []map[string]any{
					{"id": "1", "event": "event1"},
					{"id": "2", "event": "event2"},
				},
				"total": 4,
			}
		} else if offset == "2" {
			response = map[string]any{
				"data": []map[string]any{
					{"id": "3", "event": "event3"},
					{"id": "4", "event": "event4"},
				},
				"total": 4,
			}
		} else {
			response = map[string]any{
				"data":  []map[string]any{},
				"total": 4,
			}
		}

		pageCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:           server.URL,
		ResponseField: "data",
		AuthMode:      authModeBearer,
		BearerConfig: BearerConfig{
			Token: "test-token-123",
		},
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				OffsetFieldName: "offset",
				LimitFieldName:  "limit",
				StartingOffset:  0,
			},
			TotalRecordCountField: "total",
		},
		MaxPollInterval: 100 * time.Millisecond,
		ClientConfig:    confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait for at least one poll cycle (which will fetch all pages)
	time.Sleep(200 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify data was collected from multiple pages
	allLogs := sink.AllLogs()
	require.Greater(t, len(allLogs), 0)

	// Count total log records
	totalRecords := 0
	for _, logs := range allLogs {
		totalRecords += logs.LogRecordCount()
	}
	// Should have received logs from multiple pages (at least 2 pages = 4 records)
	require.GreaterOrEqual(t, totalRecords, 4)
}

// TestIntegration_TimestampPagination tests timestamp-based pagination.
func TestIntegration_TimestampPagination(t *testing.T) {
	var lastTimestamp string
	var pageSize string
	pageCount := 0
	initialTime := time.Now().Add(-1 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the timestamp and page size parameters
		lastTimestamp = r.URL.Query().Get("t0")
		pageSize = r.URL.Query().Get("perPage")

		var response []map[string]any
		if pageCount == 0 {
			// First page - return full page
			response = []map[string]any{
				{"id": "1", "message": "test1", "ts": time.Now().Add(-30 * time.Minute).Format(time.RFC3339)},
				{"id": "2", "message": "test2", "ts": time.Now().Add(-20 * time.Minute).Format(time.RFC3339)},
			}
		} else {
			// Second page - return partial page to stop pagination
			response = []map[string]any{
				{"id": "3", "message": "test3", "ts": time.Now().Add(-10 * time.Minute).Format(time.RFC3339)},
			}
		}
		pageCount++
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
		Pagination: PaginationConfig{
			Mode: paginationModeTimestamp,
			Timestamp: TimestampPagination{
				ParamName:          "t0",
				TimestampFieldName: "ts",
				PageSizeFieldName:  "perPage",
				PageSize:           200,
				InitialTimestamp:   initialTime,
			},
		},
		MaxPollInterval: 100 * time.Millisecond,
		ClientConfig:    confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait for a poll cycle (which will fetch all pages)
	time.Sleep(200 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify timestamp parameter was used
	require.NotEmpty(t, lastTimestamp)
	require.Contains(t, lastTimestamp, "T") // RFC3339 format check
	// Verify page size parameter was used
	require.Equal(t, "200", pageSize)
	// Verify multiple pages were fetched
	require.Greater(t, pageCount, 1)
}

// TestIntegration_ErrorRecovery tests that the receiver continues polling after errors.
func TestIntegration_ErrorRecovery(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request returns error
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
			return
		}
		// Subsequent requests succeed
		response := []map[string]any{
			{"id": "1", "message": "success"},
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
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		MaxPollInterval: 100 * time.Millisecond,
		ClientConfig:    confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait for multiple poll cycles
	time.Sleep(300 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify receiver continued polling after error
	require.Greater(t, requestCount, 1)

	// Verify some data was eventually collected
	allLogs := sink.AllLogs()
	require.Greater(t, len(allLogs), 0)
}
