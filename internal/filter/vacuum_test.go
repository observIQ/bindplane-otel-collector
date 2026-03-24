package filter

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestVacuumFilter_Add_MayContain(t *testing.T) {
	f := NewVacuumFilter(1000)

	vals := [][]byte{[]byte("a"), []byte("b"), []byte("1.2.3.4")}
	for _, v := range vals {
		f.Add(v)
		if !f.MayContain(v) {
			t.Errorf("MayContain(%q) = false after Add", v)
		}
	}

	absent := []byte("definitely-not-added")
	if f.MayContain(absent) {
		t.Logf("MayContain(absent) was true (false positive); filter is very small")
	}
}

func TestVacuumFilter_AddString_MayContainString(t *testing.T) {
	f := NewVacuumFilter(1000)

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

func TestVacuumFilter_MayContain_DefinitelyNotPresent(t *testing.T) {
	f := NewVacuumFilter(100_000)
	f.Add([]byte("only-one"))
	if f.MayContain([]byte("different-value")) {
		t.Error("MayContain(different-value) = true but we never added it")
	}
}

func TestVacuumFilter_Delete(t *testing.T) {
	f := NewVacuumFilter(1000)

	f.AddString("to-delete")
	if !f.MayContainString("to-delete") {
		t.Fatal("MayContainString(to-delete) = false after Add")
	}

	deleted := f.DeleteString("to-delete")
	if !deleted {
		t.Error("DeleteString(to-delete) = false, expected true")
	}

	if f.MayContainString("to-delete") {
		t.Error("MayContainString(to-delete) = true after Delete")
	}

	deletedAgain := f.DeleteString("to-delete")
	if deletedAgain {
		t.Error("DeleteString(to-delete) = true on second delete, expected false")
	}
}

func TestVacuumFilter_Delete_NonExistent(t *testing.T) {
	f := NewVacuumFilter(1000)
	deleted := f.DeleteString("never-added")
	if deleted {
		t.Error("DeleteString(never-added) = true, expected false")
	}
}

func TestNewFilter_KindVacuum(t *testing.T) {
	f, err := NewFilter(KindVacuum, VacuumOptions{Capacity: 1000})
	if err != nil {
		t.Fatalf("NewFilter(KindVacuum, ...): %v", err)
	}
	f.AddString("test")
	if !f.MayContainString("test") {
		t.Error("filter from NewFilter did not contain added value")
	}
	if f.Cap() == 0 {
		t.Error("Cap() = 0")
	}
}

func TestNewFilterFromConfig_VacuumOptions(t *testing.T) {
	f, err := NewFilterFromConfig(VacuumOptions{Capacity: 1000})
	if err != nil {
		t.Fatalf("NewFilterFromConfig(VacuumOptions): %v", err)
	}
	f.AddString("test")
	if !f.MayContainString("test") {
		t.Error("filter from NewFilterFromConfig did not contain added value")
	}
	if f.Cap() == 0 {
		t.Error("Cap() = 0")
	}
}

func TestVacuumFilter_AltIndexSymmetry(t *testing.T) {
	f := NewVacuumFilter(100_000)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		index := uint(rng.Intn(int(f.numBuckets)))
		tag := uint32(rng.Intn(255) + 1)

		alt := f.altIndex(index, tag)
		recovered := f.altIndex(alt, tag)
		if recovered != index {
			t.Errorf("altIndex symmetry failed: index=%d, tag=%d, alt=%d, recovered=%d",
				index, tag, alt, recovered)
		}
	}
}

func TestVacuumFilter_LoadFactor(t *testing.T) {
	capacity := uint(50_000)
	f := NewVacuumFilter(capacity)

	rng := rand.New(rand.NewSource(42))
	inserted := 0
	for i := 0; i < int(capacity)*2; i++ {
		key := fmt.Sprintf("key-%d-%d", rng.Int63(), i)
		countBefore := f.Count()
		f.AddString(key)
		if f.Count() > countBefore {
			inserted++
		}
		if f.victim.used {
			break
		}
	}

	loadFactor := f.LoadFactor()
	t.Logf("Vacuum filter: inserted=%d, loadFactor=%.2f%%, capacity=%d",
		inserted, loadFactor*100, f.Cap())

	if loadFactor < 0.90 {
		t.Errorf("LoadFactor=%.2f%%, expected >= 90%% (Vacuum should achieve ~95%%)",
			loadFactor*100)
	}
}

