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

package chronicleexporter

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-agent/exporter/chronicleexporter/protos/api"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtoMarshaler_MarshalRawLogs(t *testing.T) {
	logger := zap.NewNop()
	startTime := time.Now()

	tests := []struct {
		name         string
		cfg          Config
		logRecords   func() plog.Logs
		expectations func(t *testing.T, requests []*api.BatchCreateLogsRequest)
	}{
		{
			name: "Single log record with expected data",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", map[string]any{"log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["env"]`: "prod"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "WINEVTLOG", batch.LogType)
				require.Len(t, batch.Entries, 1)

				// Convert Data (byte slice) to string for comparison
				logDataAsString := string(batch.Entries[0].Data)
				expectedLogData := `Test log message`
				require.Equal(t, expectedLogData, logDataAsString)

				require.NotNil(t, batch.StartTime)
				require.True(t, timestamppb.New(startTime).AsTime().Equal(batch.StartTime.AsTime()), "Start time should be set correctly")
			},
		},
		{
			name: "Single log record with expected data, no log_type, namespace, or ingestion labels",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				RawLogField:     "body",
				OverrideLogType: true,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", nil))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "WINEVTLOG", batch.LogType)
				require.Equal(t, "", batch.Source.Namespace)
				require.Equal(t, 0, len(batch.Source.Labels))
				require.Len(t, batch.Entries, 1)

				// Convert Data (byte slice) to string for comparison
				logDataAsString := string(batch.Entries[0].Data)
				expectedLogData := `Test log message`
				require.Equal(t, expectedLogData, logDataAsString)

				require.NotNil(t, batch.StartTime)
				require.True(t, timestamppb.New(startTime).AsTime().Equal(batch.StartTime.AsTime()), "Start time should be set correctly")
			},
		},
		{
			name: "Multiple log records",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				IngestionLabels: map[string]string{`chronicle_ingestion_label["env"]`: "staging"},
				RawLogField:     "body",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1, "Expected a single batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 2, "Expected two log entries in the batch")
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(batch.Entries[0].Data))
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(batch.Entries[1].Data))
			},
		},
		{
			name: "Log record with attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				IngestionLabels: map[string]string{`chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"},
				RawLogField:     "attributes",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 1)

				// Assuming the attributes are marshaled into the Data field as a JSON string
				expectedData := `{"key1":"value1", "log_type":"WINEVTLOG", "namespace":"test", "chronicle_ingestion_label[\"key1\"]": "value1", "chronicle_ingestion_label[\"key2\"]": "value2"}`
				actualData := string(batch.Entries[0].Data)
				require.JSONEq(t, expectedData, actualData, "Log attributes should match expected")
			},
		},
		{
			name: "No log records",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "DEFAULT",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				return plog.NewLogs() // No log records added
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 0, "Expected no requests due to no log records")
			},
		},
		{
			name: "No log type set in config or attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				RawLogField:     "body",
				OverrideLogType: true,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log without logType", map[string]any{"namespace": "test", `ingestion_label["realkey1"]`: "realvalue1", `ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "", batch.LogType, "Expected log type to be empty")
			},
		},
		{
			name: "Multiple log records with duplicate data, no log type in attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record1.Attributes().FromRaw(map[string]any{"chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				record2.Attributes().FromRaw(map[string]any{"chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				// verify 1 request for one log type
				require.Len(t, requests, 1, "Expected a single batch request")
				batch := requests[0].Batch
				// verify batch source labels
				require.Len(t, batch.Source.Labels, 2)
				require.Len(t, batch.Entries, 2, "Expected two log entries in the batch")
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(batch.Entries[0].Data))
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(batch.Entries[1].Data))
			},
		},
		{
			name: "Multiple log records with different data, no log type in attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record1.Attributes().FromRaw(map[string]any{`chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				record2.Attributes().FromRaw(map[string]any{`chronicle_ingestion_label["key3"]`: "value3", `chronicle_ingestion_label["key4"]`: "value4"})
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				// verify 1 request for one log type
				require.Len(t, requests, 1, "Expected a single batch request")
				batch := requests[0].Batch
				// verify batch source labels
				require.Len(t, batch.Source.Labels, 4)
				require.Len(t, batch.Entries, 2, "Expected two log entries in the batch")
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(batch.Entries[0].Data))
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(batch.Entries[1].Data))
			},
		},
		{
			name: "Override log type with attribute",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "DEFAULT", // This should be overridden by the log_type attribute
				Namespace:       "DEFAULT",
				IngestionLabels: map[string]string{`ingestion_label["DEFAULTKEY1"]`: "DEFAULTVALUE1", `ingestion_label["DEFAULTKEY2"]`: "DEFAUTLVALUE2"},
				RawLogField:     "body",
				OverrideLogType: true,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"log_type": "windows_event.application", "namespace": "test", `ingestion_label["realkey1"]`: "realvalue1", `ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "WINEVTLOG", batch.LogType, "Expected log type to be overridden by attribute")
			},
		},
		{
			name: "Override log type with chronicle attribute",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "DEFAULT",                                                                                                               // This should be overridden by the chronicle_log_type attribute
				Namespace:       "DEFAULT",                                                                                                               // This should be overridden by the chronicle_namespace attribute
				IngestionLabels: map[string]string{`ingestion_label["DEFAULTKEY1"]`: "DEFAULTVALUE1", `ingestion_label["DEFAULTKEY2"]`: "DEFAUTLVALUE2"}, // This should be overridden by the chronicle_ingestion_label attribute
				RawLogField:     "body",
				OverrideLogType: true,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"chronicle_log_type": "ASOC_ALERT", "chronicle_namespace": "test", `chronicle_ingestion_label["realkey1"]`: "realvalue1", `chronicle_ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "ASOC_ALERT", batch.LogType, "Expected log type to be overridden by attribute")
				require.Equal(t, "test", batch.Source.Namespace, "Expected namespace to be overridden by attribute")
				require.Equal(t, `chronicle_ingestion_label["realkey1"]`, batch.Source.Labels[0].Key, "Expected ingestion label to be overridden by attribute")
				require.Equal(t, "realvalue1", batch.Source.Labels[0].Value, "Expected ingestion label to be overridden by attribute")
				require.Equal(t, `chronicle_ingestion_label["realkey2"]`, batch.Source.Labels[1].Key, "Expected ingestion label to be overridden by attribute")
				require.Equal(t, "realvalue2", batch.Source.Labels[1].Value, "Expected ingestion label to be overridden by attribute")
			},
		},
		{
			name: "Multiple log records with duplicate data, log type in attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "TEST",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})

				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				record2.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				// verify 2 requests for 2 different log types
				require.Len(t, requests, 1, "Expected a single batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 2, "Expected two log entries in the batch")
				// verify batch for first log
				require.Equal(t, "WINEVTLOGS", batch.LogType)
				require.Equal(t, "test1", batch.Source.Namespace)
				require.Equal(t, "value2", batch.Source.Labels[1].Value)
				require.Len(t, batch.Source.Labels, 2)
				require.Equal(t, `chronicle_ingestion_label["key2"]`, batch.Source.Labels[1].Key)
				require.Equal(t, "value2", batch.Source.Labels[1].Value)
			},
		},
		{
			name: "Multiple log records with different data, log type in attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "TEST",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})

				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				record2.Attributes().FromRaw(map[string]any{"chronicle_log_type": "ASOC_ALERT", "chronicle_namespace": "test2", `chronicle_ingestion_label["key3"]`: "value3", `chronicle_ingestion_label["key4"]`: "value4"})
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				// verify 2 requests for 2 different log types
				require.Len(t, requests, 2, "Expected a two batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 1, "Expected two log entries in the batch")
				// verify batch for first log
				require.Equal(t, "WINEVTLOGS", batch.LogType)
				require.Equal(t, "test1", batch.Source.Namespace)
				require.Equal(t, "value2", batch.Source.Labels[1].Value)
				require.Len(t, batch.Source.Labels, 2)
				require.Equal(t, `chronicle_ingestion_label["key2"]`, batch.Source.Labels[1].Key)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerID, err := uuid.Parse(tt.cfg.CustomerID)
			require.NoError(t, err)

			marshaler, err := newProtoMarshaler(tt.cfg, component.TelemetrySettings{Logger: logger}, customerID[:])
			marshaler.startTime = startTime
			require.NoError(t, err)

			logs := tt.logRecords()
			requests, err := marshaler.MarshalRawLogs(context.Background(), logs)
			require.NoError(t, err)

			tt.expectations(t, requests)
		})
	}
}

