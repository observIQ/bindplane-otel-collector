// Vacuum filter implementation for the filter package. The Vacuum filter is an
// improved cuckoo filter that achieves ~95% load factor (vs ~84% for standard
// cuckoo) through variable alternate ranges computed using balls-in-bins analysis.
//
// Reference: "Vacuum Filters: More Space-Efficient and Faster Replacement for
// Bloom and Cuckoo Filters" (VLDB 2020)
// https://github.com/wuwuz/Vacuum-Filter
package filter

import (
	"math"
	"math/rand"

	"github.com/twmb/murmur3"
)

// Compile-time check that VacuumFilter implements Filter.
var _ Filter = (*VacuumFilter)(nil)

const (
	vacuumAR             = 4   // number of fingerprint classes (AR in reference)
	vacuumBucketSize     = 4   // slots per bucket
	vacuumMaxCuckooCount = 500 // max eviction attempts before using victim cache
	vacuumDefaultFPBits  = 8   // default bits per fingerprint
)

// vacuumBucket holds 4 fingerprints per bucket.
type vacuumBucket [vacuumBucketSize]uint32

func (b *vacuumBucket) insert(fp uint32) bool {
	for i := 0; i < vacuumBucketSize; i++ {
		if b[i] == 0 {
			b[i] = fp
			return true
		}
	}
	return false
}

func (b *vacuumBucket) delete(fp uint32) bool {
	for i := 0; i < vacuumBucketSize; i++ {
		if b[i] == fp {
			b[i] = 0
			return true
		}
	}
	return false
}

func (b *vacuumBucket) contains(fp uint32) bool {
	for i := 0; i < vacuumBucketSize; i++ {
		if b[i] == fp {
			return true
		}
	}
	return false
}

func (b *vacuumBucket) readAll() [vacuumBucketSize]uint32 {
	return *b
}

func (b *vacuumBucket) writeSlot(i int, fp uint32) {
	b[i] = fp
}

// VacuumFilter is the Vacuum filter implementation of Filter.
type VacuumFilter struct {
	buckets    []vacuumBucket
	numBuckets uint
	count      uint
	altLen     [vacuumAR]int // variable alternate ranges per fingerprint class
	bigSeg     int
	fpMask     uint32
	fpBits     uint
	victim     struct {
		index uint
		tag   uint32
		used  bool
	}
	rng *rand.Rand
}

// ballsInBinsMaxLoad computes the expected maximum load when throwing balls
// into bins uniformly at random. This is used to compute optimal alternate ranges.
// Reference: cuckoofilter.h line 54-60
func ballsInBinsMaxLoad(balls, bins float64) float64 {
	if bins == 1 {
		return balls
	}
	return (balls / bins) + 1.5*math.Sqrt(2*balls/bins*math.Log(bins))
}

// properAltRange computes the optimal alternate range for fingerprint class i.
// M is the number of buckets, i is the fingerprint class (0-3).
// Reference: cuckoofilter.h line 62-73
func properAltRange(M, i int) int {
	b := 4.0   // slots per bucket
	lf := 0.95 // target load factor
	altRange := 8
	for altRange < M {
		f := (4.0 - float64(i)) * 0.25
		if ballsInBinsMaxLoad(f*b*lf*float64(M), float64(M)/float64(altRange)) < 0.97*b*float64(altRange) {
			return altRange
		}
		altRange <<= 1
	}
	return altRange
}

// roundUp rounds a up to the nearest multiple of b.
func roundUp(a, b int) int {
	return ((a + b - 1) / b) * b
}

// NewVacuumFilterFromOptions creates a Vacuum filter from options.
func NewVacuumFilterFromOptions(o VacuumOptions) *VacuumFilter {
	capacity := o.Capacity
	if capacity == 0 {
		capacity = 10000
	}
	fpBits := o.FingerprintBits
	if fpBits == 0 {
		fpBits = vacuumDefaultFPBits
	}
	if fpBits > 32 {
		fpBits = 32
	}

	// Compute number of buckets: capacity / 0.96 / 4
	// Reference: cuckoofilter.h line 139-140
	numBuckets := int(float64(capacity) / 0.96 / 4)
	if numBuckets < 128 {
		numBuckets = 128
	}

	f := &VacuumFilter{
		fpBits: fpBits,
		fpMask: (1 << fpBits) - 1,
		rng:    rand.New(rand.NewSource(1)),
	}

	// Compute bigSeg and altLen array
	// Reference: cuckoofilter.h line 141-157
	bigSeg := 1024
	computed := properAltRange(numBuckets, 0)
	if computed > bigSeg {
		bigSeg = computed
	}
	numBuckets = roundUp(numBuckets, bigSeg)

	f.bigSeg = bigSeg - 1
	f.altLen[0] = f.bigSeg
	for i := 1; i < vacuumAR; i++ {
		f.altLen[i] = properAltRange(numBuckets, i) - 1
	}
	// Special case for last class: double the range
	// Reference: cuckoofilter.h line 154
	f.altLen[vacuumAR-1] = (f.altLen[vacuumAR-1]+1)*2 - 1

	f.numBuckets = uint(numBuckets)
	f.buckets = make([]vacuumBucket, numBuckets)

	return f
}

// NewVacuumFilter creates a Vacuum filter sized for the given capacity.
func NewVacuumFilter(capacity uint) *VacuumFilter {
	return NewVacuumFilterFromOptions(VacuumOptions{Capacity: capacity})
}

// indexAndTag computes the primary bucket index and fingerprint for data.
func (f *VacuumFilter) indexAndTag(data []byte) (index uint, tag uint32) {
	hash := murmur3.Sum64(data)
	// Use multiplicative hashing for index (supports non-power-of-2 sizes)
	// Reference: cuckoofilter.h line 104-105
	index = uint((uint64(hash>>32) * uint64(f.numBuckets)) >> 32)
	tag = uint32(hash) & f.fpMask
	// Zero is reserved for empty slots
	if tag == 0 {
		tag = 1
	}
	return
}

