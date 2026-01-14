// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package badgerextension

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/client"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

// setupBenchmarkClient creates a temporary badger client for benchmarking
func setupBenchmarkClient(b testing.TB, syncWrites bool) (client.Client, func()) {
	b.Helper()

	dir := b.TempDir()

	c, err := client.NewClient(dir, &client.Options{SyncWrites: syncWrites}, zap.NewNop())
	if err != nil {
		b.Fatalf("failed to create client: %v", err)
	}

	cleanup := func() {
		if err := c.Close(b.Context()); err != nil {
			b.Errorf("failed to close client: %v", err)
		}
	}

	return c, cleanup
}

// generateRandomBytes creates random byte data of specified size
func generateRandomBytes(size int) []byte {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return data
}

// BenchmarkSet benchmarks the Set operation with different value sizes
func BenchmarkSet(b *testing.B) {
	benchmarks := []struct {
		name       string
		syncWrites bool
		valueSize  int
	}{
		{"100B", true, 100},
		{"1KB", true, 1024},
		{"10KB", true, 10 * 1024},
		{"100KB", true, 100 * 1024},
		{"1MB", true, 1024 * 1024},
		{"100B_NoSync", false, 100},
		{"1KB_NoSync", false, 1024},
		{"10KB_NoSync", false, 10 * 1024},
		{"100KB_NoSync", false, 100 * 1024},
		{"1MB_NoSync", false, 1024 * 1024},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			value := generateRandomBytes(bm.valueSize)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Set(ctx, key, value); err != nil {
					b.Fatalf("failed to set: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

// BenchmarkGet benchmarks the Get operation with different value sizes
func BenchmarkGet(b *testing.B) {
	benchmarks := []struct {
		name       string
		syncWrites bool
		valueSize  int
		numKeys    int
	}{
		{"100B_1000Keys", true, 100, 1000},
		{"1KB_1000Keys", true, 1024, 1000},
		{"10KB_1000Keys", true, 10 * 1024, 1000},
		{"100KB_1000Keys", true, 100 * 1024, 1000},
		{"1MB_1000Keys", true, 1024 * 1024, 1000},
		{"100B_1000Keys_NoSync", false, 100, 1000},
		{"1KB_1000Keys_NoSync", false, 1024, 1000},
		{"10KB_1000Keys_NoSync", false, 10 * 1024, 1000},
		{"100KB_1000Keys_NoSync", false, 100 * 1024, 1000},
		{"1MB_1000Keys_NoSync", false, 1024 * 1024, 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// Populate with test data
			for i := 0; i < bm.numKeys; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Set(ctx, key, value); err != nil {
					b.Fatalf("failed to set: %v", err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key_%d", i%bm.numKeys)
				if _, err := c.Get(ctx, key); err != nil {
					b.Fatalf("failed to get: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

// BenchmarkDelete benchmarks the Delete operation
func BenchmarkDelete(b *testing.B) {
	benchmarks := []struct {
		name       string
		syncWrites bool
		valueSize  int
	}{
		{"100B", true, 100},
		{"1KB", true, 1024},
		{"10KB", true, 10 * 1024},
		{"100KB", true, 100 * 1024},
		{"1MB", true, 1024 * 1024},
		{"100B_NoSync", false, 100},
		{"1KB_NoSync", false, 1024},
		{"10KB_NoSync", false, 10 * 1024},
		{"100KB_NoSync", false, 100 * 1024},
		{"1MB_NoSync", false, 1024 * 1024},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// Populate with test data
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Set(ctx, key, value); err != nil {
					b.Fatalf("failed to set: %v", err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Delete(ctx, key); err != nil {
					b.Fatalf("failed to delete: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

// BenchmarkBatch benchmarks batch operations with varying sizes
func BenchmarkBatch(b *testing.B) {
	benchmarks := []struct {
		name       string
		batchSize  int
		valueSize  int
		syncWrites bool
		opType     storage.OpType
	}{
		{"Set_10Ops_1KB", 10, 1024, true, storage.Set},
		{"Set_100Ops_1KB", 100, 1024, true, storage.Set},
		{"Set_1000Ops_1KB", 1000, 1024, true, storage.Set},
		{"Delete_10Ops", 10, 1024, true, storage.Delete},
		{"Delete_100Ops", 100, 1024, true, storage.Delete},
		{"Set_10Ops_1KB_NoSync", 10, 1024, false, storage.Set},
		{"Set_100Ops_1KB_NoSync", 100, 1024, false, storage.Set},
		{"Set_1000Ops_1KB_NoSync", 1000, 1024, false, storage.Set},
		{"Delete_10Ops_NoSync", 10, 1024, false, storage.Delete},
		{"Delete_100Ops_NoSync", 100, 1024, false, storage.Delete},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// For delete operations, pre-populate the data
			if bm.opType == storage.Delete {
				for i := 0; i < bm.batchSize*b.N; i++ {
					key := fmt.Sprintf("key_%d", i)
					if err := c.Set(ctx, key, value); err != nil {
						b.Fatalf("failed to set: %v", err)
					}
				}
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ops := make([]*storage.Operation, bm.batchSize)
				for j := 0; j < bm.batchSize; j++ {
					key := fmt.Sprintf("key_%d", i*bm.batchSize+j)
					ops[j] = &storage.Operation{
						Type:  bm.opType,
						Key:   key,
						Value: value,
					}
				}
				if err := c.Batch(ctx, ops...); err != nil {
					b.Fatalf("failed to batch: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

// BenchmarkMixedWorkload benchmarks a mixed read/write workload
func BenchmarkMixedWorkload(b *testing.B) {
	benchmarks := []struct {
		name       string
		syncWrites bool
		numKeys    int
		valueSize  int
		readRatio  float64
	}{
		{"50PercentReads_1KB_1000Keys", true, 1000, 1024, 0.5},
		{"80PercentReads_1KB_1000Keys", true, 1000, 1024, 0.8},
		{"95PercentReads_1KB_1000Keys", true, 1000, 1024, 0.95},
		{"50PercentReads_10KB_1000Keys", true, 1000, 10 * 1024, 0.5},
		{"80PercentReads_10KB_1000Keys", true, 1000, 10 * 1024, 0.8},
		{"95PercentReads_10KB_1000Keys", true, 1000, 10 * 1024, 0.95},
		{"50PercentReads_1KB_1000Keys_NoSync", false, 1000, 1024, 0.5},
		{"80PercentReads_1KB_1000Keys_NoSync", false, 1000, 1024, 0.8},
		{"95PercentReads_1KB_1000Keys_NoSync", false, 1000, 1024, 0.95},
		{"50PercentReads_10KB_1000Keys_NoSync", false, 1000, 10 * 1024, 0.5},
		{"80PercentReads_10KB_1000Keys_NoSync", false, 1000, 10 * 1024, 0.8},
		{"95PercentReads_10KB_1000Keys_NoSync", false, 1000, 10 * 1024, 0.95},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// Populate with initial data
			for i := 0; i < bm.numKeys; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Set(ctx, key, value); err != nil {
					b.Fatalf("failed to set: %v", err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				keyIdx := i % bm.numKeys
				key := fmt.Sprintf("key_%d", keyIdx)

				// Simulate read vs write based on ratio
				if float64(i%100)/100.0 < bm.readRatio {
					if _, err := c.Get(ctx, key); err != nil {
						b.Fatalf("failed to get: %v", err)
					}
				} else {
					if err := c.Set(ctx, key, value); err != nil {
						b.Fatalf("failed to set: %v", err)
					}
				}
			}
			b.StopTimer()
		})
	}
}

// BenchmarkValueLogGC benchmarks the garbage collection operation
func BenchmarkValueLogGC(b *testing.B) {
	benchmarks := []struct {
		name         string
		syncWrites   bool
		numKeys      int
		valueSize    int
		discardRatio float64
	}{
		{"1000Keys_1KB_0.5Ratio", true, 1000, 1024, 0.5},
		{"1000Keys_10KB_0.5Ratio", true, 1000, 10 * 1024, 0.5},
		{"10000Keys_1KB_0.5Ratio", true, 10000, 1024, 0.5},
		{"1000Keys_1KB_0.5Ratio_NoSync", false, 1000, 1024, 0.5},
		{"1000Keys_10KB_0.5Ratio_NoSync", false, 1000, 10 * 1024, 0.5},
		{"10000Keys_1KB_0.5Ratio_NoSync", false, 10000, 1024, 0.5},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// Populate with data
			for i := 0; i < bm.numKeys; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Set(ctx, key, value); err != nil {
					b.Fatalf("failed to set: %v", err)
				}
			}

			// Delete half the keys to create garbage
			for i := 0; i < bm.numKeys/2; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Delete(ctx, key); err != nil {
					b.Fatalf("failed to delete: %v", err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := c.RunValueLogGC(bm.discardRatio); err != nil {
					// GC may return error if no rewrite is needed, which is ok
					b.Logf("GC returned: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

// setupBenchmarkClientWithCompaction creates a badger client with custom compaction settings
func setupBenchmarkClientWithCompaction(b testing.TB, syncWrites bool, numCompactors int, numLevelZeroTables int, numLevelZeroTablesStall int) (client.Client, func()) {
	b.Helper()

	dir := b.TempDir()

	opts := &client.Options{
		SyncWrites:              syncWrites,
		NumCompactors:           numCompactors,
		NumLevelZeroTables:      numLevelZeroTables,
		NumLevelZeroTablesStall: numLevelZeroTablesStall,
	}
	c, err := client.NewClient(dir, opts, zap.NewNop())
	if err != nil {
		b.Fatalf("failed to create client: %v", err)
	}

	cleanup := func() {
		if err := c.Close(b.Context()); err != nil {
			b.Errorf("failed to close client: %v", err)
		}
	}

	return c, cleanup
}

// BenchmarkCompactionSettings benchmarks mixed workload performance with different compaction configurations
// This demonstrates the throughput vs disk cleanup trade-off
func BenchmarkCompactionSettings(b *testing.B) {
	benchmarks := []struct {
		name                    string
		numCompactors           int
		numLevelZeroTables      int
		numLevelZeroTablesStall int
		numKeys                 int
		valueSize               int
		deleteRatio             float64 // fraction of operations that are deletes
		description             string
	}{
		{
			"Default_8_Compactors_L0_3",
			8, 3, 8, 50000, 1024, 0.3,
			"Default: 8 compactors, L0 triggers at 3",
		},
		{
			"Aggressive_8_Compactors_L0_2",
			8, 2, 8, 50000, 1024, 0.3,
			"More aggressive: Compact sooner (L0=2), same parallelism",
		},
		{
			"Conservative_4_Compactors_L0_3",
			4, 3, 8, 50000, 1024, 0.3,
			"Conservative: Fewer workers (4 compactors)",
		},
		{
			"HighThroughput_16_Compactors_L0_3",
			16, 3, 16, 50000, 1024, 0.3,
			"High throughput: More workers (16 compactors)",
		},
		{
			"Balanced_12_Compactors_L0_2",
			12, 2, 12, 50000, 1024, 0.3,
			"Balanced: 12 compactors, earlier compaction (L0=2)",
		},
		{
			"HeavyDeletes_12_Compactors_L0_2",
			12, 2, 12, 50000, 1024, 0.5,
			"Queue workload: 12 compactors, L0=2, 50% deletes",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClientWithCompaction(b, false, bm.numCompactors, bm.numLevelZeroTables, bm.numLevelZeroTablesStall)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// Populate with initial data
			for i := 0; i < bm.numKeys; i++ {
				key := fmt.Sprintf("key_%d", i)
				if err := c.Set(ctx, key, value); err != nil {
					b.Fatalf("failed to set: %v", err)
				}
			}

			// Get initial disk usage before benchmark
			initialDiskUsage, err := c.GetDiskUsage()
			if err != nil {
				b.Fatalf("failed to get initial disk usage: %v", err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			// Mixed read/write/delete workload
			deleteCount := 0
			for i := 0; i < b.N; i++ {
				keyIdx := i % bm.numKeys
				key := fmt.Sprintf("key_%d", keyIdx)

				// Simulate delete vs set based on ratio
				if float64(i%100)/100.0 < bm.deleteRatio {
					if err := c.Delete(ctx, key); err != nil {
						b.Fatalf("failed to delete: %v", err)
					}
					deleteCount++
				} else {
					if err := c.Set(ctx, key, value); err != nil {
						b.Fatalf("failed to set: %v", err)
					}
				}
			}

			b.StopTimer()

			// Get final disk usage after benchmark
			finalDiskUsage, err := c.GetDiskUsage()
			if err != nil {
				b.Fatalf("failed to get final disk usage: %v", err)
			}

			// Calculate disk growth
			lsmGrowth := finalDiskUsage.LSMUsage - initialDiskUsage.LSMUsage
			valueLogGrowth := finalDiskUsage.ValueLogUsage - initialDiskUsage.ValueLogUsage
			totalGrowth := lsmGrowth + valueLogGrowth

			b.ReportMetric(float64(deleteCount), "deletes")
			b.ReportMetric(float64(lsmGrowth), "lsm_growth_bytes")
			b.ReportMetric(float64(valueLogGrowth), "valuelog_growth_bytes")
			b.ReportMetric(float64(totalGrowth), "total_growth_bytes")
			b.ReportMetric(float64(finalDiskUsage.LSMUsage), "final_lsm_bytes")
			b.ReportMetric(float64(finalDiskUsage.ValueLogUsage), "final_valuelog_bytes")
		})
	}
}
