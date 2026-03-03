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

func makeBenchmarkLogs(n int) plog.Logs {
	ld := plog.NewLogs()
	records := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
	for range n {
		record := records.AppendEmpty()
		_ = record.Body().SetEmptyMap().FromRaw(accountChangeInputBody())
	}
	return ld
}

func BenchmarkProcessLogs(b *testing.B) {
	cfg := &Config{
		OCSFVersion: OCSFVersion1_7_0,
		EventMappings: []EventMapping{
			{
				ClassID:       3001,
				FieldMappings: accountChangeFieldMappings,
			},
		},
	}

	boolTrue := true
	boolFalse := false

	b.Run("ValidationEnabled", func(b *testing.B) {
		cfg := *cfg
		cfg.RuntimeValidation = &boolTrue
		processor, err := newOCSFStandardizationProcessor(zap.NewNop(), &cfg)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			logs := makeBenchmarkLogs(100)
			b.StartTimer()
			_, _ = processor.processLogs(context.Background(), logs)
		}
	})

	b.Run("ValidationDisabled", func(b *testing.B) {
		cfg := *cfg
		cfg.RuntimeValidation = &boolFalse
		processor, err := newOCSFStandardizationProcessor(zap.NewNop(), &cfg)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			logs := makeBenchmarkLogs(100)
			b.StartTimer()
			_, _ = processor.processLogs(context.Background(), logs)
		}
	})
}
