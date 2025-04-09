// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows
// +build windows

package windows

import (
	"fmt"
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
