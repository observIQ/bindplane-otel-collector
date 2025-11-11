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

	"github.com/observiq/bindplane-otel-collector/receiver/restapireceiver/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestRESTAPILogsReceiver_StartShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := []map[string]any{
			{"id": "1", "message": "test"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:                  server.URL,
		AuthMode:             string(authModeAPIKey),
		AuthAPIKeyHeaderName: "X-API-Key",
		AuthAPIKeyValue:      "test-key",
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		PollInterval: 100 * time.Millisecond,
		ClientConfig: confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, receiver)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait a bit for polling
	time.Sleep(200 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Should have received some logs
	require.Greater(t, len(sink.AllLogs()), 0)
}

func TestRESTAPIMetricsReceiver_StartShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := []map[string]any{
			{"value": 42.0, "name": "test"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:                  server.URL,
		AuthMode:             string(authModeAPIKey),
		AuthAPIKeyHeaderName: "X-API-Key",
		AuthAPIKeyValue:      "test-key",
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		PollInterval: 100 * time.Millisecond,
		ClientConfig: confighttp.ClientConfig{},
	}

	sink := new(consumertest.MetricsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPIMetricsReceiver(params, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, receiver)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait a bit for polling
	time.Sleep(200 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Should have received some metrics
	require.Greater(t, len(sink.AllMetrics()), 0)
}

func TestRESTAPILogsReceiver_WithPagination(t *testing.T) {
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		_ = r.URL.Query().Get("limit") // limit parameter

		var response map[string]any
		if offset == "0" || offset == "" {
			response = map[string]any{
				"data": []map[string]any{
					{"id": "1", "message": "page1"},
					{"id": "2", "message": "page1"},
				},
				"total": 4,
			}
		} else if offset == "2" {
			response = map[string]any{
				"data": []map[string]any{
					{"id": "3", "message": "page2"},
					{"id": "4", "message": "page2"},
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
		URL:                  server.URL,
		ResponseField:        "data",
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
			TotalRecordCountField: "total",
		},
		PollInterval: 100 * time.Millisecond,
		ClientConfig: confighttp.ClientConfig{},
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

	// Should have received logs (from all pages in first poll cycle)
	allLogs := sink.AllLogs()
	require.Greater(t, len(allLogs), 0)

	// Count total log records across all batches
	totalRecords := 0
	for _, logs := range allLogs {
		totalRecords += logs.LogRecordCount()
	}
	// Should have received logs from multiple pages (at least 2 pages = 4 records)
	require.GreaterOrEqual(t, totalRecords, 4)
}

func TestRESTAPILogsReceiver_WithTimeBasedOffset(t *testing.T) {
	var lastTimestamp string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the timestamp parameter
		lastTimestamp = r.URL.Query().Get("since")

		response := []map[string]any{
			{"id": "1", "message": "test", "timestamp": time.Now().Format(time.RFC3339)},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	initialTime := time.Now().Add(-1 * time.Hour)
	cfg := &Config{
		URL:                  server.URL,
		AuthMode:             string(authModeAPIKey),
		AuthAPIKeyHeaderName: "X-API-Key",
		AuthAPIKeyValue:      "test-key",
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		TimeBasedOffset: TimeBasedOffsetConfig{
			Enabled:         true,
			ParamName:       "since",
			OffsetTimestamp: initialTime,
		},
		PollInterval: 100 * time.Millisecond,
		ClientConfig: confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait for a poll
	time.Sleep(200 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Should have used the timestamp parameter
	require.NotEmpty(t, lastTimestamp)
}

func TestRESTAPILogsReceiver_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &Config{
		URL:                  server.URL,
		AuthMode:             string(authModeAPIKey),
		AuthAPIKeyHeaderName: "X-API-Key",
		AuthAPIKeyValue:      "test-key",
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		PollInterval: 100 * time.Millisecond,
		ClientConfig: confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait a bit - should handle errors gracefully
	time.Sleep(200 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Receiver should still be running (errors logged but don't crash)
}

func TestRESTAPILogsReceiver_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := []map[string]any{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &Config{
		URL:                  server.URL,
		AuthMode:             string(authModeAPIKey),
		AuthAPIKeyHeaderName: "X-API-Key",
		AuthAPIKeyValue:      "test-key",
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
		PollInterval: 100 * time.Millisecond,
		ClientConfig: confighttp.ClientConfig{},
	}

	sink := new(consumertest.LogsSink)
	params := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newRESTAPILogsReceiver(params, cfg, sink)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	ctx := context.Background()

	err = receiver.Start(ctx, host)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Empty responses should be handled gracefully
	// May or may not have logs depending on implementation
}
