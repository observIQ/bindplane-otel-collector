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

package badgerextension

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// BenchmarkConcurrentWrites simulates multiple producers writing to storage
// This mimics the sending_queue with num_consumers > 1
func BenchmarkConcurrentWrites(b *testing.B) {
	benchmarks := []struct {
		name       string
		syncWrites bool
		numWriters int
		valueSize  int
	}{
		{"1Writer_1KB", true, 1, 1024},
		{"5Writers_1KB", true, 5, 1024},
		{"10Writers_1KB", true, 10, 1024},
		{"20Writers_1KB", true, 20, 1024},
		{"10Writers_10KB", true, 10, 10 * 1024},
		{"1Writer_1KB_NoSync", false, 1, 1024},
		{"5Writers_1KB_NoSync", false, 5, 1024},
		{"10Writers_1KB_NoSync", false, 10, 1024},
		{"20Writers_1KB_NoSync", false, 20, 1024},
		{"10Writers_10KB_NoSync", false, 10, 10 * 1024},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			b.ResetTimer()
			b.ReportAllocs()

			// Divide work among writers
			opsPerWriter := b.N / bm.numWriters
			var wg sync.WaitGroup

			for w := 0; w < bm.numWriters; w++ {
				wg.Add(1)
				writerID := w
				go func() {
					defer wg.Done()
					for i := 0; i < opsPerWriter; i++ {
						key := fmt.Sprintf("writer_%d_key_%d", writerID, i)
						if err := c.Set(ctx, key, value); err != nil {
							b.Errorf("failed to set: %v", err)
							return
						}
					}
				}()
			}

			wg.Wait()
			b.StopTimer()
		})
	}
}

// BenchmarkQueuePattern simulates a persistent queue pattern:
// - Multiple concurrent writers (producers)
// - Sequential batch reader (consumer)
func BenchmarkQueuePattern(b *testing.B) {
	benchmarks := []struct {
		name         string
		syncWrites   bool
		numProducers int
		batchSize    int
		valueSize    int
	}{
		{"5Producers_Batch50_1KB", true, 5, 50, 1024},
		{"10Producers_Batch50_1KB", true, 10, 50, 1024},
		{"10Producers_Batch100_1KB", true, 10, 100, 1024},
		{"10Producers_Batch50_10KB", true, 10, 50, 10 * 1024},
		{"5Producers_Batch50_1KB_NoSync", false, 5, 50, 1024},
		{"10Producers_Batch50_1KB_NoSync", false, 10, 50, 1024},
		{"10Producers_Batch100_1KB_NoSync", false, 10, 100, 1024},
		{"10Producers_Batch50_10KB_NoSync", false, 10, 50, 10 * 1024},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			b.ResetTimer()
			b.ReportAllocs()

			// Channel to simulate queue
			keysChan := make(chan string, bm.batchSize*10)

			// Start producers
			var producerWg sync.WaitGroup
			opsPerProducer := b.N / bm.numProducers

			for p := 0; p < bm.numProducers; p++ {
				producerWg.Add(1)
				producerID := p
				go func() {
					defer producerWg.Done()
					for i := 0; i < opsPerProducer; i++ {
						key := fmt.Sprintf("queue_%d_%d", producerID, i)
						if err := c.Set(ctx, key, value); err != nil {
							b.Errorf("failed to set: %v", err)
							return
						}
						keysChan <- key
					}
				}()
			}

			// Start consumer (reading in batches)
			var consumerWg sync.WaitGroup
			consumerWg.Add(1)
			go func() {
				defer consumerWg.Done()
				batch := make([]string, 0, bm.batchSize)
				totalRead := 0
				expectedTotal := b.N / bm.numProducers * bm.numProducers

				for totalRead < expectedTotal {
					select {
					case key := <-keysChan:
						batch = append(batch, key)
						if len(batch) >= bm.batchSize {
							// Batch read
							for _, k := range batch {
								if k == "" {
									continue
								}
								if _, err := c.Get(ctx, k); err != nil {
									b.Errorf("failed to get: %v", err)
									return
								}
							}
							totalRead += len(batch)
							batch = batch[:0]
						}
					default:
						if len(batch) > 0 {
							// Flush remaining
							for _, k := range batch {
								if _, err := c.Get(ctx, k); err != nil {
									b.Errorf("failed to get: %v", err)
									return
								}
							}
							totalRead += len(batch)
							batch = batch[:0]
						}
					}
				}
			}()

			producerWg.Wait()
			close(keysChan)
			consumerWg.Wait()

			b.StopTimer()
		})
	}
}

