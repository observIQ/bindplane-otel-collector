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

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/marshal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"golang.org/x/exp/rand"
)

func Test_GetRawField(t *testing.T) {
	for _, tc := range getRawFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			set := componenttest.NewNopTelemetrySettings()
			rawField, err := marshal.GetRawField(ctx, set, tc.field, tc.logRecord, tc.scope, tc.resource)
			if tc.expectErrStr != "" {
				require.Contains(t, err.Error(), tc.expectErrStr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expect, rawField)
		})
	}
}

func Benchmark_GetRawField(b *testing.B) {
	ctx := context.Background()
	set := componenttest.NewNopTelemetrySettings()
	for _, tc := range getRawFieldCases {
		b.ResetTimer()
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = marshal.GetRawField(ctx, set, tc.field, tc.logRecord, tc.scope, tc.resource)
			}
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
