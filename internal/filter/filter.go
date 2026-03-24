// Package filter provides a pluggable abstraction for probabilistic (or exact)
// set membership, used for threat intelligence and other filtering. Implementations
// include Bloom, Cuckoo, and Scalable Cuckoo filters; create them via
// NewFilterFromConfig with the appropriate options type.
//
// A Filter answers: "might this value be in the set?" MayContain returns false
// only when the value is definitely not in the set. When MayContain returns true,
// the value may be in the set (true positive) or may be a false positive; callers
// must not treat true as definitive membership.
package filter

import "fmt"

// Filter is the common interface for set-membership filters. Values are []byte
// (or strings via AddString/MayContainString) so any observable type (IPs,
// domains, hashes, etc.) can be used.
type Filter interface {
	Add(value []byte)
	AddString(s string)
	MayContain(value []byte) bool
	MayContainString(s string) bool
	Cap() uint
}

// FilterConfig is the dependency-injection interface for filter configuration.
// Each filter kind has its own options struct implementing FilterConfig; pass
// one to NewFilterFromConfig to create a filter. Only our option types implement it.
type FilterConfig interface {
	FilterKind() Kind
}

// Kind identifies the filter algorithm (bloom, cuckoo, scalable cuckoo).
type Kind string

const (
	KindBloom          Kind = "bloom"
	KindCuckoo         Kind = "cuckoo"
	KindScalableCuckoo Kind = "scalable_cuckoo"
	KindVacuum         Kind = "vacuum"
)

// BloomOptions configures a Bloom filter. Used with NewFilterFromConfig or
// NewFilter(KindBloom, opts). MaxEstimatedCount caps sizing when set (0 = no cap).
type BloomOptions struct {
	EstimatedCount    uint
	FalsePositiveRate float64
	MaxEstimatedCount uint // 0 = no cap; filter is sized for at most this many elements when set
}

// FilterKind implements FilterConfig.
func (BloomOptions) FilterKind() Kind { return KindBloom }

// CuckooOptions configures a Cuckoo filter. Capacity is the expected number of elements.
type CuckooOptions struct {
	Capacity uint
}

// FilterKind implements FilterConfig.
func (CuckooOptions) FilterKind() Kind { return KindCuckoo }

// ScalableCuckooOptions configures a Scalable Cuckoo filter. Zero values use
// library defaults (InitialCapacity 10000, LoadFactor 0.9). The upstream
// library does not export options; these fields are applied when using a
// vendored copy with an extended constructor; otherwise defaults are used.
type ScalableCuckooOptions struct {
	InitialCapacity uint    // default 10000
	LoadFactor      float32 // default 0.9
}

// FilterKind implements FilterConfig.
func (ScalableCuckooOptions) FilterKind() Kind { return KindScalableCuckoo }

// VacuumOptions configures a Vacuum filter. Capacity is the expected number of
// elements. FingerprintBits controls the fingerprint size (default 8, max 32).
type VacuumOptions struct {
	Capacity        uint
	FingerprintBits uint // default 8
}

// FilterKind implements FilterConfig.
func (VacuumOptions) FilterKind() Kind { return KindVacuum }

// NewFilterFromConfig creates a filter from a config (dependency injection).
// The concrete type of c determines the filter implementation.
func NewFilterFromConfig(c FilterConfig) (Filter, error) {
	switch c.FilterKind() {
	case KindBloom:
		o, ok := c.(BloomOptions)
		if !ok {
			return nil, fmt.Errorf("filter: Bloom requires BloomOptions, got %T", c)
		}
		return NewBloomFilterFromOptions(o), nil
	case KindCuckoo:
		o, ok := c.(CuckooOptions)
		if !ok {
			return nil, fmt.Errorf("filter: Cuckoo requires CuckooOptions, got %T", c)
		}
		return NewCuckooFilterFromOptions(o), nil
	case KindScalableCuckoo:
		o, ok := c.(ScalableCuckooOptions)
		if !ok {
			return nil, fmt.Errorf("filter: ScalableCuckoo requires ScalableCuckooOptions, got %T", c)
		}
		return NewScalableCuckooFilterFromOptions(o), nil
	case KindVacuum:
		o, ok := c.(VacuumOptions)
		if !ok {
			return nil, fmt.Errorf("filter: Vacuum requires VacuumOptions, got %T", c)
		}
		return NewVacuumFilterFromOptions(o), nil
	default:
		return nil, fmt.Errorf("filter: unknown kind %q", c.FilterKind())
	}
}

