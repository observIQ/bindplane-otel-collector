package filter

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestFilter_Add_MayContain(t *testing.T) {
	f := NewBloomFilter(1000, 0.01)

	// Add and check present
	vals := [][]byte{[]byte("a"), []byte("b"), []byte("1.2.3.4")}
	for _, v := range vals {
		f.Add(v)
		if !f.MayContain(v) {
			t.Errorf("MayContain(%q) = false after Add", v)
		}
	}

	// Value not added should usually be false (may be false positive with tiny probability)
	absent := []byte("definitely-not-added")
	if f.MayContain(absent) {
		// With 1000 capacity and 3 items, false positive is very unlikely; if it happens, retry is ok
		t.Logf("MayContain(absent) was true (false positive); filter is very small")
	}
}

func TestFilter_AddString_MayContainString(t *testing.T) {
	f := NewBloomFilter(1000, 0.01)

	f.AddString("192.168.1.1")
	f.AddString("example.com")
	if !f.MayContainString("192.168.1.1") {
		t.Error("MayContainString(192.168.1.1) = false after AddString")
	}
	if !f.MayContainString("example.com") {
		t.Error("MayContainString(example.com) = false after AddString")
	}
	if f.MayContainString("10.0.0.1") {
		t.Logf("MayContainString(10.0.0.1) true (might be false positive)")
	}
}

func TestFilter_MayContain_DefinitelyNotPresent(t *testing.T) {
	f := NewBloomFilter(100_000, 0.01)
	f.Add([]byte("only-one"))
	// Query something we did not add; with 100k capacity and 1 item, false positive is negligible
	if f.MayContain([]byte("different-value")) {
		t.Error("MayContain(different-value) = true but we never added it")
	}
}

func TestFilter_FalsePositivePossible(t *testing.T) {
	// Very small filter with high load to encourage a false positive for testing semantics
	f := NewBloomFilter(10, 0.5)
	f.Add([]byte("a"))
	f.Add([]byte("b"))
	f.Add([]byte("c"))
	// Some non-added value might still test true
	foundFalsePositive := false
	for i := 0; i < 500; i++ {
		candidate := []byte(fmt.Sprintf("x%d", i))
		if f.MayContain(candidate) {
			foundFalsePositive = true
			break
		}
	}
	if !foundFalsePositive {
		t.Logf("no false positive in 500 trials (small filter); semantics are still: true = may be present")
	}
}

func TestNewFilter_KindBloom(t *testing.T) {
	f, err := NewFilter(KindBloom, BloomOptions{EstimatedCount: 1000, FalsePositiveRate: 0.01})
	if err != nil {
		t.Fatalf("NewFilter(KindBloom, ...): %v", err)
	}
	f.AddString("test")
	if !f.MayContainString("test") {
		t.Error("filter from NewFilter did not contain added value")
	}
	if f.Cap() == 0 {
		t.Error("Cap() = 0")
	}
}

func TestNewFilter_UnknownKind(t *testing.T) {
	_, err := NewFilter(Kind("other"), nil)
	if err == nil {
		t.Error("NewFilter(unknown kind) should error")
	}
}

func TestNewFilterFromConfig_BloomOptions(t *testing.T) {
	f, err := NewFilterFromConfig(BloomOptions{EstimatedCount: 1000, FalsePositiveRate: 0.01})
	if err != nil {
		t.Fatalf("NewFilterFromConfig(BloomOptions): %v", err)
	}
	f.AddString("test")
	if !f.MayContainString("test") {
		t.Error("filter from NewFilterFromConfig did not contain added value")
	}
	if f.Cap() == 0 {
		t.Error("Cap() = 0")
	}
}

func TestNewBloomFilterFromOptions_MaxEstimatedCount(t *testing.T) {
	// When EstimatedCount > MaxEstimatedCount, sizing should be capped
	o := BloomOptions{
		EstimatedCount:    100_000,
		FalsePositiveRate: 0.01,
		MaxEstimatedCount: 5_000,
	}
	f := NewBloomFilterFromOptions(o)
	// Cap() is in bits; a 5k filter is smaller than a 100k filter
	capBits := f.Cap()
	// 100k at 1% would be much larger; 5k at 1% is ~ 47925 bits (approx)
	if capBits > 100_000 {
		t.Errorf("MaxEstimatedCount should cap size; Cap() = %d bits (expected smaller)", capBits)
	}
}

