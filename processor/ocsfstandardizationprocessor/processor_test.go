// Copyright  observIQ, Inc.
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

package ocsfstandardizationprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestProcessLogs(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		inputLogs     func() plog.Logs
		expectedBody  map[string]any
		expectDropped bool
		expectedCount int
	}{
		{
			name: "basic field mapping",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "body.src", To: "dst_endpoint.ip"},
							{From: "body.msg", To: "message"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"src": "10.0.0.1",
					"msg": "test message",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(1001),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"dst_endpoint": map[string]any{
					"ip": "10.0.0.1",
				},
				"message": "test message",
			},
			expectedCount: 1,
		},
		{
			name: "filter matches",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  `body.type == "auth"`,
						ClassID: 3002,
						FieldMappings: []FieldMapping{
							{From: "body.user", To: "actor.user.name"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"type": "auth",
					"user": "admin",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(3002),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"actor": map[string]any{
					"user": map[string]any{
						"name": "admin",
					},
				},
			},
			expectedCount: 1,
		},
		{
			name: "filter does not match - drops log",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  `body.type == "auth"`,
						ClassID: 3002,
						FieldMappings: []FieldMapping{
							{From: "body.user", To: "actor.user.name"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"type": "network",
					"user": "admin",
				})
				return ld
			},
			expectDropped: true,
		},
		{
			name: "default value used when from is missing",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 4001,
						FieldMappings: []FieldMapping{
							{To: "severity_id", Default: 1},
							{From: "body.msg", To: "message"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"msg": "hello",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(4001),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"severity_id": int64(1),
				"message":     "hello",
			},
			expectedCount: 1,
		},
		{
			name: "default value used when from expression evaluates to nil",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 4001,
						FieldMappings: []FieldMapping{
							{From: "body.missing_field", To: "status", Default: "unknown"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"msg": "hello",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(4001),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"status": "unknown",
			},
			expectedCount: 1,
		},
		{
			name: "no filter matches all logs",
			config: &Config{
				OCSFVersion: OCSFVersion1_7_0,
				EventMappings: []EventMapping{
					{
						ClassID: 2001,
						FieldMappings: []FieldMapping{
							{From: "body.msg", To: "message"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"msg": "catch all",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(2001),
				"metadata": map[string]any{
					"version": "1.7.0",
				},
				"message": "catch all",
			},
			expectedCount: 1,
		},
		{
			name: "first matching event mapping wins",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  "true",
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "body.msg", To: "message"},
						},
					},
					{
						Filter:  "true",
						ClassID: 2002,
						FieldMappings: []FieldMapping{
							{From: "body.msg", To: "other"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"msg": "test",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(1001),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"message": "test",
			},
			expectedCount: 1,
		},
		{
			name: "resource attributes accessible in filter",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  `resource.host == "web-01"`,
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "body.msg", To: "message"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				rl := ld.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("host", "web-01")
				record := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{
					"msg": "from web-01",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(1001),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"message": "from web-01",
			},
			expectedCount: 1,
		},
		{
			name: "drops empty resource and scope logs after filtering",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  `body.keep == true`,
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "body.msg", To: "message"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				// First resource - all logs will be dropped
				rl1 := ld.ResourceLogs().AppendEmpty()
				record1 := rl1.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record1.Body().SetEmptyMap().FromRaw(map[string]any{
					"keep": false,
					"msg":  "drop me",
				})
				// Second resource - log will be kept
				rl2 := ld.ResourceLogs().AppendEmpty()
				record2 := rl2.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record2.Body().SetEmptyMap().FromRaw(map[string]any{
					"keep": true,
					"msg":  "keep me",
				})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(1001),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"message": "keep me",
			},
			expectedCount: 1,
		},
		{
			name: "maps from attributes",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "attributes.service", To: "metadata.product.name"},
							{From: "attributes.env", To: "metadata.product.env"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetStr("some log")
				record.Attributes().PutStr("service", "auth-api")
				record.Attributes().PutStr("env", "production")
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(1001),
				"metadata": map[string]any{
					"version": "1.3.0",
					"product": map[string]any{
						"name": "auth-api",
						"env":  "production",
					},
				},
			},
			expectedCount: 1,
		},
		{
			name: "maps from resource attributes",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "resource.host", To: "device.hostname"},
							{From: "resource.os", To: "device.os.name"},
							{From: "body.msg", To: "message"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				rl := ld.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("host", "web-01")
				rl.Resource().Attributes().PutStr("os", "linux")
				record := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"msg": "test"})
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(1001),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"device": map[string]any{
					"hostname": "web-01",
					"os": map[string]any{
						"name": "linux",
					},
				},
				"message": "test",
			},
			expectedCount: 1,
		},
		{
			name: "maps from mixed sources",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3002,
						FieldMappings: []FieldMapping{
							{From: "resource.host", To: "device.hostname"},
							{From: "attributes.user_agent", To: "http_request.user_agent"},
							{From: "body.src_ip", To: "src_endpoint.ip"},
							{To: "category_uid", Default: 3},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				rl := ld.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("host", "proxy-01")
				record := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"src_ip": "192.168.1.1"})
				record.Attributes().PutStr("user_agent", "Mozilla/5.0")
				return ld
			},
			expectedBody: map[string]any{
				"class_uid": int64(3002),
				"metadata": map[string]any{
					"version": "1.3.0",
				},
				"device": map[string]any{
					"hostname": "proxy-01",
				},
				"http_request": map[string]any{
					"user_agent": "Mozilla/5.0",
				},
				"src_endpoint": map[string]any{
					"ip": "192.168.1.1",
				},
				"category_uid": int64(3),
			},
			expectedCount: 1,
		},
		{
			name: "no event mappings drops all logs",
			config: &Config{
				OCSFVersion:   OCSFVersion1_3_0,
				EventMappings: []EventMapping{},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetStr("test")
				return ld
			},
			expectDropped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := newOCSFStandardizationProcessor(zap.NewNop(), tt.config)
			require.NoError(t, err)

			result, err := processor.processLogs(context.Background(), tt.inputLogs())
			require.NoError(t, err)

			if tt.expectDropped {
				require.Equal(t, 0, result.ResourceLogs().Len(), "expected all logs to be dropped")
				return
			}

			require.Equal(t, tt.expectedCount, countLogRecords(result))
			body := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body()
			require.Equal(t, tt.expectedBody, body.Map().AsRaw())
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    any
		existing map[string]any
		expected map[string]any
	}{
		{
			name:     "single key",
			path:     "message",
			value:    "hello",
			existing: map[string]any{},
			expected: map[string]any{"message": "hello"},
		},
		{
			name:     "nested path",
			path:     "dst_endpoint.ip",
			value:    "10.0.0.1",
			existing: map[string]any{},
			expected: map[string]any{
				"dst_endpoint": map[string]any{
					"ip": "10.0.0.1",
				},
			},
		},
		{
			name:     "deep nested path",
			path:     "actor.user.email_addr",
			value:    "test@example.com",
			existing: map[string]any{},
			expected: map[string]any{
				"actor": map[string]any{
					"user": map[string]any{
						"email_addr": "test@example.com",
					},
				},
			},
		},
		{
			name:  "merges with existing nested map",
			path:  "metadata.product",
			value: "test",
			existing: map[string]any{
				"metadata": map[string]any{
					"version": "1.3.0",
				},
			},
			expected: map[string]any{
				"metadata": map[string]any{
					"version": "1.3.0",
					"product": "test",
				},
			},
		},
		{
			name:  "overwrites non-map intermediate",
			path:  "metadata.version",
			value: "1.3.0",
			existing: map[string]any{
				"metadata": "not a map",
			},
			expected: map[string]any{
				"metadata": map[string]any{
					"version": "1.3.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setNestedValue(tt.existing, tt.path, tt.value)
			require.Equal(t, tt.expected, tt.existing)
		})
	}
}

func TestNewOCSFStandardizationProcessor(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name: "valid config",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  "true",
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "body.src", To: "message"},
						},
					},
				},
			},
		},
		{
			name: "valid config with default only field mapping",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{To: "severity_id", Default: 1},
						},
					},
				},
			},
		},
		{
			name: "invalid from expression",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "|||invalid|||", To: "message"},
						},
					},
				},
			},
			wantErr: "compiling from expression",
		},
		{
			name: "invalid filter expression",
			config: &Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  "|||invalid|||",
						ClassID: 1001,
					},
				},
			},
			wantErr: "compiling filter expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := newOCSFStandardizationProcessor(zap.NewNop(), tt.config)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, processor)
			} else {
				require.NoError(t, err)
				require.NotNil(t, processor)
			}
		})
	}
}

func countLogRecords(ld plog.Logs) int {
	count := 0
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		for j := 0; j < ld.ResourceLogs().At(i).ScopeLogs().Len(); j++ {
			count += ld.ResourceLogs().At(i).ScopeLogs().At(j).LogRecords().Len()
		}
	}
	return count
}
