// Bloom filter implementation for the filter package. Use NewBloomFilterFromOptions
// or NewBloomFilter, or NewFilter(KindBloom, BloomOptions{}).
package filter

import (
	"github.com/bits-and-blooms/bloom/v3"
)

// Compile-time check that BloomFilter implements Filter.
var _ Filter = (*BloomFilter)(nil)

// BloomFilter is the Bloom-filter implementation of Filter. Use NewBloomFilterFromOptions
// or NewBloomFilter to create one.
type BloomFilter struct {
	inner *bloom.BloomFilter
}

// NewBloomFilterFromOptions creates a Bloom filter from options. When
// MaxEstimatedCount is > 0, sizing is capped to at most that many elements.
func NewBloomFilterFromOptions(o BloomOptions) *BloomFilter {
	n := o.EstimatedCount
	if o.MaxEstimatedCount > 0 && n > o.MaxEstimatedCount {
		n = o.MaxEstimatedCount
	}
	fpr := o.FalsePositiveRate
	if fpr <= 0 {
		fpr = 0.01
	}
	return &BloomFilter{
		inner: bloom.NewWithEstimates(n, fpr),
	}
}

// NewBloomFilter creates a Bloom filter sized for approximately estimatedCount
// elements with the given falsePositiveRate (e.g. 0.01 for 1%). MayContain
// returns false only when the value is definitely not in the set; true means
// the value may be in the set (possibly a false positive).
func NewBloomFilter(estimatedCount uint, falsePositiveRate float64) *BloomFilter {
	return NewBloomFilterFromOptions(BloomOptions{
		EstimatedCount:    estimatedCount,
		FalsePositiveRate: falsePositiveRate,
	})
}

// Add adds value to the filter. It is idempotent.
func (f *BloomFilter) Add(value []byte) {
	f.inner.Add(value)
}

// AddString adds the string to the filter. Convenient for IPs, domains, etc.
func (f *BloomFilter) AddString(s string) {
	f.inner.AddString(s)
}

// MayContain returns false if value is definitely not in the set, and true if
// it may be in the set (true positive or false positive). Do not treat true
// as definitive membership.
func (f *BloomFilter) MayContain(value []byte) bool {
	return f.inner.Test(value)
}

// MayContainString returns false if the string is definitely not in the set,
// and true if it may be in the set.
func (f *BloomFilter) MayContainString(s string) bool {
	return f.inner.TestString(s)
}

// Cap returns the capacity of the filter in bits (the size m of the underlying
// bit array). Use for sizing, serialization, or reporting; size in bytes is
// approximately Cap()/8.
func (f *BloomFilter) Cap() uint {
	return f.inner.Cap()
}