// altIndex computes the alternate bucket index for a given index and tag.
// This is THE KEY INNOVATION of the Vacuum filter: variable alternate ranges
// based on fingerprint class.
// Reference: cuckoofilter.h line 111-118
func (f *VacuumFilter) altIndex(index uint, tag uint32) uint {
	t := int(tag * 0x5bd1e995)
	seg := f.altLen[tag&(vacuumAR-1)]
	t = t & seg
	if t == 0 {
		t = 1
	}
	return index ^ uint(t)
}

// addImpl implements insertion with lookahead eviction.
// Reference: cuckoofilter.h line 287-327
func (f *VacuumFilter) addImpl(index uint, tag uint32) bool {
	// Try primary bucket first
	if f.buckets[index].insert(tag) {
		f.count++
		return true
	}

	// Try alternate bucket
	altBucket := f.altIndex(index, tag)
	if f.buckets[altBucket].insert(tag) {
		f.count++
		return true
	}

	// Need to evict - use lookahead strategy
	curIndex := index
	curTag := tag

	for kick := 0; kick < vacuumMaxCuckooCount; kick++ {
		// Try to insert current tag
		if f.buckets[curIndex].insert(curTag) {
			f.count++
			return true
		}

		// Lookahead: check all 4 slots' alternates for empty space
		// Reference: cuckoofilter.h line 304-315
		tags := f.buckets[curIndex].readAll()
		for i := 0; i < vacuumBucketSize; i++ {
			alt := f.altIndex(curIndex, tags[i])
			if f.buckets[alt].insert(tags[i]) {
				// Found space in alternate, replace slot with our tag
				f.buckets[curIndex].writeSlot(i, curTag)
				f.count++
				return true
			}
		}

		// No space found, randomly evict one slot
		// Reference: cuckoofilter.h line 317-322
		r := f.rng.Intn(vacuumBucketSize)
		oldTag := tags[r]
		f.buckets[curIndex].writeSlot(r, curTag)
		curTag = oldTag
		curIndex = f.altIndex(curIndex, curTag)
	}

	// Failed to insert after max kicks, use victim cache
	f.victim.index = curIndex
	f.victim.tag = curTag
	f.victim.used = true
	f.count++
	return true
}

// Add adds value to the filter.
func (f *VacuumFilter) Add(value []byte) {
	if f.victim.used {
		return // filter is full
	}
	index, tag := f.indexAndTag(value)
	f.addImpl(index, tag)
}

// AddString adds the string to the filter.
func (f *VacuumFilter) AddString(s string) {
	f.Add([]byte(s))
}

// MayContain returns false if value is definitely not in the set, and true if
// it may be in the set.
// Reference: cuckoofilter.h line 226-237
func (f *VacuumFilter) MayContain(value []byte) bool {
	index, tag := f.indexAndTag(value)

	// Check primary bucket or victim cache
	if f.buckets[index].contains(tag) {
		return true
	}
	if f.victim.used && f.victim.index == index && f.victim.tag == tag {
		return true
	}

	// Check alternate bucket or victim cache
	altIndex := f.altIndex(index, tag)
	if f.buckets[altIndex].contains(tag) {
		return true
	}
	if f.victim.used && f.victim.index == altIndex && f.victim.tag == tag {
		return true
	}

	return false
}

// MayContainString returns false if the string is definitely not in the set,
// and true if it may be in the set.
func (f *VacuumFilter) MayContainString(s string) bool {
	return f.MayContain([]byte(s))
}

// Delete removes a value from the filter. Returns true if the value was found
// and deleted. Note: deleting a value that was never added may delete a
// different value due to fingerprint collisions.
// Reference: cuckoofilter.h line 343-366
func (f *VacuumFilter) Delete(value []byte) bool {
	index, tag := f.indexAndTag(value)

	// Try to delete from primary bucket
	if f.buckets[index].delete(tag) {
		f.count--
		f.tryEliminateVictim()
		return true
	}

	// Try to delete from alternate bucket
	altIndex := f.altIndex(index, tag)
	if f.buckets[altIndex].delete(tag) {
		f.count--
		f.tryEliminateVictim()
		return true
	}

	// Check victim cache
	if f.victim.used && f.victim.tag == tag &&
		(f.victim.index == index || f.victim.index == altIndex) {
		f.victim.used = false
		return true
	}

	return false
}

// DeleteString removes a string from the filter.
func (f *VacuumFilter) DeleteString(s string) bool {
	return f.Delete([]byte(s))
}

// tryEliminateVictim attempts to reinsert the victim cache item after a deletion.
// Reference: cuckoofilter.h line 358-363
func (f *VacuumFilter) tryEliminateVictim() {
	if !f.victim.used {
		return
	}
	f.victim.used = false
	f.count-- // will be re-incremented by addImpl
	f.addImpl(f.victim.index, f.victim.tag)
}

// Cap returns the configured capacity in elements.
func (f *VacuumFilter) Cap() uint {
	return f.numBuckets * vacuumBucketSize
}

// Count returns the number of items currently stored.
func (f *VacuumFilter) Count() uint {
	return f.count
}

// LoadFactor returns the current load factor (fraction of slots used).
func (f *VacuumFilter) LoadFactor() float64 {
	return float64(f.count) / float64(f.numBuckets*vacuumBucketSize)
}

// SizeInBytes returns the size of the filter in bytes (excluding overhead).
func (f *VacuumFilter) SizeInBytes() uint {
	return f.numBuckets * vacuumBucketSize * 4 // 4 bytes per uint32 fingerprint
}
