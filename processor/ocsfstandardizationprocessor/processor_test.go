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

// accountChangeInputBody returns a log body with all fields needed to produce
// a valid AccountChange (3001) event when used with accountChangeFieldMappings.
func accountChangeInputBody() map[string]any {
	return map[string]any{
		"activity": 1,
		"category": 3,
		"severity": 1,
		"time":     int64(1234567890),
		"type":     300101,
		"user": map[string]any{
			"type_id": 1,
			"name":    "testuser",
		},
		"product": map[string]any{
			"vendor_name": "test-vendor",
			"name":        "test-product",
		},
	}
}

// accountChangeExpectedBody returns the expected output body for a valid AccountChange event.
func accountChangeExpectedBody(version string) map[string]any {
	return map[string]any{
		"class_uid":    int64(3001),
		"activity_id":  int64(1),
		"category_uid": int64(3),
		"severity_id":  int64(1),
		"time":         int64(1234567890),
		"type_uid":     int64(300101),
		"user": map[string]any{
			"type_id": int64(1),
			"name":    "testuser",
		},
		"metadata": map[string]any{
			"version": version,
			"product": map[string]any{
				"vendor_name": "test-vendor",
				"name":        "test-product",
			},
		},
	}
}

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
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3001,
						FieldMappings: append(accountChangeFieldMappings,
							FieldMapping{From: "body.msg", To: "message"},
						),
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["msg"] = "test message"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectedBody: func() map[string]any {
				expected := accountChangeExpectedBody("1.0.0")
				expected["message"] = "test message"
				return expected
			}(),
			expectedCount: 1,
		},
		{
			name: "filter matches",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						Filter:        `body.match == "yes"`,
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["match"] = "yes"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectedBody:  accountChangeExpectedBody("1.0.0"),
			expectedCount: 1,
		},
		{
			name: "filter does not match - drops log",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						Filter:        `body.match == "yes"`,
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["match"] = "no"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "default value used when from is missing",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3001,
						FieldMappings: []FieldMapping{
							{To: "activity_id", Default: 99},
							{From: "body.category", To: "category_uid"},
							{From: "body.severity", To: "severity_id"},
							{From: "body.time", To: "time"},
							{From: "body.type", To: "type_uid"},
							{From: "body.user", To: "user"},
							{From: "body.product", To: "metadata.product"},
						},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				return ld
			},
			expectedBody: func() map[string]any {
				expected := accountChangeExpectedBody("1.0.0")
				expected["activity_id"] = int64(99)
				return expected
			}(),
			expectedCount: 1,
		},
		{
			name: "default value used when from expression evaluates to nil",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3001,
						FieldMappings: append(accountChangeFieldMappings,
							FieldMapping{From: "body.missing_field", To: "status", Default: "unknown"},
						),
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				return ld
			},
			expectedBody: func() map[string]any {
				expected := accountChangeExpectedBody("1.0.0")
				expected["status"] = "unknown"
				return expected
			}(),
			expectedCount: 1,
		},
		{
			name: "no filter matches all logs",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				return ld
			},
			expectedBody:  accountChangeExpectedBody("1.0.0"),
			expectedCount: 1,
		},
		{
			name: "first matching event mapping wins",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						Filter:  "true",
						ClassID: 3001,
						FieldMappings: append(accountChangeFieldMappings,
							FieldMapping{From: "body.msg", To: "message"},
						),
					},
					{
						Filter:  "true",
						ClassID: 3001,
						FieldMappings: append(accountChangeFieldMappings,
							FieldMapping{From: "body.msg", To: "raw_data"},
						),
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["msg"] = "test"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectedBody: func() map[string]any {
				expected := accountChangeExpectedBody("1.0.0")
				expected["message"] = "test"
				return expected
			}(),
			expectedCount: 1,
		},
		{
			name: "resource attributes accessible in filter",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						Filter:        `resource.host == "web-01"`,
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				rl := ld.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("host", "web-01")
				record := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				return ld
			},
			expectedBody:  accountChangeExpectedBody("1.0.0"),
			expectedCount: 1,
		},
		{
			name: "drops empty resource and scope logs after filtering",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						Filter:        `body.keep == true`,
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				// First resource - all logs will be dropped
				rl1 := ld.ResourceLogs().AppendEmpty()
				record1 := rl1.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body1 := accountChangeInputBody()
				body1["keep"] = false
				err := record1.Body().SetEmptyMap().FromRaw(body1)
				require.NoError(t, err)
				// Second resource - log will be kept
				rl2 := ld.ResourceLogs().AppendEmpty()
				record2 := rl2.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body2 := accountChangeInputBody()
				body2["keep"] = true
				err = record2.Body().SetEmptyMap().FromRaw(body2)
				require.NoError(t, err)
				return ld
			},
			expectedBody:  accountChangeExpectedBody("1.0.0"),
			expectedCount: 1,
		},
		{
			name: "maps from attributes",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3001,
						FieldMappings: append(accountChangeFieldMappings,
							FieldMapping{From: "attributes.service", To: "message"},
						),
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				record.Attributes().PutStr("service", "auth-api")
				return ld
			},
			expectedBody: func() map[string]any {
				expected := accountChangeExpectedBody("1.0.0")
				expected["message"] = "auth-api"
				return expected
			}(),
			expectedCount: 1,
		},
		{
			name: "maps from resource attributes",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3001,
						FieldMappings: append(accountChangeFieldMappings,
							FieldMapping{From: "resource.host", To: "message"},
						),
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				rl := ld.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("host", "web-01")
				record := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				return ld
			},
			expectedBody: func() map[string]any {
				expected := accountChangeExpectedBody("1.0.0")
				expected["message"] = "web-01"
				return expected
			}(),
			expectedCount: 1,
		},
		{
			name: "maps from mixed sources",
			config: &Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3001,
						FieldMappings: append(accountChangeFieldMappings,
							FieldMapping{From: "resource.host", To: "device.hostname"},
							FieldMapping{From: "attributes.user_agent", To: "http_request.user_agent"},
							FieldMapping{From: "body.src_ip", To: "src_endpoint.ip"},
							FieldMapping{To: "status", Default: "active"},
						),
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				rl := ld.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("host", "proxy-01")
				record := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["src_ip"] = "192.168.1.1"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				record.Attributes().PutStr("user_agent", "Mozilla/5.0")
				return ld
			},
			expectedBody: func() map[string]any {
				expected := accountChangeExpectedBody("1.0.0")
				expected["device"] = map[string]any{"hostname": "proxy-01"}
				expected["http_request"] = map[string]any{"user_agent": "Mozilla/5.0"}
				expected["src_endpoint"] = map[string]any{"ip": "192.168.1.1"}
				expected["status"] = "active"
				return expected
			}(),
			expectedCount: 1,
		},
		{
			name: "no event mappings drops all logs",
			config: &Config{
				OCSFVersion:   OCSFVersion1_0_0,
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

func TestProcessLogsValidation(t *testing.T) {
	tests := []struct {
		name          string
		eventMappings []EventMapping
		inputLogs     func() plog.Logs
		expectDropped bool
	}{
		{
			name: "valid body passes validation",
			eventMappings: []EventMapping{
				{
					ClassID:       3001,
					FieldMappings: accountChangeFieldMappings,
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				return ld
			},
		},
		{
			name: "missing required fields drops log",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: []FieldMapping{
						{From: "body.msg", To: "message"},
					},
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(map[string]any{"msg": "test"})
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "unknown class UID drops log",
			eventMappings: []EventMapping{
				{
					ClassID:       9999,
					FieldMappings: accountChangeFieldMappings,
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				err := record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "invalid regex drops log - bad email",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.email", To: "user.email_addr"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["email"] = "not-an-email"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "valid regex passes - valid email",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.email", To: "user.email_addr"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["email"] = "user@example.com"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
		},
		{
			name: "invalid regex drops log - bad IP",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.ip", To: "src_endpoint.ip"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["ip"] = "not-an-ip"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "valid regex passes - valid IP",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.ip", To: "src_endpoint.ip"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["ip"] = "192.168.1.1"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
		},
		{
			name: "range violation drops log - port too high",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.port", To: "src_endpoint.port"},
						FieldMapping{From: "body.ip", To: "src_endpoint.ip"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["port"] = 70000
				body["ip"] = "10.0.0.1"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "range violation drops log - port negative",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.port", To: "src_endpoint.port"},
						FieldMapping{From: "body.ip", To: "src_endpoint.ip"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["port"] = -1
				body["ip"] = "10.0.0.1"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "valid range passes - port in range",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.port", To: "src_endpoint.port"},
						FieldMapping{From: "body.ip", To: "src_endpoint.ip"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["port"] = 8080
				body["ip"] = "10.0.0.1"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
		},
		{
			name: "maxlen violation drops log - IP too long",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.ip", To: "src_endpoint.ip"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				// IP max_len is 40; provide a string longer than 40 chars
				body["ip"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
		{
			name: "maxlen violation drops log - MAC too long",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.mac", To: "src_endpoint.mac"},
						FieldMapping{From: "body.ip", To: "src_endpoint.ip"},
					),
				},
			},
			inputLogs: func() plog.Logs {
				ld := plog.NewLogs()
				record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				body := accountChangeInputBody()
				body["ip"] = "10.0.0.1"
				// MAC max_len is 32; provide a string longer than 32 chars
				body["mac"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				err := record.Body().SetEmptyMap().FromRaw(body)
				require.NoError(t, err)
				return ld
			},
			expectDropped: true,
		},
	}

	for _, version := range OCSFVersions {
		for _, tt := range tests {
			t.Run(string(version)+"/"+tt.name, func(t *testing.T) {
				cfg := &Config{
					OCSFVersion:   version,
					EventMappings: tt.eventMappings,
				}
				processor, err := newOCSFStandardizationProcessor(zap.NewNop(), cfg)
				require.NoError(t, err)

				result, err := processor.processLogs(context.Background(), tt.inputLogs())
				require.NoError(t, err)

				if tt.expectDropped {
					require.Equal(t, 0, countLogRecords(result), "expected all logs to be dropped")
				} else {
					require.Equal(t, 1, countLogRecords(result))
				}
			})
		}
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
