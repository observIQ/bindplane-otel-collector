//go:build windows
// +build windows

package etw

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	nullGUIDStr = "{00000000-0000-0000-0000-000000000000}"
)

var (
	nullGUID = GUID{}
)

/*
typedef struct _GUID {
	DWORD Data1;
	WORD Data2;
	WORD Data3;
	BYTE Data4[8];
} GUID;
*/

// GUID structure
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// IsZero checks if GUID is all zeros
func (g *GUID) IsZero() bool {
	return g.Equals(&nullGUID)
}

func (g *GUID) String() string {
	return fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
		g.Data1,
		g.Data2,
		g.Data3,
		g.Data4[0], g.Data4[1],
		g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

func (g *GUID) Equals(other *GUID) bool {
	return g.Data1 == other.Data1 &&
		g.Data2 == other.Data2 &&
		g.Data3 == other.Data3 &&
		g.Data4[0] == other.Data4[0] &&
		g.Data4[1] == other.Data4[1] &&
		g.Data4[2] == other.Data4[2] &&
		g.Data4[3] == other.Data4[3] &&
		g.Data4[4] == other.Data4[4] &&
		g.Data4[5] == other.Data4[5] &&
		g.Data4[6] == other.Data4[6] &&
		g.Data4[7] == other.Data4[7]
}

var (
	guidRE = regexp.MustCompile(`^\{?[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}\}?$`)
)

// MustParseGUIDFromString parses a guid string into a GUID struct or panics
func MustParseGUIDFromString(sguid string) (guid *GUID) {
	var err error
	if guid, err = ParseGUID(sguid); err != nil {
		panic(err)
	}
	return
}

// ParseGUID parses a guid string into a GUID structure
func ParseGUID(guid string) (g *GUID, err error) {
	var u uint64

	g = &GUID{}
	guid = strings.ToUpper(guid)
	if !guidRE.MatchString(guid) {
		return nil, fmt.Errorf("bad GUID format")
	}
	guid = strings.Trim(guid, "{}")
	sp := strings.Split(guid, "-")

	if u, err = strconv.ParseUint(sp[0], 16, 32); err != nil {
		return
	}
	g.Data1 = uint32(u)
	if u, err = strconv.ParseUint(sp[1], 16, 16); err != nil {
		return
	}
	g.Data2 = uint16(u)
	if u, err = strconv.ParseUint(sp[2], 16, 16); err != nil {
		return
	}
	g.Data3 = uint16(u)
	if u, err = strconv.ParseUint(sp[3], 16, 16); err != nil {
		return
	}
	g.Data4[0] = uint8(u >> 8)
	g.Data4[1] = uint8(u & 0xff)
	if u, err = strconv.ParseUint(sp[4], 16, 64); err != nil {
		return
	}
	g.Data4[2] = uint8((u >> 40))
	g.Data4[3] = uint8((u >> 32) & 0xff)
	g.Data4[4] = uint8((u >> 24) & 0xff)
	g.Data4[5] = uint8((u >> 16) & 0xff)
	g.Data4[6] = uint8((u >> 8) & 0xff)
	g.Data4[7] = uint8(u & 0xff)

	return
}
