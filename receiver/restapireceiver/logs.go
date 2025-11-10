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
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// convertJSONToLogs converts an array of JSON objects to plog.Logs.
// Each JSON object becomes one log record.
func convertJSONToLogs(data []map[string]any, logger *zap.Logger) plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, item := range data {
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		// Set observed timestamp
		logRecord.SetObservedTimestamp(now)

		// Set body as map from JSON object
		if err := logRecord.Body().SetEmptyMap().FromRaw(item); err != nil {
			logger.Warn("unable to set log body", zap.Error(err))
			// Fallback to string representation
			logRecord.Body().SetStr("failed to parse log body")
			continue
		}

		// Try to extract timestamp from common field names
		timestamp := extractTimestamp(item, logger)
		if timestamp > 0 {
			logRecord.SetTimestamp(timestamp)
		} else {
			// Use observed timestamp as fallback
			logRecord.SetTimestamp(now)
		}
	}

	return logs
}

// extractTimestamp attempts to extract a timestamp from the JSON object.
// It checks common field names like "timestamp", "time", "created_at", etc.
func extractTimestamp(item map[string]any, logger *zap.Logger) pcommon.Timestamp {
	// Common timestamp field names
	timestampFields := []string{"timestamp", "time", "created_at", "createdAt", "date", "datetime", "@timestamp"}

	for _, fieldName := range timestampFields {
		if val, exists := item[fieldName]; exists {
			if timestamp := parseTimestamp(val); timestamp > 0 {
				return timestamp
			}
		}
	}

	return 0
}

// parseTimestamp attempts to parse a timestamp from various formats.
func parseTimestamp(val any) pcommon.Timestamp {
	switch v := val.(type) {
	case string:
		// Try RFC3339 first
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return pcommon.NewTimestampFromTime(t)
		}
		// Try RFC3339Nano
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return pcommon.NewTimestampFromTime(t)
		}
		// Try Unix timestamp as string
		if t, err := time.Parse(time.UnixDate, v); err == nil {
			return pcommon.NewTimestampFromTime(t)
		}

	case float64:
		// Unix timestamp in seconds
		if v > 0 {
			return pcommon.NewTimestampFromTime(time.Unix(int64(v), 0))
		}

	case int64:
		// Unix timestamp in seconds
		if v > 0 {
			return pcommon.NewTimestampFromTime(time.Unix(v, 0))
		}

	case int:
		// Unix timestamp in seconds
		if v > 0 {
			return pcommon.NewTimestampFromTime(time.Unix(int64(v), 0))
		}
	}

	return 0
}
