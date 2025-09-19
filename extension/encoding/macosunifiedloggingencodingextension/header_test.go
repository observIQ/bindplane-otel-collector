// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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

func TestParseHeaderChunk(t *testing.T) {
	header, err := ParseHeaderChunk(headerTestData)
	require.NoError(t, err)
	require.Equal(t, uint32(0x1000), header.ChunkTag)
	require.Equal(t, uint32(0x11), header.ChunkSubTag)
	require.Equal(t, uint32(1), header.MachTimeNumerator)
	require.Equal(t, uint32(1), header.MachTimeDenominator)
	require.Equal(t, uint64(139417370585359), header.ContinuousTime)
	require.Equal(t, uint64(1645401904), header.UnknownTime)
	require.Equal(t, uint32(625355), header.Unknown)
	require.Equal(t, uint32(300), header.BiasMin)
	require.Equal(t, uint32(0), header.DaylightSavings)
	require.Equal(t, uint32(1), header.UnknownFlags)
	require.Equal(t, uint32(24832), header.SubChunkTag)
	require.Equal(t, uint32(8), header.SubChunkDataSize)
	require.Equal(t, uint64(450429435277318), header.SubChunkContinuousTime)
	require.Equal(t, uint32(24833), header.SubChunkTag2)
	require.Equal(t, uint32(56), header.SubChunkDataSize2)
	require.Equal(t, uint32(7), header.Unknown2)
	require.Equal(t, uint32(8), header.Unknown3)
	require.Equal(t, "21A559", header.BuildVersionString)
	require.Equal(t, "MacBookPro16,1", header.HardwareModelString)
	require.Equal(t, uint32(24834), header.SubChunkTag3)
	require.Equal(t, uint32(24), header.SubChunkDataSize3)
	require.Equal(t, "C320B8CE97FA4DA59F317D392E389CEA", header.BootUUID)
	require.Equal(t, uint32(85), header.LogdPID)
	require.Equal(t, uint32(0), header.LogdExitStatus)
	require.Equal(t, uint32(24835), header.SubChunkTag4)
	require.Equal(t, uint32(48), header.SubChunkDataSize4)
	require.Equal(t, "/var/db/timezone/zoneinfo/America/New_York", header.TimezonePath)
}

var headerTestData = []byte{
	0, 16, 0, 0, 17, 0, 0, 0, 208, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 15, 105,
	217, 162, 204, 126, 0, 0, 48, 215, 18, 98, 0, 0, 0, 0, 203, 138, 9, 0, 44, 1, 0, 0, 0,
	0, 0, 0, 1, 0, 0, 0, 0, 97, 0, 0, 8, 0, 0, 0, 6, 112, 124, 198, 169, 153, 1, 0, 1, 97,
	0, 0, 56, 0, 0, 0, 7, 0, 0, 0, 8, 0, 0, 0, 50, 49, 65, 53, 53, 57, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 77, 97, 99, 66, 111, 111, 107, 80, 114, 111, 49, 54, 44, 49, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 97, 0, 0, 24, 0, 0, 0, 195, 32, 184, 206, 151,
	250, 77, 165, 159, 49, 125, 57, 46, 56, 156, 234, 85, 0, 0, 0, 0, 0, 0, 0, 3, 97, 0, 0,
	48, 0, 0, 0, 47, 118, 97, 114, 47, 100, 98, 47, 116, 105, 109, 101, 122, 111, 110, 101,
	47, 122, 111, 110, 101, 105, 110, 102, 111, 47, 65, 109, 101, 114, 105, 99, 97, 47, 78,
	101, 119, 95, 89, 111, 114, 107, 0, 0, 0, 0, 0, 0,
}
