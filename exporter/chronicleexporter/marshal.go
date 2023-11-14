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
	"time"

	"github.com/observiq/bindplane-agent/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// marshaler is an interface for marshalling logs.
//
//go:generate mockery --name logMarshaler --output ./internal/mocks --with-expecter --filename mock_marshaler.go --structname MockMarshaler
type logMarshaler interface {
	MarshalRawLogs(ctx context.Context, ld plog.Logs) ([]byte, error)
}

type marshaler struct {
	cfg          Config
	teleSettings component.TelemetrySettings
}

func newMarshaler(cfg Config, teleSettings component.TelemetrySettings) *marshaler {
	return &marshaler{
		cfg:          cfg,
		teleSettings: teleSettings,
	}
}

func (m *marshaler) MarshalRawLogs(ctx context.Context, ld plog.Logs) ([]byte, error) {
	rawLogs, err := m.extractRawLogs(ctx, ld)
	if err != nil {
		return nil, fmt.Errorf("extract raw logs: %w", err)
	}

	rawLogData := map[string]any{
		"entries":  rawLogs,
		"log_type": m.cfg.LogType,
	}

	if m.cfg.CustomerID != "" {
		rawLogData["customer_id"] = m.cfg.CustomerID
	}

	return json.Marshal(rawLogData)
}

type entry struct {
	LogText   string `json:"log_text"`
	Timestamp string `json:"ts_rfc3339"`
}

func (m *marshaler) extractRawLogs(ctx context.Context, ld plog.Logs) ([]entry, error) {
	entries := []entry{}

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				logRecord := scopeLog.LogRecords().At(k)

				var rawLog string
				var err error
				if m.cfg.RawLogField == "" {
					body := logRecord.Body().Str()
					entireLogRecord := map[string]any{
						"body":                body,
						"attributes":          logRecord.Attributes().AsRaw(),
						"resource_attributes": resourceLog.Resource().Attributes().AsRaw(),
					}

					bytesLogRecord, err := json.Marshal(entireLogRecord)
					if err != nil {
						return nil, fmt.Errorf("marshal log record: %w", err)
					}

					rawLog = string(bytesLogRecord)
				} else {
					rawLog, err = m.getRawField(ctx, logRecord, scopeLog.Scope(), resourceLog.Resource())
					if err != nil {
						m.teleSettings.Logger.Error("Error getting raw field", zap.Error(err))
						continue
					}
				}

				entries = append(entries, entry{
					LogText:   rawLog,
					Timestamp: logRecord.Timestamp().AsTime().Format(time.RFC3339Nano),
				})
			}
		}
	}

	return entries, nil
}

func (m *marshaler) getRawField(ctx context.Context, logRecord plog.LogRecord, scope pcommon.InstrumentationScope, resource pcommon.Resource) (string, error) {
	lrExpr, err := expr.NewOTTLLogRecordExpression(m.cfg.RawLogField, m.teleSettings)
	if err != nil {
		return "", fmt.Errorf("raw_log_field is invalid: %s", err)
	}
	tCtx := ottllog.NewTransformContext(logRecord, scope, resource)

	lrExprResult, err := lrExpr.Execute(ctx, tCtx)
	if err != nil {
		return "", fmt.Errorf("execute log record expression: %w", err)
	}

	if lrExprResult == nil {
		return "", fmt.Errorf("log record expression result is nil")
	}

	switch lrExprResult.(type) {
	case string:
		return lrExprResult.(string), nil
	case pcommon.Map:
		bytes, err := json.Marshal(lrExprResult.(pcommon.Map).AsRaw())
		if err != nil {
			return "", fmt.Errorf("marshal log record expression result: %w", err)
		}
		return string(bytes), nil
	}

	return "", fmt.Errorf("unsupported log record expression result type: %T", lrExprResult)
}