// NewFilter creates a filter of the given kind with the provided options.
// opts must be the matching options type (BloomOptions, CuckooOptions, or ScalableCuckooOptions).
// Prefer NewFilterFromConfig for dependency injection.
func NewFilter(kind Kind, opts interface{}) (Filter, error) {
	switch kind {
	case KindBloom:
		o, ok := opts.(BloomOptions)
		if !ok {
			return nil, fmt.Errorf("filter: Bloom requires BloomOptions, got %T", opts)
		}
		return NewFilterFromConfig(o)
	case KindCuckoo:
		o, ok := opts.(CuckooOptions)
		if !ok {
			return nil, fmt.Errorf("filter: Cuckoo requires CuckooOptions, got %T", opts)
		}
		return NewFilterFromConfig(o)
	case KindScalableCuckoo:
		o, ok := opts.(ScalableCuckooOptions)
		if !ok {
			return nil, fmt.Errorf("filter: ScalableCuckoo requires ScalableCuckooOptions, got %T", opts)
		}
		return NewFilterFromConfig(o)
	case KindVacuum:
		o, ok := opts.(VacuumOptions)
		if !ok {
			return nil, fmt.Errorf("filter: Vacuum requires VacuumOptions, got %T", opts)
		}
		return NewFilterFromConfig(o)
	default:
		return nil, fmt.Errorf("filter: unknown kind %q", kind)
	}
}

// FilterSet holds multiple named filters for grouped data sets (e.g. "ips",
// "domains"). It is algorithm-agnostic and not safe for concurrent use.
type FilterSet struct {
	filters map[string]Filter
}

// NewFilterSet returns an empty set of named filters.
func NewFilterSet() *FilterSet {
	return &FilterSet{
		filters: make(map[string]Filter),
	}
}

// AddFilterFromConfig creates a filter from the given config, stores it under name,
// and returns it so callers can hydrate it.
func (s *FilterSet) AddFilterFromConfig(name string, c FilterConfig) (Filter, error) {
	f, err := NewFilterFromConfig(c)
	if err != nil {
		return nil, err
	}
	s.filters[name] = f
	return f, nil
}

// AddFilter creates a filter with the given name using the specified kind and
// options, stores it in the set, and returns it so callers can hydrate it.
func (s *FilterSet) AddFilter(name string, kind Kind, opts interface{}) (Filter, error) {
	f, err := NewFilter(kind, opts)
	if err != nil {
		return nil, err
	}
	s.filters[name] = f
	return f, nil
}

// AddBloomFilter is a convenience for adding a Bloom filter by capacity and
// false-positive rate. It returns the filter for hydration.
func (s *FilterSet) AddBloomFilter(name string, estimatedCount uint, falsePositiveRate float64) Filter {
	f := NewBloomFilter(estimatedCount, falsePositiveRate)
	s.filters[name] = f
	return f
}

// AddCuckooFilter is a convenience for adding a Cuckoo filter by capacity.
func (s *FilterSet) AddCuckooFilter(name string, capacity uint) Filter {
	f := NewCuckooFilterFromOptions(CuckooOptions{Capacity: capacity})
	s.filters[name] = f
	return f
}

// AddScalableCuckooFilter adds a Scalable Cuckoo filter with the given options.
// Zero opts use library defaults.
func (s *FilterSet) AddScalableCuckooFilter(name string, opts ScalableCuckooOptions) Filter {
	f := NewScalableCuckooFilterFromOptions(opts)
	s.filters[name] = f
	return f
}

// AddVacuumFilter adds a Vacuum filter with the given capacity.
func (s *FilterSet) AddVacuumFilter(name string, capacity uint) Filter {
	f := NewVacuumFilterFromOptions(VacuumOptions{Capacity: capacity})
	s.filters[name] = f
	return f
}

// Filter returns the named filter, or nil if no such name exists.
func (s *FilterSet) Filter(name string) Filter {
	return s.filters[name]
}

// Names returns the names of all filters in the set.
func (s *FilterSet) Names() []string {
	names := make([]string, 0, len(s.filters))
	for n := range s.filters {
		names = append(names, n)
	}
	return names
}
