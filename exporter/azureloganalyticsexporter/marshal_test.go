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
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

var testTime = time.Date(2023, 1, 2, 3, 4, 5, 6, time.UTC)

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

	// Create expected JSON using standard marshaler for comparison
	standardMarshaler := plog.JSONMarshaler{}
	expectedJSON, err := standardMarshaler.MarshalLogs(logs)
	assert.NoError(t, err)

	t.Run("Standard JSON marshaling", func(t *testing.T) {
		// Create config with default settings (no raw log field)
		cfg := &Config{}

		// Create telemetry settings for tests
		telemetrySettings := component.TelemetrySettings{
			Logger: logger,
		}

		// Create marshaler with default config
		marshaler := newMarshaler(cfg, telemetrySettings)

		wrappedData := append([]byte{'['}, append(expectedJSON, ']')...)
		// Transform logs
		jsonBytes, err := marshaler.transformLogsToSentinelFormat(context.Background(), logs)
		assert.NoError(t, err)

		// Compare with expected output from standard marshaler
		assert.JSONEq(t, string(wrappedData), string(jsonBytes))
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
		assert.Contains(t, result[0], "RawData")

		// Parse the RawData field which should contain the attributes
		var rawData map[string]interface{}
		err = json.Unmarshal([]byte(result[0]["RawData"].(string)), &rawData)
		assert.NoError(t, err)

		// Verify the content
		assert.Contains(t, rawData, "log_type")
		assert.Equal(t, "test-log", rawData["log_type"])
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

		// Verify the content is directly the log message (not in JSON format)
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
		assert.Contains(t, result[0], "RawData")

		// Parse the RawData field which should contain the custom JSON
		var rawData map[string]interface{}
		err = json.Unmarshal([]byte(result[0]["RawData"].(string)), &rawData)
		assert.NoError(t, err)

		// Verify the content
		assert.Contains(t, rawData, "message")
		assert.Contains(t, rawData, "log_level")
		assert.Contains(t, rawData, "hostname")
		assert.Equal(t, "Test log message", rawData["message"])
		assert.Equal(t, "INFO", rawData["log_level"])
		assert.Equal(t, "test-host", rawData["hostname"])
	})
}
