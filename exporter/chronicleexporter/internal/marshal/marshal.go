// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package marshal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/ccid"
	"github.com/observiq/bindplane-otel-collector/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

const logTypeField = `attributes["log_type"]`
const chronicleLogTypeField = `attributes["chronicle_log_type"]`
const chronicleNamespaceField = `attributes["chronicle_namespace"]`
const chronicleIngestionLabelsPrefix = `chronicle_ingestion_label`

var supportedLogTypes = map[string]string{
	"windows_event.security":    "WINEVTLOG",
	"windows_event.application": "WINEVTLOG",
	"windows_event.system":      "WINEVTLOG",
	"sql_server":                "MICROSOFT_SQL",
}

// Config is a subset of the HTTPConfig but if we ever identify a need for GRPC-specific config fields,
// then we should make it a shared unexported struct and embed it in both HTTPConfig and Config.
type Config struct {
	CustomerID            string
	Namespace             string
	LogType               string
	RawLogField           string
	OverrideLogType       bool
	IngestionLabels       map[string]string
	BatchRequestSizeLimit int
	BatchLogCountLimit    int
}

type protoMarshaler struct {
	cfg         Config
	set         component.TelemetrySettings
	startTime   time.Time
	customerID  []byte
	collectorID []byte
}

func newProtoMarshaler(cfg Config, set component.TelemetrySettings) (*protoMarshaler, error) {
	customerID, err := uuid.Parse(cfg.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("parse customer ID: %w", err)
	}
	return &protoMarshaler{
		startTime:   time.Now(),
		cfg:         cfg,
		set:         set,
		customerID:  customerID[:],
		collectorID: ccid.ChronicleCollectorID[:],
	}, nil
}

func (m *protoMarshaler) StartTime() time.Time {
	return m.startTime
}

func (m *protoMarshaler) getRawLog(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
	if m.cfg.RawLogField == "" {
		entireLogRecord := map[string]any{
			"body":                logRecord.Body().Str(),
			"attributes":          logRecord.Attributes().AsRaw(),
			"resource_attributes": resource.Resource().Attributes().AsRaw(),
		}

		bytesLogRecord, err := json.Marshal(entireLogRecord)
		if err != nil {
			return "", fmt.Errorf("marshal log record: %w", err)
		}

		return string(bytesLogRecord), nil
	}
	return GetRawField(ctx, m.set, m.cfg.RawLogField, logRecord, scope, resource)
}

func (m *protoMarshaler) getLogType(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
	// check for attributes in attributes["chronicle_log_type"]
	logType, err := GetRawField(ctx, m.set, chronicleLogTypeField, logRecord, scope, resource)
	if err != nil {
		return "", fmt.Errorf("get chronicle log type: %w", err)
	}
	if logType != "" {
		return logType, nil
	}

	if m.cfg.OverrideLogType {
		logType, err := GetRawField(ctx, m.set, logTypeField, logRecord, scope, resource)

		if err != nil {
			return "", fmt.Errorf("get log type: %w", err)
		}
		if logType != "" {
			if chronicleLogType, ok := supportedLogTypes[logType]; ok {
				return chronicleLogType, nil
			}
		}
	}

	return m.cfg.LogType, nil
}

func (m *protoMarshaler) getNamespace(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
	// check for attributes in attributes["chronicle_namespace"]
	namespace, err := GetRawField(ctx, m.set, chronicleNamespaceField, logRecord, scope, resource)
	if err != nil {
		return "", fmt.Errorf("get chronicle log type: %w", err)
	}
	if namespace != "" {
		return namespace, nil
	}
	return m.cfg.Namespace, nil
}

// GetRawField is a helper function to extract a field from a log record using an OTTL expression.
func GetRawField(ctx context.Context, set component.TelemetrySettings, field string, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
	switch field {
	case "body":
		switch logRecord.Body().Type() {
		case pcommon.ValueTypeStr:
			return logRecord.Body().Str(), nil
		case pcommon.ValueTypeMap:
			bytes, err := json.Marshal(logRecord.Body().AsRaw())
			if err != nil {
				return "", fmt.Errorf("marshal log body: %w", err)
			}
			return string(bytes), nil
		}
	case logTypeField:
		attributes := logRecord.Attributes().AsRaw()
		if logType, ok := attributes["log_type"]; ok {
			if v, ok := logType.(string); ok {
				return v, nil
			}
		}
		return "", nil
	case chronicleLogTypeField:
		attributes := logRecord.Attributes().AsRaw()
		if logType, ok := attributes["chronicle_log_type"]; ok {
			if v, ok := logType.(string); ok {
				return v, nil
			}
		}
		return "", nil
	case chronicleNamespaceField:
		attributes := logRecord.Attributes().AsRaw()
		if namespace, ok := attributes["chronicle_namespace"]; ok {
			if v, ok := namespace.(string); ok {
				return v, nil
			}
		}
		return "", nil
	}

	lrExpr, err := expr.NewOTTLLogRecordExpression(field, set)
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
