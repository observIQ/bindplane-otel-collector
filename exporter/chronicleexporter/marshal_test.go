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
	"fmt"
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
		switch attribute := v.(type) {
		case string:
			lr.Attributes().PutStr(k, attribute)
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
		name            string
		logRecords      []plog.LogRecord
		expectedJSON    string
		logType         string
		rawLogField     string
		errExpected     error
		overrideLogType bool
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
			expectedJSON: ``, // No payload of logs expected
			logType:      "test_log_type",
			rawLogField:  "body",
		},
		{
			name: "Log record with missing field",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expectedJSON: ``, // No payload of logs expected
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
		{
			name: "Log type override",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Override log", map[string]any{"log_type": "windows_event.security"}),
			},
			expectedJSON:    `{"customer_id":"test_customer_id","entries":[{"log_text":"Override log","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"WINEVTLOG"}`,
			logType:         "default_log_type",
			rawLogField:     "body",
			overrideLogType: true,
		},
		{
			name: "Unsupported log type",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Unsupported log type", map[string]any{"log_type": "unsupported_type"}),
			},
			expectedJSON:    `{"customer_id":"test_customer_id","entries":[{"log_text":"Unsupported log type","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"default_log_type"}`,
			logType:         "default_log_type",
			rawLogField:     "body",
			overrideLogType: true,
		},
		{
			name: "Missing log type attribute",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Missing log type attribute", map[string]any{"key1": "value1"}),
			},
			expectedJSON:    `{"customer_id":"test_customer_id","entries":[{"log_text":"Missing log type attribute","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"default_log_type"}`,
			logType:         "default_log_type",
			rawLogField:     "body",
			overrideLogType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{LogType: tt.logType, RawLogField: tt.rawLogField, CustomerID: "test_customer_id", OverrideLogType: tt.overrideLogType}
			ce := &chronicleExporter{
				cfg:       cfg,
				logger:    zap.NewNop(),
				marshaler: &marshaler{cfg: *cfg, teleSettings: component.TelemetrySettings{Logger: zap.NewNop()}},
			}

			logs := mockLogs(tt.logRecords...)
			payloads, err := ce.marshaler.MarshalRawLogs(context.Background(), logs)
			if tt.errExpected != nil {
				require.Error(t, err, "MarshalRawLogs should return an error")
				return
			}

			require.NoError(t, err, "MarshalRawLogs should not return an error")

			// Merge payloads into a single JSON array for comparison
			results := make([]map[string]interface{}, 0, len(payloads))
			for _, p := range payloads {
				var resultJSON map[string]interface{}
				data, err := json.Marshal(p)
				require.NoError(t, err, "Marshalling result should not produce an error")
				err = json.Unmarshal(data, &resultJSON)
				require.NoError(t, err, "Unmarshalling result should not produce an error")
				results = append(results, resultJSON)
			}

			var expectedJSON []map[string]interface{}
			err = json.Unmarshal([]byte(fmt.Sprintf("[%s]", tt.expectedJSON)), &expectedJSON)
			require.NoError(t, err, "Unmarshalling expected JSON should not produce an error")

			assert.Equal(t, expectedJSON, results, "Marshalled JSON should match expected JSON")
		})
	}
}

func TestMarshalRawLogsWithLabels(t *testing.T) {
	tests := []struct {
		name            string
		logRecords      []plog.LogRecord
		expectedJSON    string
		logType         string
		rawLogField     string
		labels          []label
		overrideLogType bool
	}{
		{
			name: "Single log record with labels",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"Test body","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type","labels":[{"key":"env","value":"prod"},{"key":"region","value":"us-west"}]}`,
			logType:      "test_log_type",
			rawLogField:  "body",
			labels: []label{
				{"env", "prod"},
				{"region", "us-west"},
			},
		},
		{
			name: "Log Record with Multiple Labels",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Log with multiple labels", map[string]any{"key1": "value1"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"Log with multiple labels","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type","labels":[{"key":"env","value":"dev"},{"key":"service","value":"webapp"}]}`,
			logType:      "test_log_type",
			rawLogField:  "body",
			labels: []label{
				{"env", "dev"},
				{"service", "webapp"},
			},
		},
		{
			name: "Log Record with Empty Label Value",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Log with empty label value", map[string]any{}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"Log with empty label value","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type","labels":[{"key":"emptyLabel","value":""}]}`,
			logType:      "test_log_type",
			rawLogField:  "body",
			labels: []label{
				{"emptyLabel", ""},
			},
		},
		{
			name: "No Labels Provided",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "No labels", map[string]any{"key": "value"}),
			},
			expectedJSON: `{"customer_id":"test_customer_id","entries":[{"log_text":"No labels","ts_rfc3339":"2023-01-02T03:04:05.000000006Z"}],"log_type":"test_log_type"}`,
			logType:      "test_log_type",
			rawLogField:  "body",
			labels:       []label{},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				LogType:         tt.logType,
				RawLogField:     tt.rawLogField,
				CustomerID:      "test_customer_id",
				OverrideLogType: tt.overrideLogType,
			}
			ce := &chronicleExporter{
				cfg:       cfg,
				logger:    zap.NewNop(),
				marshaler: newMarshaler(*cfg, component.TelemetrySettings{Logger: zap.NewNop()}, tt.labels),
			}

			logs := mockLogs(tt.logRecords...)
			payloads, err := ce.marshaler.MarshalRawLogs(context.Background(), logs)

			require.NoError(t, err, "MarshalRawLogs should not return an error")

			// Merge payloads into a single JSON array for comparison
			results := make([]map[string]interface{}, 0, len(payloads))
			for _, p := range payloads {
				var resultJSON map[string]interface{}
				data, err := json.Marshal(p)
				require.NoError(t, err, "Marshalling result should not produce an error")
				err = json.Unmarshal(data, &resultJSON)
				require.NoError(t, err, "Unmarshalling result should not produce an error")
				results = append(results, resultJSON)
			}

			var expectedJSON []map[string]interface{}
			err = json.Unmarshal([]byte(fmt.Sprintf("[%s]", tt.expectedJSON)), &expectedJSON)
			require.NoError(t, err, "Unmarshalling expected JSON should not produce an error")

			assert.Equal(t, expectedJSON, results, "Marshalled JSON should match expected JSON")
		})
	}
}
