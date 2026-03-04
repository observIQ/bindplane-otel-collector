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

// authenticationInputBody returns a realistic Authentication event input body
// with fields that exercise timestamp coercion, boolean coercion, nested objects,
// and string-typed IP fields for validation.
func authenticationInputBody() map[string]any {
	return map[string]any{
		"event_type":    "authentication",
		"activity":      "1",                    // string that needs integer coercion
		"category":      3,                      // already int
		"severity":      "2",                    // string that needs integer coercion
		"time":          "2025-06-15T14:30:00Z", // RFC3339 string that needs timestamp coercion
		"start_time":    "2025-06-15T14:29:58Z", // RFC3339 string that needs timestamp coercion
		"end_time":      int64(1750000200000),   // epoch millis, already correct type
		"type":          300201,
		"is_mfa":        "true", // string that needs boolean coercion
		"is_remote":     1,      // int that needs boolean coercion
		"is_cleartext":  false,
		"auth_protocol": "SAML",
		"logon_type":    "Network",
		"status":        "Success",
		"status_id":     1,
		"message":       "User admin@example.com authenticated via SAML SSO from 192.168.1.100",
		"user": map[string]any{
			"type_id": 1,
			"name":    "admin@example.com",
		},
		"src_ip":   "192.168.1.100",
		"src_port": "443", // string that needs integer coercion
		"dst_ip":   "10.0.0.50",
		"dst_port": 8443,
		"dst_svc":  "auth-service",
		"product": map[string]any{
			"vendor_name": "observIQ",
			"name":        "bindplane-gateway",
		},
	}
}

var authenticationFieldMappings = []FieldMapping{
	{From: "body.activity", To: "activity_id"},
	{From: "body.category", To: "category_uid"},
	{From: "body.severity", To: "severity_id"},
	{From: "body.time", To: "time"},
	{From: "body.start_time", To: "start_time"},
	{From: "body.end_time", To: "end_time"},
	{From: "body.type", To: "type_uid"},
	{From: "body.is_mfa", To: "is_mfa"},
	{From: "body.is_remote", To: "is_remote"},
	{From: "body.is_cleartext", To: "is_cleartext"},
	{From: "body.auth_protocol", To: "auth_protocol"},
	{From: "body.logon_type", To: "logon_type"},
	{From: "body.status", To: "status"},
	{From: "body.status_id", To: "status_id"},
	{From: "body.message", To: "message"},
	{From: "body.user", To: "user"},
	{From: "body.src_ip", To: "src_endpoint.ip"},
	{From: "body.src_port", To: "src_endpoint.port"},
	{From: "body.dst_ip", To: "dst_endpoint.ip"},
	{From: "body.dst_port", To: "dst_endpoint.port"},
	{From: "body.dst_svc", To: "dst_endpoint.svc_name"},
	{From: "body.product", To: "metadata.product"},
}

func makeBenchmarkLogs(n int, bodyFunc func() map[string]any) plog.Logs {
	ld := plog.NewLogs()
	records := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
	for range n {
		record := records.AppendEmpty()
		record.Attributes().PutStr("event.type", "authentication")
		_ = record.Body().SetEmptyMap().FromRaw(bodyFunc())
	}
	return ld
}

func BenchmarkProcessLogs(b *testing.B) {
	cfg := &Config{
		OCSFVersion: OCSFVersion1_7_0,
		EventMappings: []EventMapping{
			{
				Filter:        `attributes["event.type"] == "authentication"`,
				ClassID:       3002,
				FieldMappings: authenticationFieldMappings,
			},
		},
	}

	b.Run("ValidationEnabled", func(b *testing.B) {
		cfg := *cfg
		boolTrue := true
		cfg.RuntimeValidation = &boolTrue
		processor, err := newOCSFStandardizationProcessor(zap.NewNop(), &cfg)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			logs := makeBenchmarkLogs(100, authenticationInputBody)
			b.StartTimer()
			_, _ = processor.processLogs(context.Background(), logs)
		}
	})

	b.Run("ValidationDisabled", func(b *testing.B) {
		cfg := *cfg
		boolFalse := false
		cfg.RuntimeValidation = &boolFalse
		processor, err := newOCSFStandardizationProcessor(zap.NewNop(), &cfg)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			logs := makeBenchmarkLogs(100, authenticationInputBody)
			b.StartTimer()
			_, _ = processor.processLogs(context.Background(), logs)
		}
	})
}
