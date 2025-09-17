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

package macosunifiedloggingencodingextension

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Base prefix for a SimpleDump Chunk up to and including UnknownNumberMessageStrings (sizes come after)
var baseSimpleDumpPrefix = []byte{
	4, 96, 0, 0, // ChunkTag
	5, 0, 0, 0, // ChunkSubTag
	219, 0, 0, 0, 0, 0, 0, 0, // ChunkDataSize
	1, 0, 0, 0, 0, 0, 0, 0, // FirstProcID
	6, 0, 0, 0, 0, 0, 0, 0, // SecondProcID
	45, 182, 196, 71, 133, 4, 0, 0, // ContinuousTime
	3, 234, 0, 0, 0, 0, 0, 0, // ThreadID
	118, 118, 1, 0, // UnknownOffset
	3, 0, // UnknownTTL
	8, 0, // UnknownType
	13, 207, 62, 139, 73, 35, 50, 62, 179, 229, 84, 115, 7, 207, 14, // SenderUUID
	172, 61, 5, 132, 95, 63, 101, 53, 143, 158, 191, 34, 54, 231, 114, 172, 1, // DSCSharedCacheUUID
	1, 0, 0, 0, // UnknownNumberMessageString
}

// Default subsystem and message byte sequences; tests may override per-case
var defaultSizeSubsystemString = []byte{79, 0, 0, 0} // Unknown SizeSubsystemString

var defaultSubsystemBytes = []byte{
	117, 115, 101, 114, 47, 53, 48, 49, 47, 99, 111, 109, 46, 97, 112, 112, 108, 101, 46, 109,
	100, 119, 111, 114, 107, 101, 114, 46, 115, 104, 97, 114, 101, 100, 46, 48, 66, 48, 48, 48,
	48, 48, 48, 45, 48, 48, 48, 48, 45, 48, 48, 48, 48, 45, 48, 48, 48, 48, 45, 48, 48, 48, 48,
	48, 48, 48, 48, 48, 48, 48, 48, 32, 91, 52, 50, 50, 57, 93, 0, // Subsystem
}

var defaultSizeMessageString = []byte{56, 0, 0, 0} // UnknownSizeMessageString

var defaultMessageBytes = []byte{
	115, 101, 114, 118, 105, 99, 101, 32, 101, 120, 105, 116, 101, 100, 58, 32, 100,
	105, 114, 116, 121, 32, 61, 32, 48, 44, 32, 115, 117, 112, 112, 111, 114, 116, 101,
	100, 32, 112, 114, 101, 115, 115, 117, 114, 101, 100, 45, 101, 120, 105, 116, 32, 61,
	32, 49, 0, 0, 0, 0, 0, 0, // MessageString
}

// BuildSimpleDumpTestData concatenates prefix, subsystem, and message slices and
// updates the little-endian size fields for subsystem and message within the prefix.
// Expected size field offsets (bytes): 96..99 = subsystem length, 100..103 = message length.
func BuildSimpleDumpTestData(prefix, subsystemSize, subsystem, messageSize, message []byte) []byte {
	out := make([]byte, len(prefix))
	copy(out, prefix)

	out = append(out, subsystemSize...)
	out = append(out, messageSize...)
	out = append(out, subsystem...)
	out = append(out, message...)

	return out
}

