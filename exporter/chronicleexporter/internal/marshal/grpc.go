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

// Package marshal contains marshalers for grpc and http
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

// GRPC is a marshaler for gRPC protos
type GRPC struct {
	protoMarshaler
}

// NewGRPC creates a new GRPC marshaler
func NewGRPC(cfg Config, set component.TelemetrySettings) (*GRPC, error) {
	m, err := newProtoMarshaler(cfg, set)
	if err != nil {
		return nil, err
	}
	return &GRPC{protoMarshaler: *m}, nil
}

// MarshalLogs marshals logs into gRPC requests
func (m *GRPC) MarshalLogs(ctx context.Context, ld plog.Logs) ([]*api.BatchCreateLogsRequest, error) {
	rawLogs, namespace, ingestionLabels, err := m.extractRawLogs(ctx, ld)
	if err != nil {
		return nil, fmt.Errorf("extract raw logs: %w", err)
	}
	return m.constructPayloads(rawLogs, namespace, ingestionLabels), nil
}

func (m *GRPC) extractRawLogs(ctx context.Context, ld plog.Logs) (map[string][]*api.LogEntry, map[string]string, map[string][]*api.Label, error) {
	entries := make(map[string][]*api.LogEntry)
	namespaceMap := make(map[string]string)
	ingestionLabelsMap := make(map[string][]*api.Label)

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				logRecord := scopeLog.LogRecords().At(k)
				rawLog, logType, namespace, ingestionLabels, err := m.processLogRecord(ctx, logRecord, scopeLog, resourceLog)

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

				entry := &api.LogEntry{
					Timestamp:      timestamppb.New(timestamp),
					CollectionTime: timestamppb.New(logRecord.ObservedTimestamp().AsTime()),
					Data:           []byte(rawLog),
				}
				entries[logType] = append(entries[logType], entry)
				// each logType maps to exactly 1 namespace value
				if namespace != "" {
					if _, ok := namespaceMap[logType]; !ok {
						namespaceMap[logType] = namespace
					}
				}
				if len(ingestionLabels) > 0 {
					// each logType maps to a list of ingestion labels
					if _, exists := ingestionLabelsMap[logType]; !exists {
						ingestionLabelsMap[logType] = make([]*api.Label, 0)
					}
					existingLabels := make(map[string]struct{})
					for _, label := range ingestionLabelsMap[logType] {
						existingLabels[label.Key] = struct{}{}
					}
					for _, label := range ingestionLabels {
						// only add to ingestionLabelsMap if the label is unique
						if _, ok := existingLabels[label.Key]; !ok {
							ingestionLabelsMap[logType] = append(ingestionLabelsMap[logType], label)
							existingLabels[label.Key] = struct{}{}
						}
					}
				}
			}
		}
	}
	return entries, namespaceMap, ingestionLabelsMap, nil
}

func (m *GRPC) processLogRecord(ctx context.Context, logRecord plog.LogRecord, scope plog.ScopeLogs, resource plog.ResourceLogs) (string, string, string, []*api.Label, error) {
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

func (m *GRPC) getIngestionLabels(logRecord plog.LogRecord) ([]*api.Label, error) {
	// check for labels in attributes["chronicle_ingestion_labels"]
	ingestionLabels, err := m.getRawNestedFields(chronicleIngestionLabelsPrefix, logRecord)
	if err != nil {
		return []*api.Label{}, fmt.Errorf("get chronicle ingestion labels: %w", err)
	}

	if len(ingestionLabels) != 0 {
		return ingestionLabels, nil
	}
	// use labels defined in config if needed
	configLabels := make([]*api.Label, 0)
	for key, value := range m.cfg.IngestionLabels {
		configLabels = append(configLabels, &api.Label{
			Key:   key,
			Value: value,
		})
	}
	return configLabels, nil
}

func (m *GRPC) getRawNestedFields(field string, logRecord plog.LogRecord) ([]*api.Label, error) {
	var nestedFields []*api.Label
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
				nestedFields = append(nestedFields, &api.Label{Key: k, Value: v})
			}
		} else {
			nestedFields = append(nestedFields, &api.Label{Key: cleanKey, Value: value.AsString()})
		}
		return true
	})
	return nestedFields, nil
}

func (m *GRPC) constructPayloads(rawLogs map[string][]*api.LogEntry, namespaceMap map[string]string, ingestionLabelsMap map[string][]*api.Label) []*api.BatchCreateLogsRequest {
	payloads := make([]*api.BatchCreateLogsRequest, 0, len(rawLogs))
	for logType, entries := range rawLogs {
		if len(entries) > 0 {
			namespace, ok := namespaceMap[logType]
			if !ok {
				namespace = m.cfg.Namespace
			}
			ingestionLabels := ingestionLabelsMap[logType]

			request := m.buildGRPCRequest(entries, logType, namespace, ingestionLabels)

			payloads = append(payloads, m.enforceMaximumsGRPCRequest(request)...)
		}
	}
	return payloads
}

func (m *GRPC) enforceMaximumsGRPCRequest(request *api.BatchCreateLogsRequest) []*api.BatchCreateLogsRequest {
	size := proto.Size(request)
	entries := request.Batch.Entries
	if size <= m.cfg.BatchRequestSizeLimit && len(entries) <= m.cfg.BatchLogCountLimit {
		return []*api.BatchCreateLogsRequest{
			request,
		}
	}

	if len(entries) < 2 {
		m.set.Logger.Error("Single entry exceeds max request size. Dropping entry", zap.Int("size", size))
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

func (m *GRPC) buildGRPCRequest(entries []*api.LogEntry, logType, namespace string, ingestionLabels []*api.Label) *api.BatchCreateLogsRequest {
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
