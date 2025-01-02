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
	"strings"
	"time"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HTTPConfig is the configuration for the HTTP marshaler
type HTTPConfig struct {
	Config
	Project   string
	Location  string
	Forwarder string
}

// HTTP is a marshaler for HTTP protos
type HTTP struct {
	protoMarshaler
	project   string
	location  string
	forwarder string
}

// NewHTTP creates a new HTTP marshaler
func NewHTTP(cfg HTTPConfig, set component.TelemetrySettings) (*HTTP, error) {
	m, err := newProtoMarshaler(cfg.Config, set)
	if err != nil {
		return nil, err
	}
	return &HTTP{
		protoMarshaler: *m,
		project:        cfg.Project,
		location:       cfg.Location,
		forwarder:      cfg.Forwarder,
	}, nil
}

// MarshalLogs marshals logs into HTTP payloads
func (m *HTTP) MarshalLogs(ctx context.Context, ld plog.Logs) (map[string][]*api.ImportLogsRequest, error) {
	rawLogs, err := m.extractRawHTTPLogs(ctx, ld)
	if err != nil {
		return nil, fmt.Errorf("extract raw logs: %w", err)
	}
	return m.constructHTTPPayloads(rawLogs), nil
}

func (m *HTTP) extractRawHTTPLogs(ctx context.Context, ld plog.Logs) (map[string][]*api.Log, error) {
	entries := make(map[string][]*api.Log)
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				logRecord := scopeLog.LogRecords().At(k)
				rawLog, logType, namespace, ingestionLabels, err := m.processHTTPLogRecord(ctx, logRecord, scopeLog, resourceLog)
				if err != nil {
					m.set.Logger.Error("Error processing log record", zap.Error(err))
					continue
				}

				if rawLog == "" {
					continue
				}

				var timestamp time.Time
				if logRecord.Timestamp() != 0 {
					timestamp = logRecord.Timestamp().AsTime()
				} else {
					timestamp = logRecord.ObservedTimestamp().AsTime()
				}

				entry := &api.Log{
					LogEntryTime:         timestamppb.New(timestamp),
					CollectionTime:       timestamppb.New(logRecord.ObservedTimestamp().AsTime()),
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

func (m *HTTP) processHTTPLogRecord(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, string, string, map[string]*api.Log_LogLabel, error) {
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

func (m *HTTP) getHTTPIngestionLabels(logRecord plog.LogRecord) (map[string]*api.Log_LogLabel, error) {
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

func (m *HTTP) getHTTPRawNestedFields(field string, logRecord plog.LogRecord) (map[string]*api.Log_LogLabel, error) {
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

func (m *HTTP) buildForwarderString() string {
	format := "projects/%s/locations/%s/instances/%s/forwarders/%s"
	return fmt.Sprintf(format, m.project, m.location, m.customerID, m.forwarder)
}

func (m *HTTP) constructHTTPPayloads(rawLogs map[string][]*api.Log) map[string][]*api.ImportLogsRequest {
	payloads := make(map[string][]*api.ImportLogsRequest, len(rawLogs))

	for logType, entries := range rawLogs {
		if len(entries) > 0 {
			request := m.buildHTTPRequest(entries)

			payloads[logType] = m.enforceMaximumsHTTPRequest(request)
		}
	}
	return payloads
}

func (m *HTTP) enforceMaximumsHTTPRequest(request *api.ImportLogsRequest) []*api.ImportLogsRequest {
	size := proto.Size(request)
	logs := request.GetInlineSource().Logs
	if size <= m.cfg.BatchRequestSizeLimit && len(logs) <= m.cfg.BatchLogCountLimit {
		return []*api.ImportLogsRequest{
			request,
		}
	}

	if len(logs) < 2 {
		m.set.Logger.Error("Single entry exceeds max request size. Dropping entry", zap.Int("size", size))
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

func (m *HTTP) buildHTTPRequest(entries []*api.Log) *api.ImportLogsRequest {
	return &api.ImportLogsRequest{
		// TODO: Add parent and hint
		// We don't yet have solid guidance on what these should be
		Parent: "",
		Hint:   "",

		Source: &api.ImportLogsRequest_InlineSource{
			InlineSource: &api.ImportLogsRequest_LogsInlineSource{
				Forwarder: m.buildForwarderString(),
				Logs:      entries,
			},
		},
	}
}
