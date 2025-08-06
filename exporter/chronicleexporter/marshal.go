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

package chronicleexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"github.com/observiq/bindplane-otel-collector/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	logTypeField                   = `attributes["log_type"]`
	chronicleLogTypeField          = `attributes["chronicle_log_type"]`
	chronicleNamespaceField        = `attributes["chronicle_namespace"]`
	chronicleIngestionLabelsPrefix = `chronicle_ingestion_label`

	// catchAllLogType is the log type that is used when the log type is not found in the log types map
	catchAllLogType = "CATCH_ALL"
)

// Specific collector IDs for Chronicle used to identify bindplane agents.
var (
	googleCollectorID           = uuid.MustParse("aaaa1111-aaaa-1111-aaaa-1111aaaa1111")
	googleEnterpriseCollectorID = uuid.MustParse("aaaa1111-aaaa-1111-aaaa-1111aaaa1112")
	enterpriseCollectorID       = uuid.MustParse("aaaa1111-aaaa-1111-aaaa-1111aaaa1113")
	defaultCollectorID          = uuid.MustParse("aaaa1111-aaaa-1111-aaaa-1111aaaa1114")
)

const (
	licenseTypeGoogle           = "Google"
	licenseTypeGoogleEnterprise = "GoogleEnterprise"
	licenseTypeEnterprise       = "Enterprise"
)

var supportedLogTypes = map[string]string{
	"windows_event.security":    "WINEVTLOG",
	"windows_event.application": "WINEVTLOG",
	"windows_event.system":      "WINEVTLOG",
	"sql_server":                "MICROSOFT_SQL",
}

type protoMarshaler struct {
	cfg          Config
	teleSettings component.TelemetrySettings
	startTime    time.Time
	customerID   []byte
	collectorID  []byte
	telemetry    *metadata.TelemetryBuilder
	logTypes     map[string]exists
	logger       *zap.Logger
}

func newProtoMarshaler(cfg Config, teleSettings component.TelemetrySettings, telemetry *metadata.TelemetryBuilder, logger *zap.Logger) (*protoMarshaler, error) {
	customerID, err := uuid.Parse(cfg.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("parse customer ID: %w", err)
	}
	return &protoMarshaler{
		startTime:    time.Now(),
		cfg:          cfg,
		teleSettings: teleSettings,
		customerID:   customerID[:],
		collectorID:  getCollectorID(cfg.LicenseType),
		telemetry:    telemetry,
		logger:       logger,
	}, nil
}

func (m *protoMarshaler) MarshalRawLogs(ctx context.Context, ld plog.Logs) ([]*api.BatchCreateLogsRequest, error) {
	logGrouper, err := m.extractRawLogs(ctx, ld)
	if err != nil {
		return nil, fmt.Errorf("extract raw logs: %w", err)
	}
	return m.constructPayloads(logGrouper), nil
}

func (m *protoMarshaler) extractRawLogs(ctx context.Context, ld plog.Logs) (*logGrouper, error) {
	logGrouper := newLogGrouper()
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				logRecord := scopeLog.LogRecords().At(k)

				rawLog, logType, namespace, ingestionLabels, err := m.processLogRecord(ctx, logRecord, scopeLog, resourceLog)

				if err != nil {
					m.teleSettings.Logger.Error("Error processing log record", zap.Error(err))
					continue
				}

				if rawLog == "" {
					continue
				}

				timestamp := getTimestamp(logRecord)
				collectionTime := getObservedTimestamp(logRecord)

				entry := &api.LogEntry{
					Timestamp:      timestamppb.New(timestamp),
					CollectionTime: timestamppb.New(collectionTime),
					Data:           []byte(rawLog),
				}
				logGrouper.Add(entry, namespace, logType, ingestionLabels)
			}
		}
	}

	return logGrouper, nil
}

func (m *protoMarshaler) processLogRecord(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, string, string, []*api.Label, error) {
	rawLog, err := m.getRawLog(ctx, logRecord, scope, resource)
	if err != nil {
		return "", "", "", nil, err
	}
	logType, err := m.getLogType(ctx, logRecord, scope, resource)
	if err != nil {
		return "", "", "", nil, err
	}
	namespace, err := m.getNamespace(ctx, logRecord, scope, resource)
	if err != nil {
		return "", "", "", nil, err
	}
	ingestionLabels, err := m.getIngestionLabels(logRecord)
	if err != nil {
		return "", "", "", nil, err
	}
	return rawLog, logType, namespace, ingestionLabels, nil
}

