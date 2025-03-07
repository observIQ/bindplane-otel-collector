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

package bindplaneauditlogs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

type mockHTTPClient struct {
	mockDo func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.mockDo(req)
}

func (m *mockHTTPClient) CloseIdleConnections() {}

func TestStartAndShutdown(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.BindplaneURLString = "https://localhost:3000"
	cfg.APIKey = "testkey"

	recv := newReceiver(t, cfg, consumertest.NewNop())

	err := recv.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	err = recv.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestGetLogs(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.BindplaneURLString = "https://localhost:3000"
	cfg.BindplaneURL = url.URL{
		Scheme: "https",
		Host:   "localhost:3000",
	}
	cfg.APIKey = "testkey"

	recv := newReceiver(t, cfg, consumertest.NewNop())

	now := time.Now().UTC()
	older := now.Add(-1 * time.Hour)
	newest := now.Add(1 * time.Hour)

	testResponse := apiResponse{
		AuditEvents: []AuditLogEvent{
			{
				ID:            "1",
				Timestamp:     &older,
				ResourceName:  "source-1",
				Description:   "Created source configuration",
				ResourceKind:  "Source",
				Configuration: "logging",
				Action:        "Created",
				User:          "admin",
				Account:       "default",
			},
			{
				ID:            "2",
				Timestamp:     &now,
				ResourceName:  "destination-1",
				Description:   "Updated destination configuration",
				ResourceKind:  "Destination",
				Configuration: "otlp",
				Action:        "Updated",
				User:          "admin",
				Account:       "default",
			},
			{
				ID:            "3",
				Timestamp:     &newest,
				ResourceName:  "processor-1",
				Description:   "Deleted processor configuration",
				ResourceKind:  "Processor",
				Configuration: "metrics",
				Action:        "Deleted",
				User:          "admin",
				Account:       "default",
			},
		},
	}

	responseBody, err := json.Marshal(testResponse)
	require.NoError(t, err)

	recv.client = &mockHTTPClient{
		mockDo: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(responseBody))),
			}, nil
		},
	}

	logs := recv.getLogs(context.Background())

	// Verify logs are sorted newest first
	require.Equal(t, 3, len(logs))
	require.Equal(t, "3", logs[0].ID)
	require.Equal(t, "2", logs[1].ID)
	require.Equal(t, "1", logs[2].ID)

	// Verify content of newest log
	newestLog := logs[0]
	require.Equal(t, "processor-1", newestLog.ResourceName)
	require.Equal(t, "Processor", string(newestLog.ResourceKind))
	require.Equal(t, "Deleted", string(newestLog.Action))
	require.Equal(t, "metrics", newestLog.Configuration)

	require.Equal(t, newest, recv.lastTimestamp)
}

func TestGetLogsErrorHandling(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.BindplaneURLString = "https://localhost:3000"
	cfg.APIKey = "testkey"

	tests := []struct {
		name       string
		setupMock  func() httpClient
		wantLength int
	}{
		{
			name: "bad request error",
			setupMock: func() httpClient {
				return &mockHTTPClient{
					mockDo: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusBadRequest,
							Body:       io.NopCloser(strings.NewReader("")),
						}, nil
					},
				}
			},
			wantLength: 0,
		},
		{
			name: "invalid json response",
			setupMock: func() httpClient {
				return &mockHTTPClient{
					mockDo: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("invalid json")),
						}, nil
					},
				}
			},
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recv := newReceiver(t, cfg, consumertest.NewNop())
			recv.client = tt.setupMock()

			logs := recv.getLogs(context.Background())
			require.Equal(t, tt.wantLength, len(logs))
		})
	}
}

func TestProcessLogEvents(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	recv := newReceiver(t, cfg, consumertest.NewNop())

	now := time.Now().UTC()
	testEvents := []AuditLogEvent{
		{
			ID:            "1",
			Timestamp:     &now,
			ResourceName:  "test-resource",
			Description:   "test description",
			ResourceKind:  "Source",
			Configuration: "test-config",
			Action:        "Created",
			User:          "test-user",
			Account:       "test-account",
		},
	}

	logs := recv.processLogEvents(pcommon.NewTimestampFromTime(now), testEvents)

	require.Equal(t, 1, logs.LogRecordCount())
	logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Verify attributes
	attrs := logRecord.Attributes()
	id, _ := attrs.Get("id")
	require.Equal(t, "1", id.Str())
	resourceName, _ := attrs.Get("resource_name")
	require.Equal(t, "test-resource", resourceName.Str())
}

func newReceiver(t *testing.T, cfg *Config, c consumer.Logs) *bindplaneAuditLogsReceiver {
	r, err := newBindplaneAuditLogsReceiver(cfg, zap.NewNop(), c)
	require.NoError(t, err)
	return r
}
