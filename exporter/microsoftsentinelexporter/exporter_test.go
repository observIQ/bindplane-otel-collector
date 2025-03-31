package microsoftsentinelexporter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestTransformLogsToSentinelFormat(t *testing.T) {
	// Create a new exporter instance
	exporter := &microsoftSentinelExporter{}

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

	// Transform logs
	jsonBytes, err := exporter.transformLogsToSentinelFormat(logs)
	assert.NoError(t, err)

	// Parse the resulting JSON
	var result []map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(t, err)

	// Verify the structure
	assert.Len(t, result, 1)
	assert.Contains(t, result[0], "TimeGenerated")
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
}