func validateBaseData(t *testing.T, chunk *SimpleDumpChunk) {
	require.Equal(t, uint32(24580), chunk.ChunkTag, "ChunkTag value incorrect")
	require.Equal(t, uint32(5), chunk.ChunkSubTag, "ChunkSubTag value incorrect")
	require.Equal(t, uint64(219), chunk.ChunkDataSize, "ChunkDataSize value incorrect")
	require.Equal(t, uint64(1), chunk.FirstProcID, "FirstProcID value incorrect")
	require.Equal(t, uint64(6), chunk.SecondProcID, "SecondProcID value incorrect")
	require.Equal(t, uint64(4970481235501), chunk.ContinuousTime, "ContinuousTime value incorrect")
	require.Equal(t, uint64(59907), chunk.ThreadID, "ThreadID value incorrect")
	require.Equal(t, uint32(95862), chunk.UnknownOffset, "UnknownOffset value incorrect")
	require.Equal(t, uint16(3), chunk.UnknownTTL, "UnknownTTL value incorrect")
	require.Equal(t, uint16(8), chunk.UnknownType, "UnknownType value incorrect")
	require.Equal(t, "0DCF3E8B4923323EB3E5547307CF0EAC", chunk.SenderUUID, "SenderUUID value incorrect")
	require.Equal(t, "3D05845F3F65358F9EBF2236E772AC01", chunk.DSCSharedCacheUUID, "DSCSharedCacheUUID value incorrect")
	require.Equal(t, uint32(1), chunk.UnknownNumberMessageStrings, "UnknownNumberMessageStrings value incorrect")
}

func TestParseSimpleDump(t *testing.T) {
	testCases := []struct {
		desc     string
		data     []byte
		validate func(*testing.T, *SimpleDumpChunk)
	}{
		{
			desc: "base",
			data: BuildSimpleDumpTestData(baseSimpleDumpPrefix, defaultSizeSubsystemString, defaultSubsystemBytes, defaultSizeMessageString, defaultMessageBytes),
			validate: func(t *testing.T, chunk *SimpleDumpChunk) {
				validateBaseData(t, chunk)
				require.Equal(t, uint32(79), chunk.UnknownSizeSubsystemString, "UnknownSizeSubsystemString value incorrect")
				require.Equal(t, uint32(56), chunk.UnknownSizeMessageString, "UnknownSizeMessageString value incorrect")
				require.Equal(t, "user/501/com.apple.mdworker.shared.0B000000-0000-0000-0000-000000000000 [4229]", chunk.Subsystem, "Subsystem value incorrect")
				require.Equal(t, "service exited: dirty = 0, supported pressured-exit = 1", chunk.MessageString, "MessageString value incorrect")
			},
		},
		{
			desc: "empty",
			data: BuildSimpleDumpTestData(baseSimpleDumpPrefix, []byte{0, 0, 0, 0}, []byte{}, defaultSizeMessageString, defaultMessageBytes),
			validate: func(t *testing.T, chunk *SimpleDumpChunk) {
				validateBaseData(t, chunk)
				require.Equal(t, uint32(0), chunk.UnknownSizeSubsystemString, "UnknownSizeSubsystemString value incorrect")
				require.Equal(t, uint32(56), chunk.UnknownSizeMessageString, "UnknownSizeMessageString value incorrect")
				require.Equal(t, "(null)", chunk.Subsystem, "Subsystem value incorrect")
				require.Equal(t, "service exited: dirty = 0, supported pressured-exit = 1", chunk.MessageString, "MessageString value incorrect")
			},
		},
		{
			desc: "empty message",
			data: BuildSimpleDumpTestData(baseSimpleDumpPrefix, defaultSizeSubsystemString, defaultSubsystemBytes, []byte{0, 0, 0, 0}, []byte{}),
			validate: func(t *testing.T, chunk *SimpleDumpChunk) {
				validateBaseData(t, chunk)
				require.Equal(t, uint32(79), chunk.UnknownSizeSubsystemString, "UnknownSizeSubsystemString value incorrect")
				require.Equal(t, uint32(0), chunk.UnknownSizeMessageString, "UnknownSizeMessageString value incorrect")
				require.Equal(t, "user/501/com.apple.mdworker.shared.0B000000-0000-0000-0000-000000000000 [4229]", chunk.Subsystem, "Subsystem value incorrect")
				require.Equal(t, "(null)", chunk.MessageString, "MessageString value incorrect")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			parsedChunk, _ := ParseSimpleDumpChunk(tc.data)
			tc.validate(t, &parsedChunk)
		})
	}
}
