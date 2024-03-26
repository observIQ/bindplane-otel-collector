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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-agent/exporter/chronicleexporter/protos/api"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtoMarshaler_MarshalRawLogs(t *testing.T) {
	logger := zap.NewNop()
	startTime := time.Now()

	tests := []struct {
		name         string
		cfg          Config
		labels       []*api.Label
		logRecords   func() plog.Logs
		expectations func(t *testing.T, requests []*api.BatchCreateLogsRequest)
	}{
		{
			name: "Single log record with expected data",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			labels: []*api.Label{
				{Key: "env", Value: "prod"},
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", map[string]any{"log_type": "WINEVTLOG"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "WINEVTLOG", batch.LogType)
				require.Len(t, batch.Entries, 1)

				// Convert Data (byte slice) to string for comparison
				logDataAsString := string(batch.Entries[0].Data)
				expectedLogData := `Test log message`
				require.Equal(t, expectedLogData, logDataAsString)

				require.NotNil(t, batch.StartTime)
				require.True(t, timestamppb.New(startTime).AsTime().Equal(batch.StartTime.AsTime()), "Start time should be set correctly")
			},
		},
		{
			name: "Multiple log records",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			labels: []*api.Label{
				{Key: "env", Value: "staging"},
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1, "Expected a single batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 2, "Expected two log entries in the batch")
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(batch.Entries[0].Data))
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(batch.Entries[1].Data))
			},
		},
		{
			name: "Log record with attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				RawLogField:     "attributes",
				OverrideLogType: false,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 1)

				// Assuming the attributes are marshaled into the Data field as a JSON string
				expectedData := `{"key1":"value1", "log_type":"WINEVTLOG"}`
				actualData := string(batch.Entries[0].Data)
				require.JSONEq(t, expectedData, actualData, "Log attributes should match expected")
			},
		},
		{
			name: "No log records",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "DEFAULT",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return plog.NewLogs() // No log records added
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 0, "Expected no requests due to no log records")
			},
		},
		{
			name: "Override log type with attribute",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "DEFAULT", // This should be overridden by the log_type attribute
				RawLogField:     "body",
				OverrideLogType: true,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"log_type": "windows_event.custom"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "WINEVTLOG", batch.LogType, "Expected log type to be overridden by attribute")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			marshaler, err := newProtoMarshaler(tt.cfg, component.TelemetrySettings{Logger: logger}, tt.labels)
			marshaler.startTime = startTime
			require.NoError(t, err)

			logs := tt.logRecords()
			requests, err := marshaler.MarshalRawLogs(context.Background(), logs)
			require.NoError(t, err)

			tt.expectations(t, requests)
		})
	}
}

func mockLogRecord(body string, attributes map[string]any) plog.LogRecord {
	lr := plog.NewLogRecord()
	lr.Body().SetStr(body)
	for k, v := range attributes {
		switch val := v.(type) {
		case string:
			lr.Attributes().PutStr(k, val)
		default:
		}
	}
	return lr
}

func mockLogs(record plog.LogRecord) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	record.CopyTo(sl.LogRecords().AppendEmpty())
	return logs
}
