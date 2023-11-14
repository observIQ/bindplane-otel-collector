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

package chronicleexporter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

var testTime = time.Date(2023, 1, 2, 3, 4, 5, 6, time.UTC)

// mockLogRecord creates a simple mock plog.LogRecord for testing.
func mockLogRecord(t *testing.T, body string, attributes map[string]any) plog.LogRecord {
	lr := plog.NewLogRecord()
	lr.Body().SetStr(body)
	lr.Attributes().EnsureCapacity(len(attributes))
	lr.SetTimestamp(pcommon.NewTimestampFromTime(testTime))
	for k, v := range attributes {
		switch v.(type) {
		case string:
			lr.Attributes().PutStr(k, v.(string))
		case map[string]any:
			lr.Attributes().FromRaw(attributes)
		default:
			t.Fatalf("unexpected attribute type: %T", v)
		}
	}
	return lr
}

// mockLogs creates mock plog.Logs with the given records.
func mockLogs(records ...plog.LogRecord) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	for _, rec := range records {
		rec.CopyTo(sl.LogRecords().AppendEmpty())
	}
	return logs
}

// mockLogRecordWithNestedBody creates a log record with a nested body structure.
func mockLogRecordWithNestedBody(body map[string]any) plog.LogRecord {
	lr := plog.NewLogRecord()
	lr.Body().SetEmptyMap().EnsureCapacity(len(body))
	lr.Body().Map().FromRaw(body)
	return lr
}

func TestMarshalRawLogs(t *testing.T) {
	tests := []struct {
		name         string
		logRecords   []plog.LogRecord
		expectedJSON string
		logType      string
		rawLogField  string
		errExpected  error
	}{
		{
			name: "Single log record",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"Test body","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  "body",
		},
		{
			name: "Multiple log records",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "First log", map[string]any{"key1": "value1"}),
				mockLogRecord(t, "Second log", map[string]any{"key2": "value2"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"First log","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"},{"log_text":"Second log","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  "body",
		},
		{
			name: "Log record with attributes",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1", "key2": "value2"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"{\"key1\":\"value1\",\"key2\":\"value2\"}","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  "attributes",
		},
		{
			name: "Nested rawLogField",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"nested": map[string]any{"key": "value"}}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"value","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  `attributes["nested"]["key"]`,
		},
		{
			name:         "Empty log record",
			logRecords:   []plog.LogRecord{},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  "body",
		},
		{
			name: "Log record with missing field",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  `attributes["missing"]`,
			// No error expected because the record will be dropped.
		},
		{
			name: "Nested body field",
			logRecords: []plog.LogRecord{
				mockLogRecordWithNestedBody(map[string]any{
					"event": map[string]any{
						"type": "login_attempt",
						"details": map[string]any{
							"username": "user123",
							"ip":       "192.168.1.1",
						},
					},
				}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"{\"event\":{\"details\":{\"ip\":\"192.168.1.1\",\"username\":\"user123\"},\"type\":\"login_attempt\"}}","ts_rfc3339":"1970-01-01T00:00:00Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  "body",
		},
		{
			name: "use Nested body field",
			logRecords: []plog.LogRecord{
				mockLogRecordWithNestedBody(map[string]any{
					"event": map[string]any{
						"type": "login_attempt",
						"details": map[string]any{
							"username": "user123",
							"ip":       "192.168.1.1",
						},
					},
				}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"user123","ts_rfc3339":"1970-01-01T00:00:00Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  `body["event"]["details"]["username"]`,
		},
		{
			name: "No rawLogField specified",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test log without raw field", map[string]any{"key1": "value1"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"{\"attributes\":{\"key1\":\"value1\"},\"body\":\"Test log without raw field\",\"resource_attributes\":{}}","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  "", // No rawLogField specified
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{LogType: tt.logType, RawLogField: tt.rawLogField, CustomerID: "test_customer_id"}
			ce := &chronicleExporter{
				cfg:       cfg,
				logger:    zap.NewNop(),
				marshaler: &marshaler{cfg: *cfg, teleSettings: component.TelemetrySettings{Logger: zap.NewNop()}},
			}

			logs := mockLogs(tt.logRecords...)
			result, err := ce.marshaler.MarshalRawLogs(context.Background(), logs)
			if tt.errExpected != nil {
				require.Error(t, err, "MarshalRawLogs should return an error")
				return
			}

			require.NoError(t, err, "MarshalRawLogs should not return an error")

			var resultJSON map[string]interface{}
			err = json.Unmarshal(result, &resultJSON)
			require.NoError(t, err, "Unmarshalling result should not produce an error")

			var expectedJSON map[string]interface{}
			err = json.Unmarshal([]byte(tt.expectedJSON), &expectedJSON)
			require.NoError(t, err, "Unmarshalling expected JSON should not produce an error")

			assert.Equal(t, tt.expectedJSON, string(result), "Marshalled JSON should match expected JSON")
		})
	}
}