func (m *protoMarshaler) processHTTPLogRecord(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, string, string, map[string]*api.Log_LogLabel, error) {
	rawLog, err := m.getRawLog(ctx, logRecord, scope, resource)
	if err != nil {
		return "", "", "", nil, err
	}

	logType, err := m.getLogType(ctx, logRecord, scope, resource)
	if err != nil {
		return "", "", "", nil, err
	}
	namespace, err := m.getNamespace(ctx, logRecord, scope, resource)
	if err != nil {
		return "", "", "", nil, err
	}
	ingestionLabels, err := m.getHTTPIngestionLabels(logRecord)
	if err != nil {
		return "", "", "", nil, err
	}

	return rawLog, logType, namespace, ingestionLabels, nil
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
	return m.getRawField(ctx, m.cfg.RawLogField, logRecord, scope, resource)
}

func (m *protoMarshaler) shouldValidateLogType() bool {
	return m.cfg.ValidateLogTypes && m.logTypes != nil
}

func (m *protoMarshaler) getLogType(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
	// check for attributes in attributes["chronicle_log_type"]
	logType, err := m.getRawField(ctx, chronicleLogTypeField, logRecord, scope, resource)
	if err != nil {
		return "", fmt.Errorf("get chronicle log type: %w", err)
	}

	if logType != "" {
		if m.shouldValidateLogType() {
			if _, ok := m.logTypes[logType]; ok {
				return logType, nil
			}
			m.logger.Warn("Log type could not be validated", zap.String("logType", logType), zap.String("logTypeField", chronicleLogTypeField))
		} else {
			return logType, nil
		}
	}

	if m.cfg.OverrideLogType {
		logType, err := m.getRawField(ctx, logTypeField, logRecord, scope, resource)

		if err != nil {
			return "", fmt.Errorf("get log type: %w", err)
		}
		if logType != "" {
			if chronicleLogType, ok := supportedLogTypes[logType]; ok {
				return chronicleLogType, nil
			}
		}
	}

	if m.cfg.LogType == "" {
		return catchAllLogType, nil
	}

	if m.shouldValidateLogType() {
		if _, ok := m.logTypes[m.cfg.LogType]; !ok {
			m.logger.Warn("Default log type not found in log types map", zap.String("logType", m.cfg.LogType))
			return catchAllLogType, nil
		}
	}
	return m.cfg.LogType, nil
}

func (m *protoMarshaler) getNamespace(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
	// check for attributes in attributes["chronicle_namespace"]
	namespace, err := m.getRawField(ctx, chronicleNamespaceField, logRecord, scope, resource)
	if err != nil {
		return "", fmt.Errorf("get chronicle log type: %w", err)
	}
	if namespace != "" {
		return namespace, nil
	}
	return m.cfg.Namespace, nil
}

func (m *protoMarshaler) getIngestionLabels(logRecord plog.LogRecord) ([]*api.Label, error) {
	// check for labels in attributes["chronicle_ingestion_labels"]
	ingestionLabels, err := m.getRawNestedFields(chronicleIngestionLabelsPrefix, logRecord)
	if err != nil {
		return []*api.Label{}, fmt.Errorf("get chronicle ingestion labels: %w", err)
	}

	// merge in labels defined in config, using the labels defined in the log record if they exist
	configLabels := make([]*api.Label, 0)
	for key, value := range m.cfg.IngestionLabels {
		if _, exists := ingestionLabels[key]; !exists {
			configLabels = append(configLabels, &api.Label{
				Key:   key,
				Value: value,
			})
		}
	}
	for key, value := range ingestionLabels {
		configLabels = append(configLabels, &api.Label{
			Key:   key,
			Value: value,
		})
	}
	return configLabels, nil
}

func (m *protoMarshaler) getHTTPIngestionLabels(logRecord plog.LogRecord) (map[string]*api.Log_LogLabel, error) {
	// Check for labels in attributes["chronicle_ingestion_labels"]
	ingestionLabels, err := m.getHTTPRawNestedFields(chronicleIngestionLabelsPrefix, logRecord)
	if err != nil {
		return nil, fmt.Errorf("get chronicle ingestion labels: %w", err)
	}

	if len(ingestionLabels) != 0 {
		return ingestionLabels, nil
	}

	// use labels defined in the config if needed
	configLabels := make(map[string]*api.Log_LogLabel)
	for key, value := range m.cfg.IngestionLabels {
		configLabels[key] = &api.Log_LogLabel{
			Value: value,
		}
	}
	return configLabels, nil
}

