package filter

import (
	"fmt"
	"testing"
)

// benchmarkLookupSetupFilter fills f with approximately half of n keys and returns
// one absent and one present key for fair comparison across filter types.
func benchmarkLookupSetupFilter(b *testing.B, n int, f Filter) (absent []byte, present []byte) {
	b.Helper()
	fill := n / 2
	if fill < 1 {
		fill = 1
	}
	for i := 0; i < fill; i++ {
		s := fmt.Sprintf("key-%d", i)
		f.AddString(s)
	}
	return []byte("absent-key-not-in-set"), []byte("key-0")
}

func BenchmarkComparison_Memory(b *testing.B) {
	sizes := []int{1_000, 10_000, 100_000}
	for _, n := range sizes {
		n := n
		b.Run(fmt.Sprintf("Bloom_N=%d", n), func(b *testing.B) {
			opts := BloomOptions{EstimatedCount: uint(n), FalsePositiveRate: 0.01}
			sample := NewBloomFilterFromOptions(opts)
			b.ReportMetric(float64(sample.Cap()), "bits")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = NewBloomFilterFromOptions(opts)
			}
		})
		b.Run(fmt.Sprintf("Cuckoo_N=%d", n), func(b *testing.B) {
			opts := CuckooOptions{Capacity: uint(n)}
			sample := NewCuckooFilterFromOptions(opts)
			b.ReportMetric(float64(sample.Cap()), "capacity")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = NewCuckooFilterFromOptions(opts)
			}
		})
		b.Run(fmt.Sprintf("ScalableCuckoo_N=%d", n), func(b *testing.B) {
			opts := ScalableCuckooOptions{}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = NewScalableCuckooFilterFromOptions(opts)
			}
		})
	}
}

func BenchmarkComparison_MayContain_absent(b *testing.B) {
	n := 10_000
	b.Run("Bloom", func(b *testing.B) {
		f := NewBloomFilterFromOptions(BloomOptions{EstimatedCount: uint(n), FalsePositiveRate: 0.01})
		absent, _ := benchmarkLookupSetupFilter(b, n, f)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContain(absent)
		}
	})
	b.Run("Cuckoo", func(b *testing.B) {
		f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: uint(n)})
		absent, _ := benchmarkLookupSetupFilter(b, n, f)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContain(absent)
		}
	})
	b.Run("ScalableCuckoo", func(b *testing.B) {
		f := NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
		absent, _ := benchmarkLookupSetupFilter(b, n, f)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContain(absent)
		}
	})
}

func BenchmarkComparison_MayContain_present(b *testing.B) {
	n := 10_000
	b.Run("Bloom", func(b *testing.B) {
		f := NewBloomFilterFromOptions(BloomOptions{EstimatedCount: uint(n), FalsePositiveRate: 0.01})
		_, present := benchmarkLookupSetupFilter(b, n, f)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContain(present)
		}
	})
	b.Run("Cuckoo", func(b *testing.B) {
		f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: uint(n)})
		_, present := benchmarkLookupSetupFilter(b, n, f)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContain(present)
		}
	})
	b.Run("ScalableCuckoo", func(b *testing.B) {
		f := NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
		_, present := benchmarkLookupSetupFilter(b, n, f)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContain(present)
		}
	})
}

func BenchmarkComparison_MayContainString_absent(b *testing.B) {
	n := 10_000
	b.Run("Bloom", func(b *testing.B) {
		f := NewBloomFilterFromOptions(BloomOptions{EstimatedCount: uint(n), FalsePositiveRate: 0.01})
		absent, _ := benchmarkLookupSetupFilter(b, n, f)
		absentStr := string(absent)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContainString(absentStr)
		}
	})
	b.Run("Cuckoo", func(b *testing.B) {
		f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: uint(n)})
		absent, _ := benchmarkLookupSetupFilter(b, n, f)
		absentStr := string(absent)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContainString(absentStr)
		}
	})
	b.Run("ScalableCuckoo", func(b *testing.B) {
		f := NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
		absent, _ := benchmarkLookupSetupFilter(b, n, f)
		absentStr := string(absent)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContainString(absentStr)
		}
	})
}

func BenchmarkComparison_MayContainString_present(b *testing.B) {
	n := 10_000
	b.Run("Bloom", func(b *testing.B) {
		f := NewBloomFilterFromOptions(BloomOptions{EstimatedCount: uint(n), FalsePositiveRate: 0.01})
		_, present := benchmarkLookupSetupFilter(b, n, f)
		presentStr := string(present)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContainString(presentStr)
		}
	})
	b.Run("Cuckoo", func(b *testing.B) {
		f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: uint(n)})
		_, present := benchmarkLookupSetupFilter(b, n, f)
		presentStr := string(present)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContainString(presentStr)
		}
	})
	b.Run("ScalableCuckoo", func(b *testing.B) {
		f := NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
		_, present := benchmarkLookupSetupFilter(b, n, f)
		presentStr := string(present)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = f.MayContainString(presentStr)
		}
	})
}