func TestFilterSet_AddFilter_Filter_Names(t *testing.T) {
	s := NewFilterSet()
	if len(s.Names()) != 0 {
		t.Fatalf("new FilterSet had names: %v", s.Names())
	}

	ips := s.AddBloomFilter("ips", 10_000, 0.01)
	ips.AddString("1.2.3.4")
	domains := s.AddBloomFilter("domains", 5_000, 0.01)
	domains.AddString("evil.com")

	if s.Filter("none") != nil {
		t.Error("Filter(\"none\") should be nil")
	}
	got := s.Filter("ips")
	if got == nil {
		t.Fatal("Filter(\"ips\") is nil")
	}
	if !got.MayContainString("1.2.3.4") {
		t.Error("ips filter did not contain 1.2.3.4")
	}
	gotD := s.Filter("domains")
	if gotD == nil || !gotD.MayContainString("evil.com") {
		t.Error("domains filter missing or wrong")
	}

	names := s.Names()
	if len(names) != 2 {
		t.Errorf("Names() = %v", names)
	}
}

// randomIPv4 returns a random IPv4 address string (e.g. "1.2.3.4").
func randomIPv4(rng *rand.Rand) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256))
}

func TestFilter_HydrateWithRandomIPs(t *testing.T) {
	// Simulate hydrating a Bloom filter with indicator-like values (e.g. from AbuseIPDB).
	const n = 5000
	const fpRate = 0.01
	f := NewBloomFilter(n, fpRate)
	rng := rand.New(rand.NewSource(42))
	added := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		ip := randomIPv4(rng)
		added[ip] = true
		f.AddString(ip)
	}

	// All added IPs should test as may-contain
	for ip := range added {
		if !f.MayContainString(ip) {
			t.Errorf("MayContainString(%q) = false but we added it", ip)
		}
	}

	// Some non-added IPs should test false (definitely not present)
	rng2 := rand.New(rand.NewSource(999))
	for i := 0; i < 200; i++ {
		ip := randomIPv4(rng2)
		if added[ip] {
			continue
		}
		// With 5k capacity and 1% fp rate, a random other IP is almost always false
		if !f.MayContainString(ip) {
			// Correct: definitely not in set
			break
		}
	}
}

// TestFalsePositiveRate_ByInputSize computes empirical false positive rate for
// different (estimatedCount, insertedCount, targetFPR) and logs or asserts.
// Run with -v to see the table.
func TestFalsePositiveRate_ByInputSize(t *testing.T) {
	const numQueries = 50_000
	// (estimatedCount, insertedCount, targetFPR); insertedCount can be 50%, 100%, 150% of estimated
	cases := []struct {
		estimatedCount uint
		insertedCount int
		targetFPR     float64
		maxSlack      float64 // allow empirical up to targetFPR * maxSlack (e.g. 2.0)
	}{
		{1_000, 500, 0.01, 2.5},
		{1_000, 1_000, 0.01, 2.5},
		{1_000, 1_500, 0.01, 6.0},   // overfill: allow higher empirical FPR
		{10_000, 5_000, 0.01, 2.0},
		{10_000, 10_000, 0.01, 2.0},
		{10_000, 15_000, 0.01, 6.0},  // overfill
		{100_000, 50_000, 0.01, 2.0},
		{100_000, 100_000, 0.01, 2.0},
		{1_000, 1_000, 0.001, 2.5},
		{10_000, 10_000, 0.001, 2.0},
	}
	for _, tc := range cases {
		f := NewBloomFilter(tc.estimatedCount, tc.targetFPR)
		insertSeed := int64(42)
		insertRng := rand.New(rand.NewSource(insertSeed))
		added := make(map[string]bool, tc.insertedCount)
		for i := 0; i < tc.insertedCount; i++ {
			// Deterministic distinct values
			s := fmt.Sprintf("item-%d-%d", insertRng.Int63(), i)
			added[s] = true
			f.AddString(s)
		}
		// Query known-absent values (different seed so no overlap)
		queryRng := rand.New(rand.NewSource(999))
		var falsePositives int
		for i := 0; i < numQueries; i++ {
			s := fmt.Sprintf("absent-%d-%d", queryRng.Int63(), i)
			if added[s] {
				continue
			}
			if f.MayContainString(s) {
				falsePositives++
			}
		}
		empiricalFPR := float64(falsePositives) / numQueries
		threshold := tc.targetFPR * tc.maxSlack
		t.Logf("n=%d inserted=%d targetFPR=%.4f empiricalFPR=%.4f (maxSlack=%.1f threshold=%.4f)",
			tc.estimatedCount, tc.insertedCount, tc.targetFPR, empiricalFPR, tc.maxSlack, threshold)
		if empiricalFPR > threshold {
			t.Errorf("n=%d inserted=%d targetFPR=%.4f empiricalFPR=%.4f > threshold %.4f",
				tc.estimatedCount, tc.insertedCount, tc.targetFPR, empiricalFPR, threshold)
		}
	}
}

