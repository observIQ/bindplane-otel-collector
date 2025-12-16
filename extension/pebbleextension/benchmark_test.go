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
package pebbleextension

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/pebbleextension/internal/client"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

// setupBenchmarkClient creates a temporary pebble client for benchmarking
func setupBenchmarkClient(b testing.TB, sync bool) (client.Client, func()) {
	b.Helper()

	dir := b.TempDir()

	c, err := client.NewClient(dir, &client.Options{Sync: sync})
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
		name      string
		sync      bool
		valueSize int
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
			c, cleanup := setupBenchmarkClient(b, bm.sync)
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
		name      string
		sync      bool
		valueSize int
		numKeys   int
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
			c, cleanup := setupBenchmarkClient(b, bm.sync)
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
		name      string
		sync      bool
		valueSize int
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
			c, cleanup := setupBenchmarkClient(b, bm.sync)
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
		name      string
		batchSize int
		valueSize int
		sync      bool
		opType    storage.OpType
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
			c, cleanup := setupBenchmarkClient(b, bm.sync)
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
		name      string
		sync      bool
		numKeys   int
		valueSize int
		readRatio float64
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
			c, cleanup := setupBenchmarkClient(b, bm.sync)
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
