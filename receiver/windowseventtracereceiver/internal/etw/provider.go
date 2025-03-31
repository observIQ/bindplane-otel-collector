//go:build windows
// +build windows

package etw

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

var (
	providers map[string]*Provider

	DefaultProvider = Provider{EnableLevel: 0xff}

	// Error returned when a provider is not found on the system
	ErrUnkownProvider = fmt.Errorf("unknown provider")
)

type ProviderMap map[string]*Provider

type Provider struct {
	GUID            string
	Name            string
	EnableLevel     uint8
	MatchAnyKeyword uint64
	MatchAllKeyword uint64
	Filter          []uint16
}

// IsZero returns true if the provider is empty
func (p *Provider) IsZero() bool {
	return p.GUID == ""
}

func (p *Provider) eventIDFilterDescriptor() (d EventFilterDescriptor) {

	efeid := AllocEventFilterEventID(p.Filter)
	efeid.FilterIn = 0x1

	d = EventFilterDescriptor{
		Ptr:  uint64(uintptr(unsafe.Pointer(efeid))),
		Size: uint32(efeid.Size()),
		Type: EVENT_FILTER_TYPE_EVENT_ID,
	}

	return
}

func (p *Provider) BuildFilterDesc() (fd []EventFilterDescriptor) {

	fd = append(fd, p.eventIDFilterDescriptor())

	return
}

// MustParseProvider parses a provider string or panic
func MustParseProvider(s string) (p Provider) {
	var err error
	if p, err = ParseProvider(s); err != nil {
		panic(err)
	}
	return
}

// IsKnownProvider returns true if the provider is known
func IsKnownProvider(p string) bool {
	prov := ResolveProvider(p)
	return !prov.IsZero()
}

// ParseProvider parses a string and returns a provider.
// The returned provider is initialized from DefaultProvider.
// Format (Name|GUID) string:EnableLevel uint8:Event IDs comma sep string:MatchAnyKeyword uint16:MatchAllKeyword uint16
// Example: Microsoft-Windows-Kernel-File:0xff:13,14:0x80
func ParseProvider(s string) (p Provider, err error) {
	var u uint64

	split := strings.Split(s, ":")
	for i := 0; i < len(split); i++ {
		chunk := split[i]
		switch i {
		case 0:
			p = ResolveProvider(chunk)
			if p.IsZero() {
				err = fmt.Errorf("%w %s", ErrUnkownProvider, chunk)
				return
			}
		case 1:
			if chunk == "" {
				break
			}
			// parsing EnableLevel
			if u, err = strconv.ParseUint(chunk, 0, 8); err != nil {
				err = fmt.Errorf("failed to parse EnableLevel: %w", err)
				return
			} else {
				p.EnableLevel = uint8(u)
			}
		case 2:
			if chunk == "" {
				break
			}
			// parsing event ids
			for _, eid := range strings.Split(chunk, ",") {
				if u, err = strconv.ParseUint(eid, 0, 16); err != nil {
					err = fmt.Errorf("failed to parse EventID: %w", err)
					return
				} else {
					p.Filter = append(p.Filter, uint16(u))
				}
			}
		case 3:
			if chunk == "" {
				break
			}

			// parsing MatchAnyKeyword
			if u, err = strconv.ParseUint(chunk, 0, 64); err != nil {
				err = fmt.Errorf("failed to parse MatchAnyKeyword: %w", err)
				return
			} else {
				p.MatchAnyKeyword = u
			}
		case 4:
			if chunk == "" {
				break
			}

			// parsing MatchAllKeyword
			if u, err = strconv.ParseUint(chunk, 0, 64); err != nil {
				err = fmt.Errorf("failed to parse MatchAllKeyword: %w", err)
				return
			} else {
				p.MatchAllKeyword = u
			}
		default:
			return
		}
	}
	return
}

// EnumerateProviders returns a ProviderMap containing available providers
// keys are both provider's GUIDs and provider's names
func EnumerateProviders() (m ProviderMap) {
	var buf *ProviderEnumerationInfo
	size := uint32(1)
	for {
		tmp := make([]byte, size)
		buf = (*ProviderEnumerationInfo)(unsafe.Pointer(&tmp[0]))
		if err := TdhEnumerateProviders(buf, &size); err != ERROR_INSUFFICIENT_BUFFER {
			break
		}
	}
	m = make(ProviderMap)
	startProvEnumInfo := uintptr(unsafe.Pointer(buf))
	it := uintptr(unsafe.Pointer(&buf.TraceProviderInfoArray[0]))
	for i := uintptr(0); i < uintptr(buf.NumberOfProviders); i++ {
		ptpi := (*TraceProviderInfo)(unsafe.Pointer(it + i*unsafe.Sizeof(buf.TraceProviderInfoArray[0])))
		guid := ptpi.ProviderGuid.String()
		name := UTF16AtOffsetToString(startProvEnumInfo, uintptr(ptpi.ProviderNameOffset))
		// We use a default provider here
		p := DefaultProvider
		p.GUID = guid
		p.Name = name
		m[name] = &p
		m[guid] = &p
	}
	return
}

// ResolveProvider return a Provider structure given a GUID or
// a provider name as input
func ResolveProvider(s string) (p Provider) {

	if providers == nil {
		providers = EnumerateProviders()
	}

	if g, err := ParseGUID(s); err == nil {
		s = g.String()
	}

	if prov, ok := providers[s]; ok {
		// search provider by name
		return *prov
	}

	return
}
