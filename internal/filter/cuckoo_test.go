package filter

import (
	"fmt"
	"testing"
)

func TestCuckooFilter_Add_MayContain(t *testing.T) {
	f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: 1000})
	vals := [][]byte{[]byte("a"), []byte("b"), []byte("1.2.3.4")}
	for _, v := range vals {
		f.Add(v)
		if !f.MayContain(v) {
			t.Errorf("MayContain(%q) = false after Add", v)
		}
	}
	if f.MayContain([]byte("absent")) {
		t.Error("MayContain(absent) = true but we never added it")
	}
}

func TestCuckooFilter_AddString_MayContainString(t *testing.T) {
	f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: 1000})
	f.AddString("192.168.1.1")
	f.AddString("example.com")
	if !f.MayContainString("192.168.1.1") {
		t.Error("MayContainString(192.168.1.1) = false after AddString")
	}
	if !f.MayContainString("example.com") {
		t.Error("MayContainString(example.com) = false after AddString")
	}
	if f.MayContainString("10.0.0.1") {
		t.Error("MayContainString(10.0.0.1) = true but we never added it")
	}
}

func TestCuckooFilter_Cap(t *testing.T) {
	f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: 5000})
	if f.Cap() != 5000 {
		t.Errorf("Cap() = %d, want 5000", f.Cap())
	}
}

func TestNewFilterFromConfig_CuckooOptions(t *testing.T) {
	f, err := NewFilterFromConfig(CuckooOptions{Capacity: 1000})
	if err != nil {
		t.Fatalf("NewFilterFromConfig(CuckooOptions): %v", err)
	}
	f.AddString("test")
	if !f.MayContainString("test") {
		t.Error("filter did not contain added value")
	}
}

func TestNewFilter_KindCuckoo(t *testing.T) {
	f, err := NewFilter(KindCuckoo, CuckooOptions{Capacity: 1000})
	if err != nil {
		t.Fatalf("NewFilter(KindCuckoo, ...): %v", err)
	}
	f.AddString("x")
	if !f.MayContainString("x") {
		t.Error("filter did not contain added value")
	}
}

func benchmarkCuckooLookupSetup(b *testing.B, capacity uint) (f *CuckooFilter, absent []byte, present []byte) {
	b.Helper()
	f = NewCuckooFilterFromOptions(CuckooOptions{Capacity: capacity})
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

func BenchmarkCuckooFilter_Memory(b *testing.B) {
	capacities := []uint{1_000, 10_000, 100_000, 1_000_000}
	for _, n := range capacities {
		n := n
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			sample := NewCuckooFilterFromOptions(CuckooOptions{Capacity: n})
			b.ReportMetric(float64(sample.Cap()), "capacity")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = NewCuckooFilterFromOptions(CuckooOptions{Capacity: n})
			}
		})
	}
}

func BenchmarkCuckooFilter_MayContain(b *testing.B) {
	capacities := []uint{1_000, 10_000, 100_000, 1_000_000}
	for _, n := range capacities {
		n := n
		f, absent, present := benchmarkCuckooLookupSetup(b, n)
		b.Run(fmt.Sprintf("N=%d_absent", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.MayContain(absent)
			}
		})
		b.Run(fmt.Sprintf("N=%d_present", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.MayContain(present)
			}
		})
		_ = f
	}
}

func BenchmarkCuckooFilter_MayContainString(b *testing.B) {
	capacities := []uint{1_000, 10_000, 100_000, 1_000_000}
	for _, n := range capacities {
		n := n
		f, absent, present := benchmarkCuckooLookupSetup(b, n)
		absentStr := string(absent)
		presentStr := string(present)
		b.Run(fmt.Sprintf("N=%d_absent", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.MayContainString(absentStr)
			}
		})
		b.Run(fmt.Sprintf("N=%d_present", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.MayContainString(presentStr)
			}
		})
		_ = f
	}
}
