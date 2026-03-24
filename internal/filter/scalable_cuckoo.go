// Scalable Cuckoo filter implementation using github.com/seiflotfy/cuckoofilter.
// ScalableCuckooOptions is accepted for API consistency; upstream does not export options.
package filter

import (
	cuckoolib "github.com/seiflotfy/cuckoofilter"
)

// Compile-time check that ScalableCuckooFilter implements Filter.
var _ Filter = (*ScalableCuckooFilter)(nil)

// ScalableCuckooFilter is the Scalable Cuckoo filter implementation of Filter.
type ScalableCuckooFilter struct {
	inner *cuckoolib.ScalableCuckooFilter
}

// NewScalableCuckooFilterFromOptions creates a Scalable Cuckoo filter. opts reserved for future use.
func NewScalableCuckooFilterFromOptions(o ScalableCuckooOptions) *ScalableCuckooFilter {
	_ = o
	return &ScalableCuckooFilter{inner: cuckoolib.NewScalableCuckooFilter()}
}

// Add adds value to the filter. Uses InsertUnique for idempotent set semantics.
func (f *ScalableCuckooFilter) Add(value []byte) {
	f.inner.InsertUnique(value)
}

// AddString adds the string to the filter.
func (f *ScalableCuckooFilter) AddString(s string) {
	f.inner.InsertUnique([]byte(s))
}

// MayContain returns false if value is definitely not in the set, and true if it may be.
func (f *ScalableCuckooFilter) MayContain(value []byte) bool {
	return f.inner.Lookup(value)
}

// MayContainString returns false if the string is definitely not in the set, and true if it may be.
func (f *ScalableCuckooFilter) MayContainString(s string) bool {
	return f.inner.Lookup([]byte(s))
}

// Cap returns 0; scalable filters have no single fixed capacity.
func (f *ScalableCuckooFilter) Cap() uint {
	return 0
}
