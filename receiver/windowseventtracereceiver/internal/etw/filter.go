// //go:build windows
// // +build windows

package etw

// import (
// 	"sync"

// 	"github.com/0xrawsec/golang-utils/datastructs"
// )

// type IEvent interface {
// 	ProviderGUID() string
// 	EventID() uint16
// }

// type EventFilter interface {
// 	// Match must return true if the event has to be filtered in
// 	Match(IEvent) bool
// 	// Update adds a filter for/from a given provider
// 	Update(p *Provider)
// }

// type baseFilter struct {
// 	sync.RWMutex
// 	m map[string]Set
// }

// func (f *baseFilter) matchKey(key string, e IEvent) bool {
// 	f.RLock()
// 	defer f.RUnlock()

// 	// map is nil
// 	if f.m == nil {
// 		return true
// 	}

// 	// Filter is empty
// 	if len(f.m) == 0 {
// 		return true
// 	}

// 	if eventids, ok := f.m[key]; ok {
// 		if eventids.Len() > 0 {
// 			return eventids.Contains(e.EventID())
// 		}
// 		return true
// 	}
// 	// we return true if no filter is found
// 	return true
// }

// // ProviderFilter structure to filter events based on Provider
// // definition
// type ProviderFilter struct {
// 	baseFilter
// }

// // Set datastruct that represent a thread safe set
// type Set struct {
// 	i   uint
// 	set map[interface{}]uint
// }

// func (s *Set) Len() int {
// 	return len(s.set)
// }

// func (s *Set) Contains(e interface{}) bool {
// 	return s.set[e] != 0
// }

// // NewProviderFilter creates a new ProviderFilter structure
// func NewProviderFilter() *ProviderFilter {
// 	f := ProviderFilter{}
// 	f.m = make(map[string]Set)
// 	return &f
// }

// // Match implements EventFilter
// func (f *ProviderFilter) Match(e IEvent) bool {
// 	return f.matchKey(e.ProviderGUID(), e)
// }

// // FromProvider implements EventFilter
// func (f *ProviderFilter) Update(p *Provider) {
// 	f.Lock()
// 	defer f.Unlock()
// 	if len(p.Filter) > 0 {
// 		s := datastructs.ToInterfaceSlice(p.Filter)
// 		f.m[p.GUID] = datastructs.NewInitSet(s...)
// 	}
// }