func TestProtoMarshaler_MarshalRawLogsForHTTP(t *testing.T) {
	logger := zap.NewNop()
	startTime := time.Now()

	tests := []struct {
		name         string
		cfg          Config
		labels       []*api.Label
		logRecords   func() plog.Logs
		expectations func(t *testing.T, requests map[string]*api.ImportLogsRequest)
	}{
		{
			name: "Single log record with expected data",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				RawLogField:     "body",
				OverrideLogType: false,
				Protocol:        protocolHTTPS,
				Project:         "test-project",
				Location:        "us",
				Forwarder:       uuid.New().String(),
			},
			labels: []*api.Label{
				{Key: "env", Value: "prod"},
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", map[string]any{"log_type": "WINEVTLOG", "namespace": "test"}))
			},
			expectations: func(t *testing.T, requests map[string]*api.ImportLogsRequest) {
				require.Len(t, requests, 1)
				logs := requests["WINEVTLOG"].GetInlineSource().Logs
				require.Len(t, logs, 1)
				// Convert Data (byte slice) to string for comparison
				logDataAsString := string(logs[0].Data)
				expectedLogData := `Test log message`
				require.Equal(t, expectedLogData, logDataAsString)
			},
		},
		{
			name: "Multiple log records",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				IngestionLabels: map[string]string{`chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"},
				RawLogField:     "body",
				OverrideLogType: false,
			},
			labels: []*api.Label{
				{Key: "env", Value: "staging"},
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				return logs
			},
			expectations: func(t *testing.T, requests map[string]*api.ImportLogsRequest) {
				require.Len(t, requests, 1, "Expected a single batch request")
				logs := requests["WINEVTLOG"].GetInlineSource().Logs
				require.Len(t, logs, 2, "Expected two log entries in the batch")
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(logs[0].Data))
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(logs[1].Data))
			},
		},
		{
			name: "Log record with attributes",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "WINEVTLOG",
				Namespace:       "test",
				IngestionLabels: map[string]string{`chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"},
				RawLogField:     "attributes",
				OverrideLogType: false,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"}))
			},
			expectations: func(t *testing.T, requests map[string]*api.ImportLogsRequest) {
				require.Len(t, requests, 1)
				logs := requests["WINEVTLOG"].GetInlineSource().Logs
				// Assuming the attributes are marshaled into the Data field as a JSON string
				expectedData := `{"key1":"value1", "log_type":"WINEVTLOG", "namespace":"test", "chronicle_ingestion_label[\"key1\"]": "value1", "chronicle_ingestion_label[\"key2\"]": "value2"}`
				actualData := string(logs[0].Data)
				require.JSONEq(t, expectedData, actualData, "Log attributes should match expected")
			},
		},
		{
			name: "No log records",
			cfg: Config{
				CustomerID:      uuid.New().String(),
				LogType:         "DEFAULT",
				Namespace:       "DEFAULT",
				RawLogField:     "body",
				OverrideLogType: false,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return plog.NewLogs() // No log records added
			},
			expectations: func(t *testing.T, requests map[string]*api.ImportLogsRequest) {
				require.Len(t, requests, 0, "Expected no requests due to no log records")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerID, err := uuid.Parse(tt.cfg.CustomerID)
			require.NoError(t, err)

			marshaler, err := newProtoMarshaler(tt.cfg, component.TelemetrySettings{Logger: logger}, customerID[:])
			marshaler.startTime = startTime
			require.NoError(t, err)

			logs := tt.logRecords()
			requests, err := marshaler.MarshalRawLogsForHTTP(context.Background(), logs)
			require.NoError(t, err)

			tt.expectations(t, requests)
		})
	}
}

func mockLogRecord(body string, attributes map[string]any) plog.LogRecord {
	lr := plog.NewLogRecord()
	lr.Body().SetStr(body)
	for k, v := range attributes {
		switch val := v.(type) {
		case string:
			lr.Attributes().PutStr(k, val)
		default:
		}
	}
	return lr
}

func mockLogs(record plog.LogRecord) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	record.CopyTo(sl.LogRecords().AppendEmpty())
	return logs
}