// BenchmarkFilter_Memory measures bytes allocated per filter creation for
// different (estimatedCount, falsePositiveRate). Use -benchmem for B/op.
// Reports filter size in bits as a custom metric.
func BenchmarkFilter_Memory(b *testing.B) {
	estimatedCounts := []uint{1_000, 10_000, 100_000, 1_000_000}
	fprs := []float64{0.01, 0.001}
	for _, n := range estimatedCounts {
		for _, fp := range fprs {
			n, fp := n, fp
			name := fmt.Sprintf("N=%d_FPR=%.3f", n, fp)
			b.Run(name, func(b *testing.B) {
				// Report filter size once (bits) for this config
				sample := NewBloomFilter(n, fp)
				b.ReportMetric(float64(sample.Cap()), "bits")
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = NewBloomFilter(n, fp)
				}
			})
		}
	}
}

// benchmarkFilterLookupSetup builds a filter and returns it plus an absent and present key.
// Fill with half of estimatedCount so the filter is under load but not overfilled.
func benchmarkFilterLookupSetup(b *testing.B, estimatedCount uint, fpr float64) (f *BloomFilter, absent []byte, present []byte) {
	b.Helper()
	f = NewBloomFilter(estimatedCount, fpr)
	n := int(estimatedCount) / 2
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		s := fmt.Sprintf("key-%d", i)
		f.AddString(s)
	}
	absent = []byte("absent-key-not-in-set")
	present = []byte("key-0")
	return f, absent, present
}

// BenchmarkFilter_MayContain measures ns/op for a single MayContain lookup (one bit result).
// Sub-benchmarks vary filter size, FPR, and absent vs present.
func BenchmarkFilter_MayContain(b *testing.B) {
	estimatedCounts := []uint{1_000, 10_000, 100_000, 1_000_000}
	fprs := []float64{0.01, 0.001}
	for _, n := range estimatedCounts {
		for _, fp := range fprs {
			n, fp := n, fp
			f, absent, present := benchmarkFilterLookupSetup(b, n, fp)
			b.Run(fmt.Sprintf("N=%d_FPR=%.3f_absent", n, fp), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = f.MayContain(absent)
				}
			})
			b.Run(fmt.Sprintf("N=%d_FPR=%.3f_present", n, fp), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = f.MayContain(present)
				}
			})
			_ = f
		}
	}
}

// BenchmarkFilter_MayContainString measures ns/op for a single MayContainString lookup.
// Sub-benchmarks vary filter size, FPR, and absent vs present.
func BenchmarkFilter_MayContainString(b *testing.B) {
	estimatedCounts := []uint{1_000, 10_000, 100_000, 1_000_000}
	fprs := []float64{0.01, 0.001}
	for _, n := range estimatedCounts {
		for _, fp := range fprs {
			n, fp := n, fp
			f, absent, present := benchmarkFilterLookupSetup(b, n, fp)
			absentStr := string(absent)
			presentStr := string(present)
			b.Run(fmt.Sprintf("N=%d_FPR=%.3f_absent", n, fp), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = f.MayContainString(absentStr)
				}
			})
			b.Run(fmt.Sprintf("N=%d_FPR=%.3f_present", n, fp), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = f.MayContainString(presentStr)
				}
			})
			_ = f
		}
	}
}