// BenchmarkThroughput measures raw throughput with concurrent writers
// This is closest to what you're measuring with your collector configs
func BenchmarkThroughput(b *testing.B) {
	benchmarks := []struct {
		name       string
		syncWrites bool
		numWriters int
		valueSize  int
		totalOps   int
	}{
		{"10Writers_1KB_100kOps", true, 10, 1024, 100000},
		{"10Writers_1KB_500kOps", true, 10, 1024, 500000},
		{"10Writers_10KB_100kOps", true, 10, 10 * 1024, 100000},
		{"10Writers_1KB_100kOps_NoSync", false, 10, 1024, 100000},
		{"10Writers_1KB_500kOps_NoSync", false, 10, 1024, 500000},
		{"10Writers_10KB_100kOps_NoSync", false, 10, 10 * 1024, 100000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// Run once (b.N is not used for throughput tests)
			b.ResetTimer()
			b.ReportAllocs()

			opsPerWriter := bm.totalOps / bm.numWriters
			var wg sync.WaitGroup

			for w := 0; w < bm.numWriters; w++ {
				wg.Add(1)
				writerID := w
				go func() {
					defer wg.Done()
					for i := 0; i < opsPerWriter; i++ {
						key := fmt.Sprintf("throughput_%d_%d", writerID, i)
						if err := c.Set(ctx, key, value); err != nil {
							b.Errorf("failed to set: %v", err)
							return
						}
					}
				}()
			}

			wg.Wait()
			b.StopTimer()

			// Report throughput
			opsPerSec := float64(bm.totalOps) / b.Elapsed().Seconds()
			b.ReportMetric(opsPerSec, "ops/sec")
			mbPerSec := float64(bm.totalOps*bm.valueSize) / (1024 * 1024) / b.Elapsed().Seconds()
			b.ReportMetric(mbPerSec, "MB/sec")
		})
	}
}

// BenchmarkConcurrentReadWrite simulates mixed concurrent read/write workload
func BenchmarkConcurrentReadWrite(b *testing.B) {
	benchmarks := []struct {
		name        string
		syncWrites  bool
		numWriters  int
		numReaders  int
		valueSize   int
		preloadKeys int
	}{
		{"5W_5R_1KB_1000Keys", true, 5, 5, 1024, 1000},
		{"5W_5R_1KB_1000Keys_NoSync", false, 5, 5, 1024, 1000},
		{"10W_10R_1KB_1000Keys", true, 10, 10, 1024, 1000},
		{"10W_10R_1KB_1000Keys_NoSync", false, 10, 10, 1024, 1000},
		{"5W_15R_1KB_1000Keys", true, 5, 15, 1024, 1000},
		{"5W_15R_1KB_1000Keys_NoSync", false, 5, 15, 1024, 1000},
		{"10W_10R_10KB_1000Keys", true, 10, 10, 10 * 1024, 1000},
		{"10W_10R_10KB_1000Keys_NoSync", false, 10, 10, 10 * 1024, 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			c, cleanup := setupBenchmarkClient(b, bm.syncWrites)
			defer cleanup()

			ctx := context.Background()
			value := generateRandomBytes(bm.valueSize)

			// Preload some keys
			for i := 0; i < bm.preloadKeys; i++ {
				key := fmt.Sprintf("preload_%d", i)
				if err := c.Set(ctx, key, value); err != nil {
					b.Fatalf("failed to preload: %v", err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			opsPerGoroutine := b.N / (bm.numWriters + bm.numReaders)
			var wg sync.WaitGroup

			// Start writers
			for w := 0; w < bm.numWriters; w++ {
				wg.Add(1)
				writerID := w
				go func() {
					defer wg.Done()
					for i := 0; i < opsPerGoroutine; i++ {
						key := fmt.Sprintf("concurrent_%d_%d", writerID, i)
						if err := c.Set(ctx, key, value); err != nil {
							b.Errorf("failed to set: %v", err)
							return
						}
					}
				}()
			}

			// Start readers
			for r := 0; r < bm.numReaders; r++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; i < opsPerGoroutine; i++ {
						key := fmt.Sprintf("preload_%d", i%bm.preloadKeys)
						if _, err := c.Get(ctx, key); err != nil {
							b.Errorf("failed to get: %v", err)
							return
						}
					}
				}()
			}

			wg.Wait()
			b.StopTimer()
		})
	}
}
