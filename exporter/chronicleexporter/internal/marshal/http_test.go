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
)

func TestHTTP(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name         string
		cfg          marshal.HTTPConfig
		labels       []*api.Label
		logRecords   func() plog.Logs
		expectations func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time)
	}{
		{
			name: "Single log record with expected data",
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
				Project:   "test-project",
				Location:  "us",
				Forwarder: uuid.New().String(),
			},
			labels: []*api.Label{
				{Key: "env", Value: "prod"},
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Test log message", map[string]any{"log_type": "WINEVTLOG", "namespace": "test"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, _ time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "attributes",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "DEFAULT",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return plog.NewLogs() // No log records added
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
				require.Len(t, requests, 0, "Expected no requests due to no log records")
			},
		},
		{
			name: "No log type set in config or attributes",
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "attributes",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
			},
			labels: []*api.Label{},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("", map[string]any{"key1": "value1", "log_type": "WINEVTLOG", "namespace": "test", `chronicle_ingestion_label["key1"]`: "value1", `chronicle_ingestion_label["key2"]`: "value2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "DEFAULT", // This should be overridden by the log_type attribute
					RawLogField:           "body",
					OverrideLogType:       true,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"log_type": "windows_event.application", "namespace": "test", `ingestion_label["realkey1"]`: "realvalue1", `ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
				require.Len(t, requests, 1)
				logs := requests["WINEVTLOG"][0].GetInlineSource().Logs
				require.NotEqual(t, len(logs), 0)
			},
		},
		{
			name: "Override log type with chronicle attribute",
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "DEFAULT", // This should be overridden by the chronicle_log_type attribute
					RawLogField:           "body",
					OverrideLogType:       true,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
			},
			logRecords: func() plog.Logs {
				return mockLogs(mockLogRecord("Log with overridden type", map[string]any{"chronicle_log_type": "ASOC_ALERT", "chronicle_namespace": "test", `chronicle_ingestion_label["realkey1"]`: "realvalue1", `chronicle_ingestion_label["realkey2"]`: "realvalue2"}))
			},
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 5242880,
				},
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

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    2000,
					BatchRequestSizeLimit: 5242880,
				},
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

			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 100000,
				},
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
				require.Len(t, requests, 1, "Expected one log type")
				require.Len(t, requests["WINEVTLOG"], 0, "Expected WINEVTLOG log type to have zero requests")
			},
		},
		{
			name: "Unsplittable log record, single log exceeds request size limit, mixed with okay logs",
			cfg: marshal.HTTPConfig{
				Config: marshal.Config{
					CustomerID:            uuid.New().String(),
					LogType:               "WINEVTLOG",
					RawLogField:           "body",
					OverrideLogType:       false,
					BatchLogCountLimit:    1000,
					BatchRequestSizeLimit: 100000,
				},
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
			expectations: func(t *testing.T, requests map[string][]*api.ImportLogsRequest, startTime time.Time) {
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
			marshaler, err := marshal.NewHTTP(tt.cfg, component.TelemetrySettings{Logger: logger})
			require.NoError(t, err)

			logs := tt.logRecords()
			requests, err := marshaler.MarshalLogs(context.Background(), logs)
			require.NoError(t, err)

			tt.expectations(t, requests, marshaler.StartTime())
		})
	}
}
