package microsoftsentinelexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/observiq/bindplane-otel-collector/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// sentinelMarshaler handles transforming logs for Microsoft Sentinel
type sentinelMarshaler struct {
	cfg          *Config
	teleSettings component.TelemetrySettings
	logger       *zap.Logger
}

// newMarshaler creates a new instance of the sentinel marshaler
func newMarshaler(cfg *Config, teleSettings component.TelemetrySettings) *sentinelMarshaler {
	return &sentinelMarshaler{
		cfg:          cfg,
		teleSettings: teleSettings,
		logger:       teleSettings.Logger,
	}
}

// getRawField extracts a field value using OTTL expression
func (m *sentinelMarshaler) getRawField(ctx context.Context, field string, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
	lrExpr, err := expr.NewOTTLLogRecordExpression(field, m.teleSettings)
	if err != nil {
		return "", fmt.Errorf("raw_log_field is invalid: %s", err)
	}
	tCtx := ottllog.NewTransformContext(logRecord, scope.Scope(), resource.Resource(), scope, resource)

	lrExprResult, err := lrExpr.Execute(ctx, tCtx)
	if err != nil {
		return "", fmt.Errorf("execute log record expression: %w", err)
	}

	if lrExprResult == nil {
		return "", nil
	}

	switch result := lrExprResult.(type) {
	case string:
		return result, nil
	case pcommon.Map:
		bytes, err := json.Marshal(result.AsRaw())
		if err != nil {
			return "", fmt.Errorf("marshal log record expression result: %w", err)
		}
		return string(bytes), nil
	default:
		return "", fmt.Errorf("unsupported log record expression result type: %T", lrExprResult)
	}
}

// transformLogsToSentinelFormat transforms logs to Microsoft Sentinel format
func (m *sentinelMarshaler) transformLogsToSentinelFormat(ctx context.Context, ld plog.Logs) ([]byte, error) {
	// Check if we're using raw log mode
	if m.cfg.RawLogField != "" {
		return m.transformRawLogsToSentinelFormat(ctx, ld)
	}

	// Default transformation logic (follows original code)
	var sentinelLogs []map[string]interface{}

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				logRecord := sl.LogRecords().At(k)

				// Convert the timestamp to RFC3339 format for TimeGenerated
				timeGenerated := logRecord.Timestamp().AsTime().Format(time.RFC3339)

				// Create a new single-record logs object to hold just this log
				singleLogRecord := plog.NewLogs()
				resourceLog := singleLogRecord.ResourceLogs().AppendEmpty()

				// Copy resource
				rl.Resource().CopyTo(resourceLog.Resource())

				// Copy scope
				scopeLog := resourceLog.ScopeLogs().AppendEmpty()
				sl.Scope().CopyTo(scopeLog.Scope())

				// Copy log record
				logRecordCopy := scopeLog.LogRecords().AppendEmpty()
				logRecord.CopyTo(logRecordCopy)

				// Use the JSONMarshaler to convert to JSON
				marshaler := plog.JSONMarshaler{}
				logJSON, err := marshaler.MarshalLogs(singleLogRecord)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal log to JSON: %w", err)
				}

				// Unmarshal the JSON to a map
				var logMap map[string]interface{}
				if err := json.Unmarshal(logJSON, &logMap); err != nil {
					return nil, fmt.Errorf("failed to unmarshal log JSON: %w", err)
				}

				// Extract the inner resourceLogs array directly
				innerResourceLogs, ok := logMap["resourceLogs"]
				if !ok {
					return nil, fmt.Errorf("unexpected structure: resourceLogs not found")
				}

				// Create the Sentinel format with TimeGenerated and the inner resourceLogs
				sentinelLog := map[string]interface{}{
					"TimeGenerated": timeGenerated,
					"resourceLogs":  innerResourceLogs, // Use the inner content directly
				}

				sentinelLogs = append(sentinelLogs, sentinelLog)
			}
		}
	}

	jsonLogs, err := json.Marshal(sentinelLogs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert logs to JSON: %w", err)
	}
	return jsonLogs, nil
}

// transformRawLogsToSentinelFormat transforms logs to Microsoft Sentinel format using the raw log approach
func (m *sentinelMarshaler) transformRawLogsToSentinelFormat(ctx context.Context, ld plog.Logs) ([]byte, error) {
	var sentinelLogs []map[string]interface{}

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)

		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)

			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				logRecord := scopeLog.LogRecords().At(k)

				// Escape any unescaped newlines (if body is a string)
				logBody := logRecord.Body()
				if logBody.Type() == pcommon.ValueTypeStr {
					logBody.SetStr(strings.ReplaceAll(logBody.AsString(), "\n", "\\n"))
				}

				// Extract raw log using the getRawField method
				rawLogStr, err := m.getRawField(ctx, m.cfg.RawLogField, logRecord, scopeLog, resourceLog)
				if err != nil {
					m.logger.Error("Error extracting raw log", zap.Error(err))
					continue
				}

				if rawLogStr == "" {
					m.logger.Error("Error processing log record: raw log is empty")
					continue
				}

				// Convert the timestamp to RFC3339 format for TimeGenerated
				timeGenerated := logRecord.Timestamp().AsTime().Format(time.RFC3339)

				// Try to parse rawLogStr as JSON first
				var rawLogData map[string]interface{}
				if err := json.Unmarshal([]byte(rawLogStr), &rawLogData); err != nil {
					// If not valid JSON, use as raw string
					sentinelLogs = append(sentinelLogs, map[string]interface{}{
						"TimeGenerated": timeGenerated,
						"RawData":       rawLogStr,
					})
				} else {
					// If valid JSON, merge with TimeGenerated
					rawLogData["TimeGenerated"] = timeGenerated
					sentinelLogs = append(sentinelLogs, rawLogData)
				}
			}
		}
	}

	jsonLogs, err := json.Marshal(sentinelLogs)
	fmt.Println(string(jsonLogs))
	if err != nil {
		return nil, fmt.Errorf("failed to convert logs to JSON: %w", err)
	}
	return jsonLogs, nil
}
