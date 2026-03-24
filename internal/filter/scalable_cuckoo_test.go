package filter

import (
	"fmt"
	"testing"
)

func TestScalableCuckooFilter_Add_MayContain(t *testing.T) {
	f := NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
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

func TestScalableCuckooFilter_AddString_MayContainString(t *testing.T) {
	f := NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
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

func TestScalableCuckooFilter_Cap(t *testing.T) {
	f := NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
	if f.Cap() != 0 {
		t.Errorf("Cap() = %d, want 0", f.Cap())
	}
}

func TestNewFilterFromConfig_ScalableCuckooOptions(t *testing.T) {
	f, err := NewFilterFromConfig(ScalableCuckooOptions{})
	if err != nil {
		t.Fatalf("NewFilterFromConfig(ScalableCuckooOptions): %v", err)
	}
	f.AddString("test")
	if !f.MayContainString("test") {
		t.Error("filter did not contain added value")
	}
}

func TestNewFilter_KindScalableCuckoo(t *testing.T) {
	f, err := NewFilter(KindScalableCuckoo, ScalableCuckooOptions{})
	if err != nil {
		t.Fatalf("NewFilter(KindScalableCuckoo, ...): %v", err)
	}
	f.AddString("x")
	if !f.MayContainString("x") {
		t.Error("filter did not contain added value")
	}
}

func benchmarkScalableCuckooLookupSetup(b *testing.B, fillCount int) (f *ScalableCuckooFilter, absent []byte, present []byte) {
	b.Helper()
	f = NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
	if fillCount < 1 {
		fillCount = 1
	}
	for i := 0; i < fillCount; i++ {
		s := fmt.Sprintf("key-%d", i)
		f.AddString(s)
	}
	absent = []byte("absent-key-not-in-set")
	present = []byte("key-0")
	return f, absent, present
}

func BenchmarkScalableCuckooFilter_Memory(b *testing.B) {
	b.Run("create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewScalableCuckooFilterFromOptions(ScalableCuckooOptions{})
		}
	})
}

func BenchmarkScalableCuckooFilter_MayContain(b *testing.B) {
	fillCounts := []int{500, 5_000, 50_000}
	for _, n := range fillCounts {
		n := n
		f, absent, present := benchmarkScalableCuckooLookupSetup(b, n)
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

func BenchmarkScalableCuckooFilter_MayContainString(b *testing.B) {
	fillCounts := []int{500, 5_000, 50_000}
	for _, n := range fillCounts {
		n := n
		f, absent, present := benchmarkScalableCuckooLookupSetup(b, n)
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