func (m *protoMarshaler) getRawField(ctx context.Context, field string, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, error) {
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

func (m *protoMarshaler) getRawNestedFields(field string, logRecord plog.LogRecord) (map[string]string, error) {
	nestedFields := make(map[string]string)
	logRecord.Attributes().Range(func(key string, value pcommon.Value) bool {
		if !strings.HasPrefix(key, field) {
			return true
		}
		// Extract the key name from the nested field
		cleanKey := strings.Trim(key[len(field):], `[]"`)
		var jsonMap map[string]string

		// If needs to be parsed as JSON
		if err := json.Unmarshal([]byte(value.AsString()), &jsonMap); err == nil {
			for k, v := range jsonMap {
				nestedFields[k] = v
			}
		} else {
			nestedFields[cleanKey] = value.AsString()
		}
		return true
	})
	return nestedFields, nil
}

func (m *protoMarshaler) getHTTPRawNestedFields(field string, logRecord plog.LogRecord) (map[string]*api.Log_LogLabel, error) {
	nestedFields := make(map[string]*api.Log_LogLabel) // Map with key as string and value as Log_LogLabel
	logRecord.Attributes().Range(func(key string, value pcommon.Value) bool {
		if !strings.HasPrefix(key, field) {
			return true
		}
		// Extract the key name from the nested field
		cleanKey := strings.Trim(key[len(field):], `[]"`)
		var jsonMap map[string]string

		// If needs to be parsed as JSON
		if err := json.Unmarshal([]byte(value.AsString()), &jsonMap); err == nil {
			for k, v := range jsonMap {
				nestedFields[k] = &api.Log_LogLabel{
					Value: v,
				}
			}
		} else {
			nestedFields[cleanKey] = &api.Log_LogLabel{
				Value: value.AsString(),
			}
		}
		return true
	})

	return nestedFields, nil
}

func (m *protoMarshaler) constructPayloads(logGrouper *logGrouper) []*api.BatchCreateLogsRequest {
	payloads := make([]*api.BatchCreateLogsRequest, 0, len(logGrouper.groups))

	metricCtx := context.Background()

	logGrouper.ForEach(func(entries []*api.LogEntry, namespace, logType string, ingestionLabels []*api.Label) {
		if namespace == "" {
			namespace = m.cfg.Namespace
		}

		request := m.buildGRPCRequest(entries, logType, namespace, ingestionLabels)

		payloads = append(payloads, m.enforceMaximumsGRPCRequest(request)...)
		for _, payload := range payloads {
			m.telemetry.ExporterBatchSize.Record(metricCtx, int64(len(payload.Batch.Entries)))
			m.telemetry.ExporterPayloadSize.Record(metricCtx, int64(proto.Size(payload)))
		}
	})
	return payloads
}

func (m *protoMarshaler) enforceMaximumsGRPCRequest(request *api.BatchCreateLogsRequest) []*api.BatchCreateLogsRequest {
	size := proto.Size(request)
	entries := request.Batch.Entries
	if size <= m.cfg.BatchRequestSizeLimitGRPC {
		return []*api.BatchCreateLogsRequest{
			request,
		}
	}

	if len(entries) < 2 {
		m.teleSettings.Logger.Error("Single entry exceeds max request size. Dropping entry", zap.Int("size", size))
		return []*api.BatchCreateLogsRequest{}
	}

	// split request into two
	mid := len(entries) / 2
	leftHalf := entries[:mid]
	rightHalf := entries[mid:]

	request.Batch.Entries = leftHalf
	otherHalfRequest := m.buildGRPCRequest(rightHalf, request.Batch.LogType, request.Batch.Source.Namespace, request.Batch.Source.Labels)

	// re-enforce max size restriction on each half
	enforcedRequest := m.enforceMaximumsGRPCRequest(request)
	enforcedOtherHalfRequest := m.enforceMaximumsGRPCRequest(otherHalfRequest)

	return append(enforcedRequest, enforcedOtherHalfRequest...)
}

func (m *protoMarshaler) buildGRPCRequest(entries []*api.LogEntry, logType, namespace string, ingestionLabels []*api.Label) *api.BatchCreateLogsRequest {
	return &api.BatchCreateLogsRequest{
		Batch: &api.LogEntryBatch{
			StartTime: timestamppb.New(m.startTime),
			Entries:   entries,
			LogType:   logType,
			Source: &api.EventSource{
				CollectorId: m.collectorID,
				CustomerId:  m.customerID,
				Labels:      ingestionLabels,
				Namespace:   namespace,
			},
		},
	}
}

func (m *protoMarshaler) MarshalRawLogsForHTTP(ctx context.Context, ld plog.Logs) (map[string][]*api.ImportLogsRequest, error) {
	rawLogs, err := m.extractRawHTTPLogs(ctx, ld)
	if err != nil {
		return nil, fmt.Errorf("extract raw logs: %w", err)
	}
	return m.constructHTTPPayloads(rawLogs), nil
}

func (m *protoMarshaler) extractRawHTTPLogs(ctx context.Context, ld plog.Logs) (map[string][]*api.Log, error) {
	entries := make(map[string][]*api.Log)
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				logRecord := scopeLog.LogRecords().At(k)
				rawLog, logType, namespace, ingestionLabels, err := m.processHTTPLogRecord(ctx, logRecord, scopeLog, resourceLog)
				if err != nil {
					m.teleSettings.Logger.Error("Error processing log record", zap.Error(err))
					continue
				}

				if rawLog == "" {
					continue
				}

				timestamp := getTimestamp(logRecord)
				collectionTime := getObservedTimestamp(logRecord)

				entry := &api.Log{
					LogEntryTime:         timestamppb.New(timestamp),
					CollectionTime:       timestamppb.New(collectionTime),
					Data:                 []byte(rawLog),
					EnvironmentNamespace: namespace,
					Labels:               ingestionLabels,
				}
				entries[logType] = append(entries[logType], entry)
			}
		}
	}

	return entries, nil
}

