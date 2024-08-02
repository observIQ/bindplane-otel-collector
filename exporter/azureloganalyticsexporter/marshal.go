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

package azureloganalyticsexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/observiq/bindplane-otel-collector/internal/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"

	"go.uber.org/zap"
)

// azureLogAnalyticsMarshaler handles transforming logs for Azure Log Analytics
type azureLogAnalyticsMarshaler struct {
	cfg          *Config
	teleSettings component.TelemetrySettings
	logger       *zap.Logger
}

// newMarshaler creates a new instance of the azureLogAnalyticsMarshaler
func newMarshaler(cfg *Config, teleSettings component.TelemetrySettings) *azureLogAnalyticsMarshaler {
	return &azureLogAnalyticsMarshaler{
		cfg:          cfg,
		teleSettings: teleSettings,
		logger:       teleSettings.Logger,
	}
}

// getRawField extracts a field value using OTTL expression
func (m *azureLogAnalyticsMarshaler) getRawField(ctx context.Context, field string, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
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
func (m *azureLogAnalyticsMarshaler) transformLogsToSentinelFormat(ctx context.Context, ld plog.Logs) ([]byte, error) {
	// Check if we're using raw log mode
	if m.cfg.RawLogField != "" {
		return m.transformRawLogsToAzureLogAnalyticsFormat(ctx, ld)
	}
	td := plogotlp.NewExportRequestFromLogs(ld)

	jsonData, err := td.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	wrappedData := append([]byte{'['}, append(jsonData, ']')...)
	return wrappedData, nil
}

// transformRawLogsToAzureLogAnalyticsFormat transforms logs to Azure Log Analytics format using the raw log approach
func (m *azureLogAnalyticsMarshaler) transformRawLogsToAzureLogAnalyticsFormat(ctx context.Context, ld plog.Logs) ([]byte, error) {
	var azureLogAnalyticsLogs []map[string]interface{}

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

				azureLogAnalyticsLogs = append(azureLogAnalyticsLogs, map[string]interface{}{
					"RawData": rawLogStr,
				})

			}
		}
	}

	jsonLogs, err := json.Marshal(azureLogAnalyticsLogs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert logs to JSON: %w", err)
	}
	return jsonLogs, nil
}