func TestVacuumFilter_HydrateWithRandomIPs(t *testing.T) {
	const n = 5000
	f := NewVacuumFilter(uint(n))
	rng := rand.New(rand.NewSource(42))
	added := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d",
			rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256))
		added[ip] = true
		f.AddString(ip)
	}

	for ip := range added {
		if !f.MayContainString(ip) {
			t.Errorf("MayContainString(%q) = false but we added it", ip)
		}
	}

	rng2 := rand.New(rand.NewSource(999))
	for i := 0; i < 200; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d",
			rng2.Intn(256), rng2.Intn(256), rng2.Intn(256), rng2.Intn(256))
		if added[ip] {
			continue
		}
		if !f.MayContainString(ip) {
			break
		}
	}
}

func TestVacuumFilter_FalsePositiveRate(t *testing.T) {
	const numQueries = 50_000
	// With 8-bit fingerprints and 2 bucket locations, theoretical FPR is ~2*4/256 ≈ 3.1%
	// In practice we see 1-2.5% depending on load
	cases := []struct {
		capacity      uint
		insertedCount int
		maxFPR        float64
	}{
		{10_000, 5_000, 0.02},
		{10_000, 10_000, 0.025},
		{100_000, 50_000, 0.02},
		{100_000, 100_000, 0.03},
	}

	for _, tc := range cases {
		f := NewVacuumFilter(tc.capacity)
		insertRng := rand.New(rand.NewSource(42))
		added := make(map[string]bool, tc.insertedCount)
		for i := 0; i < tc.insertedCount; i++ {
			s := fmt.Sprintf("item-%d-%d", insertRng.Int63(), i)
			added[s] = true
			f.AddString(s)
		}

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
		t.Logf("capacity=%d inserted=%d empiricalFPR=%.4f maxFPR=%.4f",
			tc.capacity, tc.insertedCount, empiricalFPR, tc.maxFPR)
		if empiricalFPR > tc.maxFPR {
			t.Errorf("capacity=%d inserted=%d empiricalFPR=%.4f > maxFPR=%.4f",
				tc.capacity, tc.insertedCount, empiricalFPR, tc.maxFPR)
		}
	}
}

func TestFilterSet_AddVacuumFilter(t *testing.T) {
	s := NewFilterSet()
	f := s.AddVacuumFilter("threats", 10_000)
	f.AddString("malicious.com")

	got := s.Filter("threats")
	if got == nil {
		t.Fatal("Filter(\"threats\") is nil")
	}
	if !got.MayContainString("malicious.com") {
		t.Error("threats filter did not contain malicious.com")
	}
}

func benchmarkVacuumFilterSetup(b *testing.B, capacity uint) (f *VacuumFilter, absent []byte, present []byte) {
	b.Helper()
	f = NewVacuumFilter(capacity)
	n := int(capacity) / 2
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

func BenchmarkVacuumFilter_MayContain(b *testing.B) {
	capacities := []uint{1_000, 10_000, 100_000, 1_000_000}
	for _, cap := range capacities {
		cap := cap
		f, absent, present := benchmarkVacuumFilterSetup(b, cap)
		b.Run(fmt.Sprintf("N=%d_absent", cap), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.MayContain(absent)
			}
		})
		b.Run(fmt.Sprintf("N=%d_present", cap), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.MayContain(present)
			}
		})
	}
}

func BenchmarkVacuumFilter_Add(b *testing.B) {
	capacities := []uint{10_000, 100_000, 1_000_000}
	for _, cap := range capacities {
		cap := cap
		b.Run(fmt.Sprintf("N=%d", cap), func(b *testing.B) {
			f := NewVacuumFilter(cap)
			keys := make([][]byte, b.N)
			for i := range keys {
				keys[i] = []byte(fmt.Sprintf("key-%d", i))
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f.Add(keys[i])
			}
		})
	}
}

func BenchmarkVacuumFilter_Memory(b *testing.B) {
	capacities := []uint{1_000, 10_000, 100_000, 1_000_000}
	for _, cap := range capacities {
		cap := cap
		b.Run(fmt.Sprintf("N=%d", cap), func(b *testing.B) {
			sample := NewVacuumFilter(cap)
			b.ReportMetric(float64(sample.SizeInBytes()), "bytes")
			b.ReportMetric(float64(sample.Cap()), "slots")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = NewVacuumFilter(cap)
			}
		})
	}
}