func getTimestamp(logRecord plog.LogRecord) time.Time {
	if logRecord.Timestamp() != 0 {
		return logRecord.Timestamp().AsTime()
	}
	return getObservedTimestamp(logRecord)
}

func getObservedTimestamp(logRecord plog.LogRecord) time.Time {
	if logRecord.ObservedTimestamp() != 0 {
		return logRecord.ObservedTimestamp().AsTime()
	}
	return time.Now()
}

func buildForwarderString(cfg Config) string {
	format := "projects/%s/locations/%s/instances/%s/forwarders/%s"
	return fmt.Sprintf(format, cfg.Project, cfg.Location, cfg.CustomerID, cfg.Forwarder)
}

func (m *protoMarshaler) constructHTTPPayloads(rawLogs map[string][]*api.Log) map[string][]*api.ImportLogsRequest {
	payloads := make(map[string][]*api.ImportLogsRequest, len(rawLogs))

	metricCtx := context.Background()

	for logType, entries := range rawLogs {
		if len(entries) > 0 {
			request := m.buildHTTPRequest(entries)

			payloads[logType] = m.enforceMaximumsHTTPRequest(request)
			for _, payload := range payloads[logType] {
				m.telemetry.ExporterBatchSize.Record(metricCtx, int64(len(payload.GetInlineSource().Logs)))
				m.telemetry.ExporterPayloadSize.Record(metricCtx, int64(proto.Size(payload)))
			}
		}
	}
	return payloads
}

func (m *protoMarshaler) enforceMaximumsHTTPRequest(request *api.ImportLogsRequest) []*api.ImportLogsRequest {
	size := proto.Size(request)
	logs := request.GetInlineSource().Logs
	if size <= m.cfg.BatchRequestSizeLimitHTTP {
		return []*api.ImportLogsRequest{
			request,
		}
	}

	if len(logs) < 2 {
		m.teleSettings.Logger.Error("Single entry exceeds max request size. Dropping entry", zap.Int("size", size))
		return []*api.ImportLogsRequest{}
	}

	// split request into two
	mid := len(logs) / 2
	leftHalf := logs[:mid]
	rightHalf := logs[mid:]

	request.GetInlineSource().Logs = leftHalf
	otherHalfRequest := m.buildHTTPRequest(rightHalf)

	// re-enforce max size restriction on each half
	enforcedRequest := m.enforceMaximumsHTTPRequest(request)
	enforcedOtherHalfRequest := m.enforceMaximumsHTTPRequest(otherHalfRequest)

	return append(enforcedRequest, enforcedOtherHalfRequest...)
}

func (m *protoMarshaler) buildHTTPRequest(entries []*api.Log) *api.ImportLogsRequest {
	return &api.ImportLogsRequest{
		// TODO: Add parent and hint
		// We don't yet have solid guidance on what these should be
		Parent: "",
		Hint:   "",

		Source: &api.ImportLogsRequest_InlineSource{
			InlineSource: &api.ImportLogsRequest_LogsInlineSource{
				Forwarder: buildForwarderString(m.cfg),
				Logs:      entries,
			},
		},
	}
}

func getCollectorID(licenseType string) []byte {
	switch strings.ToLower(licenseType) {
	case strings.ToLower(licenseTypeGoogle):
		return googleCollectorID[:]
	case strings.ToLower(licenseTypeGoogleEnterprise):
		return googleEnterpriseCollectorID[:]
	case strings.ToLower(licenseTypeEnterprise):
		return enterpriseCollectorID[:]
	default:
		return defaultCollectorID[:]
	}
}
