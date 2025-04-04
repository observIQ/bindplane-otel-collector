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

package azureloganalyticsexporter

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
	// Set timestamp for consistent testing
	lr.SetTimestamp(pcommon.NewTimestampFromTime(testTime))
	return lr
}
func TestMarshalRawLogs(t *testing.T) {
	tests := []struct {
		name        string
		logRecords  []plog.LogRecord
		expected    []map[string]interface{}
		rawLogField string
		wantErr     bool
	}{
		{
			name: "Simple log record",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}),
			},
			expected: []map[string]interface{}{
				{
					"key1": "value1",
					"body": "Test body",
				},
			},
			rawLogField: `{"body": body, "key1": attributes["key1"]}`,
			wantErr:     false,
		},
		{
			name: "Nested body log record",
			logRecords: []plog.LogRecord{
				mockLogRecordWithNestedBody(map[string]any{"nested": "value"}),
			},
			expected: []map[string]interface{}{
				{
					"nested": "value",
				},
			},
			rawLogField: `body`,
			wantErr:     false,
		},
		{
			name: "String body log record",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "test", map[string]any{}),
			},
			expected: []map[string]interface{}{
				{
					"RawData": "test",
				},
			},
			rawLogField: `body`,
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
			expected: []map[string]interface{}{
				{
					"RawData": "info",
				},
			},
			rawLogField: `attributes["level"]`,
			wantErr:     false,
		},
		{
			name: "Valid rawLogField - nested attribute",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test body", map[string]any{"event": map[string]any{"type": "login"}}),
			},
			expected: []map[string]interface{}{
				{
					"type": "login",
				},
			},
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
		{
			name: "String body with newline character",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test \nbody", map[string]any{"key1": "value1"}),
			},
			expected: []map[string]interface{}{
				{
					"RawData": "Test \\nbody",
				},
			},
			rawLogField: `body`,
			wantErr:     false,
		},
		{
			name: "Does not affect already escaped newline characters in string body",
			logRecords: []plog.LogRecord{
				mockLogRecord(t, "Test\\n \nbody", map[string]any{"key1": "value1"}),
			},
			expected: []map[string]interface{}{
				{
					"RawData": "Test\\n \\nbody",
				},
			},
			rawLogField: `body`,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, observedLogs := observer.New(zap.InfoLevel)
			logger := zap.New(core)
			cfg := &Config{RawLogField: tt.rawLogField}
			m := newMarshaler(cfg, component.TelemetrySettings{Logger: logger})

			logs := mockLogs(tt.logRecords...)
			marshalledBytes, err := m.transformLogsToSentinelFormat(context.Background(), logs)

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
				require.NoError(t, err, "Did not expect an error to be returned")

				// Parse the resulting JSON
				var result []map[string]interface{}
				err = json.Unmarshal(marshalledBytes, &result)
				require.NoError(t, err, "Failed to unmarshal result JSON")

				// Compare with expected
				require.Equal(t, len(tt.expected), len(result),
					"Expected %d marshalled logs, got %d", len(tt.expected), len(result))

				for i, expected := range tt.expected {
					for k, v := range expected {
						require.Equal(t, v, result[i][k],
							"Field %s doesn't match in log %d", k, i)
					}
				}
			}
		})
	}
}

