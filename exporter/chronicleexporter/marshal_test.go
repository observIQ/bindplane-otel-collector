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
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"golang.org/x/exp/rand"
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
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           true,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "attributes",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
				CustomerID:                uuid.New().String(),
				LogType:                   "DEFAULT",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
				CustomerID:                uuid.New().String(),
				RawLogField:               "body",
				OverrideLogType:           true,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "DEFAULT", // This should be overridden by the log_type attribute
				RawLogField:               "body",
				OverrideLogType:           true,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
				CustomerID:                uuid.New().String(),
				LogType:                   "DEFAULT", // This should be overridden by the chronicle_log_type attribute
				RawLogField:               "body",
				OverrideLogType:           true,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"chronicle_log_type": "ASOC_ALERT", "chronicle_namespace": "test", `chronicle_ingestion_label["realkey1"]`: "realvalue1", `chronicle_ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := tokenWithLength(5242881)
				record1.Body().SetStr(string(body))
				record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				return logs
			},

			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
				// verify 1 request, with 1 batch
				require.Len(t, requests, 0, "Expected a zero requests")
			},
		},
		{
			name: "Multiple valid log records + unsplittable log entries",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitGRPC:    1000,
				BatchRequestSizeLimitGRPC: 5242880,
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
			expectations: func(t *testing.T, requests []*api.BatchCreateLogsRequest) {
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
		expectations func(t *testing.T, requests map[string][]*api.ImportLogsRequest)
	}{
		{
			name: "Single log record with expected data",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				Protocol:                  protocolHTTPS,
				Project:                   "test-project",
				Location:                  "us",
				Forwarder:                 uuid.New().String(),
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			labels: []*api.Label{
				{Key: "env", Value: "prod"},
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", map[string]any{"log_type": "WINEVTLOG", "namespace": "test"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1)
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
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
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1, "Expected a single batch request")
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
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
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "attributes",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1)
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
				// Assuming the attributes are marshaled into the Data field as a JSON string
				expectedData := `{"key1":"value1", "log_type":"WINEVTLOG", "namespace":"test", "chronicle_ingestion_label[\"key1\"]": "value1", "chronicle_ingestion_label[\"key2\"]": "value2"}`
				actualData := string(logs[0].Data)
				require.JSONEq(t, expectedData, actualData, "Log attributes should match expected")
			},
		},
		{
			name: "No log records",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "DEFAULT",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return plog.NewLogs() // No log records added
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 0, "Expected no requests due to no log records")
			},
		},
		{
			name: "No log type set in config or attributes",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "attributes",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1)
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
				// Assuming the attributes are marshaled into the Data field as a JSON string
				expectedData := `{"key1":"value1", "log_type":"WINEVTLOG", "namespace":"test", "chronicle_ingestion_label[\"key1\"]": "value1", "chronicle_ingestion_label[\"key2\"]": "value2"}`
				actualData := string(logs[0].Data)
				require.JSONEq(t, expectedData, actualData, "Log attributes should match expected")
			},
		},
		{
			name: "Multiple log records with duplicate data, no log type in attributes",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				// verify one request for log type in config
				require.Len(t, requests, 1, "Expected a single batch request")
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
				// verify batch source labels
				require.Len(t, logs[0].Labels, 2)
				require.Len(t, logs, 2, "Expected two log entries in the batch")
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(logs[0].Data))
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(logs[1].Data))
			},
		},
		{
			name: "Multiple log records with different data, no log type in attributes",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				// verify one request for one log type
				require.Len(t, requests, 1, "Expected a single batch request")
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
				require.Len(t, logs, 2, "Expected two log entries in the batch")
				require.Equal(t, "", logs[0].EnvironmentNamespace)
				// verify batch source labels
				require.Len(t, logs[0].Labels, 2)
				require.Len(t, logs[1].Labels, 2)
				// Verifying the first log entry data
				require.Equal(t, "First log message", string(logs[0].Data))
				// Verifying the second log entry data
				require.Equal(t, "Second log message", string(logs[1].Data))
			},
		},
		{
			name: "Override log type with attribute",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "DEFAULT", // This should be overridden by the log_type attribute
				RawLogField:               "body",
				OverrideLogType:           true,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"log_type": "windows_event.application", "namespace": "test", `ingestion_label["realkey1"]`: "realvalue1", `ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1)
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
				require.NotEqual(t, len(logs), 0)
			},
		},
		{
			name: "Override log type with chronicle attribute",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "DEFAULT", // This should be overridden by the chronicle_log_type attribute
				RawLogField:               "body",
				OverrideLogType:           true,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"chronicle_log_type": "ASOC_ALERT", "chronicle_namespace": "test", `chronicle_ingestion_label["realkey1"]`: "realvalue1", `chronicle_ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1)
				logs := requests["ASOC_ALERT"][0].GetInlineSource().Logs
				require.Equal(t, "test", logs[0].EnvironmentNamespace, "Expected namespace to be overridden by attribute")
				expectedLabels := map[string]string{
					"realkey1": "realvalue1",
					"realkey2": "realvalue2",
				}
				for key, label := range logs[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Multiple log records with duplicate data, log type in attributes",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				// verify 1 request, 2 batches for same log type
				require.Len(t, requests, 1, "Expected a single batch request")
				logs := requests["WINEVTLOGS"][0].GetInlineSource().Logs
				require.Len(t, logs, 2, "Expected two log entries in the batch")
				// verify variables
				require.Equal(t, "test1", logs[0].EnvironmentNamespace)
				require.Len(t, logs[0].Labels, 2)
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
				for key, label := range logs[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Multiple log records with different data, log type in attributes",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
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

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
					"key4": "value4",
				}
				// verify 2 requests, with 1 batch for different log types
				require.Len(t, requests, 2, "Expected a two batch request")

				logs1 := requests["WINEVTLOGS1"][0].GetInlineSource().Logs
				require.Len(t, logs1, 1, "Expected one log entries in the batch")
				// verify variables for first log
				require.Equal(t, logs1[0].EnvironmentNamespace, "test1")
				require.Len(t, logs1[0].Labels, 2)
				for key, label := range logs1[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs2 := requests["WINEVTLOGS2"][0].GetInlineSource().Logs
				require.Len(t, logs2, 1, "Expected one log entries in the batch")
				// verify variables for second log
				require.Equal(t, logs2[0].EnvironmentNamespace, "test2")
				require.Len(t, logs2[0].Labels, 2)
				for key, label := range logs2[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Many log records all one batch",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				for i := 0; i < 1000; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr("First log message")
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}

				return logs
			},

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
				// verify 1 requests
				require.Len(t, requests, 1, "Expected a one batch request")

				logs1 := requests["WINEVTLOGS1"][0].GetInlineSource().Logs
				require.Len(t, logs1, 1000, "Expected one thousand log entries in the batch")
				// verify variables for first log
				require.Equal(t, logs1[0].EnvironmentNamespace, "test1")
				require.Len(t, logs1[0].Labels, 2)
				for key, label := range logs1[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Many log records split into two batches",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				for i := 0; i < 1001; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr("First log message")
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}

				return logs
			},

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
				// verify 1 request log type
				require.Len(t, requests, 1, "Expected one log type for the requests")
				winEvtLogRequests := requests["WINEVTLOGS1"]
				require.Len(t, winEvtLogRequests, 2, "Expected two batches")

				logs1 := winEvtLogRequests[0].GetInlineSource().Logs
				require.Len(t, logs1, 500, "Expected 500 log entries in the first batch")
				// verify variables for first log
				require.Equal(t, logs1[0].EnvironmentNamespace, "test1")
				require.Len(t, logs1[0].Labels, 2)
				for key, label := range logs1[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs2 := winEvtLogRequests[1].GetInlineSource().Logs
				require.Len(t, logs2, 501, "Expected 501 log entries in the second batch")
				// verify variables for first log
				require.Equal(t, logs2[0].EnvironmentNamespace, "test1")
				require.Len(t, logs2[0].Labels, 2)
				for key, label := range logs2[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Recursively split batch multiple times because too many logs",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				for i := 0; i < 2002; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr("First log message")
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}

				return logs
			},

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
				// verify 1 request log type
				require.Len(t, requests, 1, "Expected one log type for the requests")
				winEvtLogRequests := requests["WINEVTLOGS1"]
				require.Len(t, winEvtLogRequests, 4, "Expected four batches")

				logs1 := winEvtLogRequests[0].GetInlineSource().Logs
				require.Len(t, logs1, 500, "Expected 500 log entries in the first batch")
				// verify variables for first log
				require.Equal(t, logs1[0].EnvironmentNamespace, "test1")
				require.Len(t, logs1[0].Labels, 2)
				for key, label := range logs1[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs2 := winEvtLogRequests[1].GetInlineSource().Logs
				require.Len(t, logs2, 501, "Expected 501 log entries in the second batch")
				// verify variables for first log
				require.Equal(t, logs2[0].EnvironmentNamespace, "test1")
				require.Len(t, logs2[0].Labels, 2)
				for key, label := range logs2[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs3 := winEvtLogRequests[2].GetInlineSource().Logs
				require.Len(t, logs3, 500, "Expected 500 log entries in the third batch")
				// verify variables for first log
				require.Equal(t, logs3[0].EnvironmentNamespace, "test1")
				require.Len(t, logs3[0].Labels, 2)
				for key, label := range logs3[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs4 := winEvtLogRequests[3].GetInlineSource().Logs
				require.Len(t, logs4, 501, "Expected 501 log entries in the fourth batch")
				// verify variables for first log
				require.Equal(t, logs4[0].EnvironmentNamespace, "test1")
				require.Len(t, logs4[0].Labels, 2)
				for key, label := range logs4[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Many log records split into two batches because request size too large",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				// 8192 * 640 = 5242880
				body := tokenWithLength(8192)
				for i := 0; i < 640; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr(string(body))
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}

				return logs
			},

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
				// verify 1 request log type
				require.Len(t, requests, 1, "Expected one log type for the requests")
				winEvtLogRequests := requests["WINEVTLOGS1"]
				require.Len(t, winEvtLogRequests, 2, "Expected two batches")

				logs1 := winEvtLogRequests[0].GetInlineSource().Logs
				require.Len(t, logs1, 320, "Expected 320 log entries in the first batch")
				// verify variables for first log
				require.Equal(t, logs1[0].EnvironmentNamespace, "test1")
				require.Len(t, logs1[0].Labels, 2)
				for key, label := range logs1[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs2 := winEvtLogRequests[1].GetInlineSource().Logs
				require.Len(t, logs2, 320, "Expected 320 log entries in the second batch")
				// verify variables for first log
				require.Equal(t, logs2[0].EnvironmentNamespace, "test1")
				require.Len(t, logs2[0].Labels, 2)
				for key, label := range logs2[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Recursively split into batches because request size too large",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    2000,
				BatchRequestSizeLimitHTTP: 5242880,
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				// 8192 * 1280 = 5242880 * 2
				body := tokenWithLength(8192)
				for i := 0; i < 1280; i++ {
					record1 := logRecords.AppendEmpty()
					record1.Body().SetStr(string(body))
					record1.Attributes().FromRaw(map[string]any{"chronicle_log_type": "WINEVTLOGS1", "chronicle_namespace": "test1", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"})
				}

				return logs
			},

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				expectedLabels := map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
				// verify 1 request log type
				require.Len(t, requests, 1, "Expected one log type for the requests")
				winEvtLogRequests := requests["WINEVTLOGS1"]
				require.Len(t, winEvtLogRequests, 4, "Expected four batches")

				logs1 := winEvtLogRequests[0].GetInlineSource().Logs
				require.Len(t, logs1, 320, "Expected 320 log entries in the first batch")
				// verify variables for first log
				require.Equal(t, logs1[0].EnvironmentNamespace, "test1")
				require.Len(t, logs1[0].Labels, 2)
				for key, label := range logs1[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs2 := winEvtLogRequests[1].GetInlineSource().Logs
				require.Len(t, logs2, 320, "Expected 320 log entries in the second batch")
				// verify variables for first log
				require.Equal(t, logs2[0].EnvironmentNamespace, "test1")
				require.Len(t, logs2[0].Labels, 2)
				for key, label := range logs2[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs3 := winEvtLogRequests[2].GetInlineSource().Logs
				require.Len(t, logs3, 320, "Expected 320 log entries in the third batch")
				// verify variables for first log
				require.Equal(t, logs3[0].EnvironmentNamespace, "test1")
				require.Len(t, logs3[0].Labels, 2)
				for key, label := range logs3[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}

				logs4 := winEvtLogRequests[3].GetInlineSource().Logs
				require.Len(t, logs4, 320, "Expected 320 log entries in the fourth batch")
				// verify variables for first log
				require.Equal(t, logs4[0].EnvironmentNamespace, "test1")
				require.Len(t, logs4[0].Labels, 2)
				for key, label := range logs4[0].Labels {
					require.Equal(t, expectedLabels[key], label.Value, "Expected ingestion label to be overridden by attribute")
				}
			},
		},
		{
			name: "Unsplittable log record, single log exceeds request size limit",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 100000,
			},
			labels: []*api.Label{
				{Key: "env", Value: "staging"},
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				record1 := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetStr(string(tokenWithLength(100000)))
				return logs
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1, "Expected one log type")
				require.Len(t, requests["WINEVTLOG"], 0, "Expected WINEVTLOG log type to have zero requests")
			},
		},
		{
			name: "Unsplittable log record, single log exceeds request size limit, mixed with okay logs",
			cfg: Config{
				CustomerID:                uuid.New().String(),
				LogType:                   "WINEVTLOG",
				RawLogField:               "body",
				OverrideLogType:           false,
				BatchLogCountLimitHTTP:    1000,
				BatchRequestSizeLimitHTTP: 100000,
			},
			labels: []*api.Label{
				{Key: "env", Value: "staging"},
			},
			logRecords: func() plog.Logs {
				logs := plog.NewLogs()
				tooLargeBody := string(tokenWithLength(100001))
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest) {
				require.Len(t, requests, 1, "Expected one log type")
				winEvtLogRequests := requests["WINEVTLOG"]
				require.Len(t, winEvtLogRequests, 2, "Expected WINEVTLOG log type to have zero requests")

				logs1 := winEvtLogRequests[0].GetInlineSource().Logs
				require.Len(t, logs1, 1, "Expected 1 log entry in the first batch")
				require.Equal(t, string(logs1[0].Data), "First log message")

				logs2 := winEvtLogRequests[1].GetInlineSource().Logs
				require.Len(t, logs2, 1, "Expected 1 log entry in the second batch")
				require.Equal(t, string(logs2[0].Data), "Second log message")
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

func tokenWithLength(length int) []byte {
	charset := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
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

type getRawFieldCase struct {
	name         string
	field        string
	logRecord    plog.LogRecord
	scope        plog.ScopeLogs
	resource     plog.ResourceLogs
	expect       string
	expectErrStr string
}

// Used by tests and benchmarks
var getRawFieldCases = []getRawFieldCase{
	{
		name:  "String body",
		field: "body",
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Body().SetStr("<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='Service Control Manager' Guid='{555908d1-a6d7-4695-8e1e-26931d2012f4}' EventSourceName='Service Control Manager'/><EventID Qualifiers='16384'>7036</EventID><Version>0</Version><Level>4</Level><Task>0</Task><Opcode>0</Opcode><Keywords>0x8080000000000000</Keywords><TimeCreated SystemTime='2024-11-08T18:51:13.504187700Z'/><EventRecordID>3562</EventRecordID><Correlation/><Execution ProcessID='604' ThreadID='4792'/><Channel>System</Channel><Computer>WIN-L6PC55MPB98</Computer><Security/></System><EventData><Data Name='param1'>Print Spooler</Data><Data Name='param2'>stopped</Data><Binary>530070006F006F006C00650072002F0031000000</Binary></EventData></Event>")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='Service Control Manager' Guid='{555908d1-a6d7-4695-8e1e-26931d2012f4}' EventSourceName='Service Control Manager'/><EventID Qualifiers='16384'>7036</EventID><Version>0</Version><Level>4</Level><Task>0</Task><Opcode>0</Opcode><Keywords>0x8080000000000000</Keywords><TimeCreated SystemTime='2024-11-08T18:51:13.504187700Z'/><EventRecordID>3562</EventRecordID><Correlation/><Execution ProcessID='604' ThreadID='4792'/><Channel>System</Channel><Computer>WIN-L6PC55MPB98</Computer><Security/></System><EventData><Data Name='param1'>Print Spooler</Data><Data Name='param2'>stopped</Data><Binary>530070006F006F006C00650072002F0031000000</Binary></EventData></Event>",
	},
	{
		name:  "Empty body",
		field: "body",
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Body().SetStr("")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "",
	},
	{
		name:  "Map body",
		field: "body",
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Body().SetEmptyMap()
			lr.Body().Map().PutStr("param1", "Print Spooler")
			lr.Body().Map().PutStr("param2", "stopped")
			lr.Body().Map().PutStr("binary", "530070006F006F006C00650072002F0031000000")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   `{"binary":"530070006F006F006C00650072002F0031000000","param1":"Print Spooler","param2":"stopped"}`,
	},
	{
		name:  "Map body field",
		field: "body[\"param1\"]",
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Body().SetEmptyMap()
			lr.Body().Map().PutStr("param1", "Print Spooler")
			lr.Body().Map().PutStr("param2", "stopped")
			lr.Body().Map().PutStr("binary", "530070006F006F006C00650072002F0031000000")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "Print Spooler",
	},
	{
		name:  "Map body field missing",
		field: "body[\"missing\"]",
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Body().SetEmptyMap()
			lr.Body().Map().PutStr("param1", "Print Spooler")
			lr.Body().Map().PutStr("param2", "stopped")
			lr.Body().Map().PutStr("binary", "530070006F006F006C00650072002F0031000000")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "",
	},
	{
		name:  "Attribute log_type",
		field: `attributes["log_type"]`,
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Attributes().PutStr("status", "200")
			lr.Attributes().PutStr("log.file.name", "/var/log/containers/agent_agent_ns.log")
			lr.Attributes().PutStr("log_type", "WINEVTLOG")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "WINEVTLOG",
	},
	{
		name:  "Attribute log_type missing",
		field: `attributes["log_type"]`,
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Attributes().PutStr("status", "200")
			lr.Attributes().PutStr("log.file.name", "/var/log/containers/agent_agent_ns.log")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "",
	},
	{
		name:  "Attribute chronicle_log_type",
		field: `attributes["chronicle_log_type"]`,
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Attributes().PutStr("status", "200")
			lr.Attributes().PutStr("log.file.name", "/var/log/containers/agent_agent_ns.log")
			lr.Attributes().PutStr("chronicle_log_type", "MICROSOFT_SQL")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "MICROSOFT_SQL",
	},
	{
		name:  "Attribute chronicle_namespace",
		field: `attributes["chronicle_namespace"]`,
		logRecord: func() plog.LogRecord {
			lr := plog.NewLogRecord()
			lr.Attributes().PutStr("status", "200")
			lr.Attributes().PutStr("log_type", "k8s-container")
			lr.Attributes().PutStr("log.file.name", "/var/log/containers/agent_agent_ns.log")
			lr.Attributes().PutStr("chronicle_log_type", "MICROSOFT_SQL")
			lr.Attributes().PutStr("chronicle_namespace", "test")
			return lr
		}(),
		scope:    plog.NewScopeLogs(),
		resource: plog.NewResourceLogs(),
		expect:   "test",
	},
}

func Test_getRawField(t *testing.T) {
	for _, tc := range getRawFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &protoMarshaler{}
			m.teleSettings.Logger = zap.NewNop()

			ctx := context.Background()

			rawField, err := m.getRawField(ctx, tc.field, tc.logRecord, tc.scope, tc.resource)
			if tc.expectErrStr != "" {
				require.Contains(t, err.Error(), tc.expectErrStr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expect, rawField)
		})
	}
}

func Benchmark_getRawField(b *testing.B) {
	m := &protoMarshaler{}
	m.teleSettings.Logger = zap.NewNop()

	ctx := context.Background()

	for _, tc := range getRawFieldCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = m.getRawField(ctx, tc.field, tc.logRecord, tc.scope, tc.resource)
			}
		})
	}
}
