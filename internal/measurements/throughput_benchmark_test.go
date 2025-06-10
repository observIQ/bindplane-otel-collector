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

package measurements

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/otel/metric/noop"
)

// generateTestLogs creates a plog.Logs with the specified number of log records
func generateTestLogs(numRecords int, withOriginal bool) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()

	for i := 0; i < numRecords; i++ {
		logRecord := sl.LogRecords().AppendEmpty()
		logRecord.Body().SetStr("test log message")

		if withOriginal {
			logRecord.Attributes().PutStr("log.record.original", "original log content")
		}
	}

	return logs
}

// BenchmarkAddLogsMeasureRawBytes benchmarks the AddLogs function with different log volumes and measureRawBytes set to true
func BenchmarkAddLogsMeasureRawBytes(b *testing.B) {
	benchmarks := []struct {
		name         string
		numRecords   int
		withOriginal bool
	}{
		{"10Logs_WithOriginal", 10, true},
		{"100Logs_WithOriginal", 100, true},
		{"1000Logs_WithOriginal", 1000, true},
		{"10000Logs_WithOriginal", 10000, true},
		{"10Logs_WithoutOriginal", 10, false},
		{"100Logs_WithoutOriginal", 100, false},
		{"1000Logs_WithoutOriginal", 1000, false},
		{"10000Logs_WithoutOriginal", 10000, false},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			mp := noop.NewMeterProvider()
			tm, err := NewThroughputMeasurements(mp, "test", nil)
			if err != nil {
				b.Fatal(err)
			}

			logs := generateTestLogs(bm.numRecords, bm.withOriginal)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tm.AddLogs(ctx, logs, true)
			}
		})
	}
}

// BenchmarkAddLogsMeasureRawBytesFalse benchmarks the AddLogs function with different log volumes and measureRawBytes set to false
func BenchmarkAddLogsMeasureRawBytesFalse(b *testing.B) {
	benchmarks := []struct {
		name         string
		numRecords   int
		withOriginal bool
	}{
		{"10Logs_WithOriginal", 10, true},
		{"100Logs_WithOriginal", 100, true},
		{"1000Logs_WithOriginal", 1000, true},
		{"10000Logs_WithOriginal", 10000, true},
		{"10Logs_WithoutOriginal", 10, false},
		{"100Logs_WithoutOriginal", 100, false},
		{"1000Logs_WithoutOriginal", 1000, false},
		{"10000Logs_WithoutOriginal", 10000, false},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			mp := noop.NewMeterProvider()
			tm, err := NewThroughputMeasurements(mp, "test", nil)
			if err != nil {
				b.Fatal(err)
			}

			logs := generateTestLogs(bm.numRecords, bm.withOriginal)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tm.AddLogs(ctx, logs, false)
			}
		})
	}
}

// BenchmarkWithoutAddLogs benchmarks a version without using AddLogs
func BenchmarkWithoutAddLogs(b *testing.B) {
	benchmarks := []struct {
		name         string
		numRecords   int
		withOriginal bool
	}{
		{"10Logs_WithOriginal", 10, true},
		{"100Logs_WithOriginal", 100, true},
		{"1000Logs_WithOriginal", 1000, true},
		{"10000Logs_WithOriginal", 10000, true},
		{"10Logs_WithoutOriginal", 10, false},
		{"100Logs_WithoutOriginal", 100, false},
		{"1000Logs_WithoutOriginal", 1000, false},
		{"10000Logs_WithoutOriginal", 10000, false},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			mp := noop.NewMeterProvider()
			tm, err := NewThroughputMeasurements(mp, "test", nil)
			if err != nil {
				b.Fatal(err)
			}

			logs := generateTestLogs(bm.numRecords, bm.withOriginal)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Not using AddLogs to compare performance
				tm.collectionSequenceNumber.Add(1)
				sizer := plog.ProtoMarshaler{}
				totalSize := int64(sizer.LogsSize(logs))
				tm.logSize.Add(ctx, totalSize)
				tm.logCount.Add(ctx, int64(logs.LogRecordCount()))
			}
		})
	}
}