func TestTransformLogsToSentinelFormat(t *testing.T) {
	// Create test logs
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()

	// Add resource attributes
	rl.Resource().Attributes().PutStr("service.name", "test-service")
	rl.Resource().Attributes().PutStr("host.name", "test-host")

	// Add scope logs
	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("test-scope")
	sl.Scope().SetVersion("v1.0.0")

	// Add log record
	lr := sl.LogRecords().AppendEmpty()
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)))
	lr.SetSeverityText("INFO")
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.Body().SetStr("Test log message")
	lr.Attributes().PutStr("log_type", "test-log")

	// Create logger for tests
	logger := zap.NewNop()

	t.Run("Default transformation", func(t *testing.T) {
		// Create config with default settings (no raw log field)
		cfg := &Config{}

		// Create telemetry settings for tests
		telemetrySettings := component.TelemetrySettings{
			Logger: logger,
		}

		// Create marshaler with default config
		marshaler := newMarshaler(cfg, telemetrySettings)

		// Transform logs
		jsonBytes, err := marshaler.transformLogsToSentinelFormat(context.Background(), logs)
		assert.NoError(t, err)

		// Parse the resulting JSON
		var result []map[string]interface{}
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)

		// Verify the structure
		assert.Len(t, result, 1)
		assert.Contains(t, result[0], "resourceLogs")

		// Check that there's no nested resourceLogs
		resourceLogs, ok := result[0]["resourceLogs"].([]interface{})
		assert.True(t, ok, "resourceLogs should be an array, not an object")

		// Verify the content
		assert.Len(t, resourceLogs, 1)
		rlMap := resourceLogs[0].(map[string]interface{})

		// Check resource attributes
		resource := rlMap["resource"].(map[string]interface{})
		attributes := resource["attributes"].([]interface{})

		// Helper function to find attribute by key
		findAttr := func(key string) interface{} {
			for _, attr := range attributes {
				attrMap := attr.(map[string]interface{})
				if attrMap["key"] == key {
					return attrMap["value"]
				}
			}
			return nil
		}

		serviceAttr := findAttr("service.name")
		assert.Equal(t, "test-service", serviceAttr.(map[string]interface{})["stringValue"])

		hostAttr := findAttr("host.name")
		assert.Equal(t, "test-host", hostAttr.(map[string]interface{})["stringValue"])

		// Check log record
		scopeLogs := rlMap["scopeLogs"].([]interface{})
		assert.Len(t, scopeLogs, 1)

		logRecords := scopeLogs[0].(map[string]interface{})["logRecords"].([]interface{})
		assert.Len(t, logRecords, 1)

		logRecord := logRecords[0].(map[string]interface{})
		assert.Equal(t, "Test log message", logRecord["body"].(map[string]interface{})["stringValue"])
		assert.Equal(t, "INFO", logRecord["severityText"])
	})

	t.Run("Raw log field transformation using attributes", func(t *testing.T) {
		// Create config with raw log field
		cfg := &Config{
			RawLogField: `attributes`,
		}

		// Create telemetry settings for tests
		telemetrySettings := component.TelemetrySettings{
			Logger: logger,
		}

		// Create marshaler with raw log field config
		marshaler := newMarshaler(cfg, telemetrySettings)

		// Transform logs
		jsonBytes, err := marshaler.transformLogsToSentinelFormat(context.Background(), logs)
		assert.NoError(t, err)

		// Parse the resulting JSON
		var result []map[string]interface{}
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)

		// Verify the structure
		assert.Len(t, result, 1)
		assert.Contains(t, result[0], "log_type")

		// Verify the content
		assert.Equal(t, "test-log", result[0]["log_type"])
	})

	t.Run("Raw log field transformation using body", func(t *testing.T) {
		// Create config with raw log field set to extract body
		cfg := &Config{
			RawLogField: `body`,
		}

		// Create telemetry settings for tests
		telemetrySettings := component.TelemetrySettings{
			Logger: logger,
		}

		// Create marshaler with raw log field config
		marshaler := newMarshaler(cfg, telemetrySettings)

		// Transform logs
		jsonBytes, err := marshaler.transformLogsToSentinelFormat(context.Background(), logs)
		assert.NoError(t, err)

		// Parse the resulting JSON
		var result []map[string]interface{}
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)

		// Verify the structure
		assert.Len(t, result, 1)
		assert.Contains(t, result[0], "RawData")

		// Verify the content
		assert.Equal(t, "Test log message", result[0]["RawData"])
	})

	t.Run("Raw log field transformation using custom expression", func(t *testing.T) {
		// Create a custom expression that returns a combination of values
		cfg := &Config{
			RawLogField: `{"message": body, "log_level": severity_text, "hostname": resource.attributes["host.name"]}`,
		}

		// Create telemetry settings for tests
		telemetrySettings := component.TelemetrySettings{
			Logger: logger,
		}

		// Create marshaler with raw log field config
		marshaler := newMarshaler(cfg, telemetrySettings)

		// Transform logs
		jsonBytes, err := marshaler.transformLogsToSentinelFormat(context.Background(), logs)
		assert.NoError(t, err)

		// Parse the resulting JSON
		var result []map[string]interface{}
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)

		// Verify the structure
		assert.Len(t, result, 1)
		assert.Contains(t, result[0], "message")
		assert.Contains(t, result[0], "log_level")
		assert.Contains(t, result[0], "hostname")

		// Verify the content
		assert.Equal(t, "Test log message", result[0]["message"])
		assert.Equal(t, "INFO", result[0]["log_level"])
		assert.Equal(t, "test-host", result[0]["hostname"])
	})
}
