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

package marshal_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/marshal"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGRPC(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name         string
		cfg          marshal.Config
		logRecords   func() plog.Logs
		expectations func(t *testing.T, requests []*api.BatchCreateLogsRequest, startTime time.Time)
	}{
		{
			name: "Single log record with expected data",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", map[string]any{"log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["env"]`: "prod"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, startTime time.Time) {
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
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       true,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", nil))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, startTime time.Time) {
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
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
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
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "attributes",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
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
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "DEFAULT",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				return plog.NewLogs() // No log records added
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				require.Len(t, requests, 0, "Expected no requests due to no log records")
			},
		},
		{
			name: "No log type set in config or attributes",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				RawLogField:           "body",
				OverrideLogType:       true,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log without logType", map[string]any{"namespace": "test", `ingestion_label["realkey1"]`: "realvalue1", `ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "", batch.LogType, "Expected log type to be empty")
			},
		},
		{
			name: "Multiple log records with duplicate data, no log type in attributes",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
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
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify one request for log type in config
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
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
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
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify one request for one log type
				require.Len(t, requests, 1, "Expected a single batch request")
				batch := requests[0].Batch
				require.Equal(t, "WINEVTLOG", batch.LogType)
				require.Equal(t, "", batch.Source.Namespace)
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
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "DEFAULT", // This should be overridden by the log_type attribute
				RawLogField:           "body",
				OverrideLogType:       true,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"log_type": "windows_event.application", "namespace": "test", `ingestion_label["realkey1"]`: "realvalue1", `ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "WINEVTLOG", batch.LogType, "Expected log type to be overridden by attribute")
			},
		},
		{
			name: "Override log type with chronicle attribute",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "DEFAULT", // This should be overridden by the chronicle_log_type attribute
				RawLogField:           "body",
				OverrideLogType:       true,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"chronicle_log_type": "ASOC_ALERT", "chronicle_namespace": "test", `chronicle_ingestion_label["realkey1"]`: "realvalue1", `chronicle_ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				require.Len(t, requests, 1)
				batch := requests[0].Batch
				require.Equal(t, "ASOC_ALERT", batch.LogType, "Expected log type to be overridden by attribute")
				require.Equal(t, "test", batch.Source.Namespace, "Expected namespace to be overridden by attribute")
				expectedLabels := map[string]string{
					"realkey1": "realvalue1",
					"realkey2": "realvalue2",
				}
				for _, label := range batch.Source.Labels {
					require.Equal(t, expectedLabels[label.Key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Multiple log records with duplicate data, log type in attributes",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
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
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify 1 request, 2 batches for same log type
				require.Len(t, requests, 1, "Expected a single batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 2, "Expected two log entries in the batch")
				// verify batch for first log
				require.Equal(t, "WINEVTLOGS", batch.LogType)
				require.Equal(t, "test1", batch.Source.Namespace)
				require.Len(t, batch.Source.Labels, 2)
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
				for _, label := range batch.Source.Labels {
					require.Equal(t, expectedLabels[label.Key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Multiple log records with different data, log type in attributes",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr("First log message")
				record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})

				record2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetStr("Second log message")
				record2.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS2", "chronicle_namespace": "test2", `chronicle_ingestion_label["key3"]`: "value3", `chronicle_ingestion_label["key4"]`: "value4"})
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify 2 requests, with 1 batch for different log types
				require.Len(t, requests, 2, "Expected a two batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 1, "Expected one log entries in the batch")
				// verify batch for first log
				require.Contains(t, batch.LogType, "WINEVTLOGS")
				require.Contains(t, batch.Source.Namespace, "test")
				require.Len(t, batch.Source.Labels, 2)

				batch2 := requests[1].Batch
				require.Len(t, batch2.Entries, 1, "Expected one log entries in the batch")
				// verify batch for second log
				require.Contains(t, batch2.LogType, "WINEVTLOGS")
				require.Contains(t, batch2.Source.Namespace, "test")
				require.Len(t, batch2.Source.Labels, 2)
				// verify ingestion labels
				for _, req := range requests {
					for _, label := range req.Batch.Source.Labels {
						require.Contains(t, []string{
							"key1",
							"key2",
							"key3",
							"key4",
						}, label.Key)
						require.Contains(t, []string{
							"value1",
							"value2",
							"value3",
							"value4",
						}, label.Value)
					}
				}
			},
		},
		{
			name: "Many logs, all one batch",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				for i := 0; i < 1000; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr("Log message")
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify 1 request, with 1 batch
				require.Len(t, requests, 1, "Expected a one-batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 1000, "Expected 1000 log entries in the batch")
				// verify batch for first log
				require.Contains(t, batch.LogType, "WINEVTLOGS")
				require.Contains(t, batch.Source.Namespace, "test")
				require.Len(t, batch.Source.Labels, 2)

				// verify ingestion labels
				for _, req := range requests {
					for _, label := range req.Batch.Source.Labels {
						require.Contains(t, []string{
							"key1",
							"key2",
							"key3",
							"key4",
						}, label.Key)
						require.Contains(t, []string{
							"value1",
							"value2",
							"value3",
							"value4",
						}, label.Value)
					}
				}
			},
		},
		{
			name: "Single batch split into multiple because more than 1000 logs",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				for i := 0; i < 1001; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr("Log message")
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify 1 request, with 1 batch
				require.Len(t, requests, 2, "Expected a two-batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 500, "Expected 500 log entries in the first batch")
				// verify batch for first log
				require.Contains(t, batch.LogType, "WINEVTLOGS")
				require.Contains(t, batch.Source.Namespace, "test")
				require.Len(t, batch.Source.Labels, 2)

				batch2 := requests[1].Batch
				require.Len(t, batch2.Entries, 501, "Expected 501 log entries in the second batch")
				// verify batch for first log
				require.Contains(t, batch2.LogType, "WINEVTLOGS")
				require.Contains(t, batch2.Source.Namespace, "test")
				require.Len(t, batch2.Source.Labels, 2)

				// verify ingestion labels
				for _, req := range requests {
					for _, label := range req.Batch.Source.Labels {
						require.Contains(t, []string{
							"key1",
							"key2",
							"key3",
							"key4",
						}, label.Key)
						require.Contains(t, []string{
							"value1",
							"value2",
							"value3",
							"value4",
						}, label.Value)
					}
				}
			},
		},
		{
			name: "Recursively split batch, exceeds 1000 entries multiple times",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				for i := 0; i < 2002; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr("Log message")
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify 1 request, with 1 batch
				require.Len(t, requests, 4, "Expected a four-batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 500, "Expected 500 log entries in the first batch")
				// verify batch for first log
				require.Contains(t, batch.LogType, "WINEVTLOGS")
				require.Contains(t, batch.Source.Namespace, "test")
				require.Len(t, batch.Source.Labels, 2)

				batch2 := requests[1].Batch
				require.Len(t, batch2.Entries, 501, "Expected 501 log entries in the second batch")
				// verify batch for first log
				require.Contains(t, batch2.LogType, "WINEVTLOGS")
				require.Contains(t, batch2.Source.Namespace, "test")
				require.Len(t, batch2.Source.Labels, 2)

				batch3 := requests[2].Batch
				require.Len(t, batch3.Entries, 500, "Expected 500 log entries in the third batch")
				// verify batch for first log
				require.Contains(t, batch3.LogType, "WINEVTLOGS")
				require.Contains(t, batch3.Source.Namespace, "test")
				require.Len(t, batch3.Source.Labels, 2)

				batch4 := requests[3].Batch
				require.Len(t, batch4.Entries, 501, "Expected 501 log entries in the fourth batch")
				// verify batch for first log
				require.Contains(t, batch4.LogType, "WINEVTLOGS")
				require.Contains(t, batch4.Source.Namespace, "test")
				require.Len(t, batch4.Source.Labels, 2)

				// verify ingestion labels
				for _, req := range requests {
					for _, label := range req.Batch.Source.Labels {
						require.Contains(t, []string{
							"key1",
							"key2",
							"key3",
							"key4",
						}, label.Key)
						require.Contains(t, []string{
							"value1",
							"value2",
							"value3",
							"value4",
						}, label.Value)
					}
				}
			},
		},
		{
			name: "Single batch split into multiple because request size too large",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				// create 640 logs with size 8192 bytes each - totalling 5242880 bytes. non-body fields put us over limit
				for i := 0; i < 640; i++ {
					record1 := logRecords.AppendEmpty()
					body := tokenWithLength(8192)
					record1.Body().SetStr(string(body))
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify  request, with 1 batch
				require.Len(t, requests, 2, "Expected a two-batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 320, "Expected 320 log entries in the first batch")
				// verify batch for first log
				require.Contains(t, batch.LogType, "WINEVTLOGS")
				require.Contains(t, batch.Source.Namespace, "test")
				require.Len(t, batch.Source.Labels, 2)

				batch2 := requests[1].Batch
				require.Len(t, batch2.Entries, 320, "Expected 320 log entries in the second batch")
				// verify batch for first log
				require.Contains(t, batch2.LogType, "WINEVTLOGS")
				require.Contains(t, batch2.Source.Namespace, "test")
				require.Len(t, batch2.Source.Labels, 2)

				// verify ingestion labels
				for _, req := range requests {
					for _, label := range req.Batch.Source.Labels {
						require.Contains(t, []string{
							"key1",
							"key2",
							"key3",
							"key4",
						}, label.Key)
						require.Contains(t, []string{
							"value1",
							"value2",
							"value3",
							"value4",
						}, label.Value)
					}
				}
			},
		},
		{
			name: "Recursively split batch into multiple because request size too large",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				// create 1280 logs with size 8192 bytes each - totalling 5242880 * 2 bytes. non-body fields put us over twice the limit
				for i := 0; i < 1280; i++ {
					record1 := logRecords.AppendEmpty()
					body := tokenWithLength(8192)
					record1.Body().SetStr(string(body))
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify 1 request, with 1 batch
				require.Len(t, requests, 4, "Expected a four-batch request")
				batch := requests[0].Batch
				require.Len(t, batch.Entries, 320, "Expected 320 log entries in the first batch")
				// verify batch for first log
				require.Contains(t, batch.LogType, "WINEVTLOGS")
				require.Contains(t, batch.Source.Namespace, "test")
				require.Len(t, batch.Source.Labels, 2)

				batch2 := requests[1].Batch
				require.Len(t, batch2.Entries, 320, "Expected 320 log entries in the second batch")
				// verify batch for first log
				require.Contains(t, batch2.LogType, "WINEVTLOGS")
				require.Contains(t, batch2.Source.Namespace, "test")
				require.Len(t, batch2.Source.Labels, 2)

				batch3 := requests[2].Batch
				require.Len(t, batch3.Entries, 320, "Expected 320 log entries in the third batch")
				// verify batch for first log
				require.Contains(t, batch3.LogType, "WINEVTLOGS")
				require.Contains(t, batch3.Source.Namespace, "test")
				require.Len(t, batch3.Source.Labels, 2)

				batch4 := requests[3].Batch
				require.Len(t, batch4.Entries, 320, "Expected 320 log entries in the fourth batch")
				// verify batch for first log
				require.Contains(t, batch4.LogType, "WINEVTLOGS")
				require.Contains(t, batch4.Source.Namespace, "test")
				require.Len(t, batch4.Source.Labels, 2)

				// verify ingestion labels
				for _, req := range requests {
					for _, label := range req.Batch.Source.Labels {
						require.Contains(t, []string{
							"key1",
							"key2",
							"key3",
							"key4",
						}, label.Key)
						require.Contains(t, []string{
							"value1",
							"value2",
							"value3",
							"value4",
						}, label.Value)
					}
				}
			},
		},
		{
			name: "Unsplittable batch, single log exceeds max request size",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := tokenWithLength(5242881)
				record1.Body().SetStr(string(body))
				record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// verify 1 request, with 1 batch
				require.Len(t, requests, 0, "Expected a zero requests")
			},
		},
		{
			name: "Multiple valid log records + unsplittable log entries",
			cfg: marshal.Config{
				CustomerID:            uuid.New().String(),
				LogType:               "WINEVTLOG",
				RawLogField:           "body",
				OverrideLogType:       false,
				BatchLogCountLimit:    1000,
				BatchRequestSizeLimit: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				tooLargeBody := string(tokenWithLength(5242881))
				// first normal log, then impossible to split log
				logRecords1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				record1 := logRecords1.AppendEmpty()
				record1.Body().SetStr("First log message")
				tooLargeRecord1 := logRecords1.AppendEmpty()
				tooLargeRecord1.Body().SetStr(tooLargeBody)
				// first impossible to split log, then normal log
				logRecords2 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				tooLargeRecord2 := logRecords2.AppendEmpty()
				tooLargeRecord2.Body().SetStr(tooLargeBody)
				record2 := logRecords2.AppendEmpty()
				record2.Body().SetStr("Second log message")
				return logs
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest, _ time.Time) {
				// this is a kind of weird edge case, the overly large logs makes the final requests quite inefficient, but it's going to be so rare that the inefficiency isn't a real concern
				require.Len(t, requests, 2, "Expected two batch requests")
				batch1 := requests[0].Batch
				require.Len(t, batch1.Entries, 1, "Expected one log entry in the first batch")
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(batch1.Entries[0].Data))

				batch2 := requests[1].Batch
				require.Len(t, batch2.Entries, 1, "Expected one log entry in the second batch")
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(batch2.Entries[0].Data))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaler, err := marshal.NewGRPC(tt.cfg, component.TelemetrySettings{Logger: logger})
			require.NoError(t, err)

			logs := tt.logRecords()
			requests, err := marshaler.MarshalLogs(context.Background(), logs)
			require.NoError(t, err)

			tt.expectations(t, requests, marshaler.StartTime())
		})
	}
}
