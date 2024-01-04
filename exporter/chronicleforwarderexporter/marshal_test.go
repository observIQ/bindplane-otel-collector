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

package chronicleforwarderexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
		case int:
			lr.Attributes().PutInt(k, int64(v.(int)))
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
		name        string
		logRecords  []plog.LogRecord
		expected    []string
		rawLogField string
		wantErr     bool
	}{
		{
			name: "Simple log record",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expected:    []string{`{"attributes":{"key1":"value1"},"body":"Test body","resource_attributes":{}}`},
			rawLogField: "",
			wantErr:     false,
		},
		{
			name: "Nested body log record",
			logRecords: []plog.LogRecord{
				mockLogRecordWithNestedBody(map[string]any{"nested": "value"}),
			},
			expected:    []string{`{"attributes":{},"body":{"nested":"value"},"resource_attributes":{}}`},
			rawLogField: "",
			wantErr:     false,
		},
		{
			name: "String body log record",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "test", map[string]any{}),
			},
			expected:    []string{`{"attributes":{},"body":"test","resource_attributes":{}}`},
			rawLogField: "",
			wantErr:     false,
		},
		{
			name: "Invalid raw log field",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expected:    nil,
			rawLogField: "invalid_field",
			wantErr:     true,
		},
		{
			name: "Valid rawLogField - simple attribute",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"level": "info"}),
			},
			expected:    []string{"info"},
			rawLogField: `attributes["level"]`,
			wantErr:     false,
		},
		{
			name: "Valid rawLogField - nested attribute",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"event": map[string]any{"type": "login"}}),
			},
			expected:    []string{`{"type":"login"}`},
			rawLogField: `attributes["event"]`,
			wantErr:     false,
		},
		{
			name: "Invalid rawLogField - non-existent field",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expected:    nil,
			rawLogField: `attributes["nonexistent"]`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, observedLogs := observer.New(zap.InfoLevel)
			logger := zap.New(core)
			cfg := Config{RawLogField: tt.rawLogField}
			m := newMarshaler(cfg, component.TelemetrySettings{Logger: logger})

			logs := mockLogs(tt.logRecords...)
			marshalledLogs, err := m.MarshalRawLogs(context.Background(), logs)
			require.NoError(t, err)

			// Check for errors in the logs
			var foundError bool
			for _, log := range observedLogs.All() {
				if log.Level == zap.ErrorLevel {
					foundError = true
					break
				}
			}

			if tt.wantErr {
				require.True(t, foundError, "Expected an error to be logged")
			} else {
				require.False(t, foundError, "Did not expect an error to be logged")

				// Directly compare the marshalled strings with the expected strings
				if len(tt.expected) == len(marshalledLogs) {
					for i, expected := range tt.expected {
						require.Equal(t, expected, marshalledLogs[i], "Marshalled log does not match expected")
					}
				} else {
					t.Errorf("Expected %d marshalled logs, got %d", len(tt.expected), len(marshalledLogs))
				}
			}
		})
	}
}
