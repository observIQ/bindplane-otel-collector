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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/models"
)

func TestGetULHeaderData(t *testing.T) {
	unifiedLogData := &UnifiedLogData{}
	err := getULHeaderData(getULHeaderDataTestData, unifiedLogData)
	require.NoError(t, err)
	require.Equal(t, 1, len(unifiedLogData.HeaderData))
}

func TestGetULCatalogData(t *testing.T) {
	data := &UnifiedLogCatalogData{}
	err := getULCatalogData(getULCatalogDataTestData, data)
	require.NoError(t, err)
	require.Equal(t, uint32(0x600B), data.CatalogData.ChunkTag)
	require.Equal(t, uint32(17), data.CatalogData.ChunkSubtag)
	require.Equal(t, uint64(464), data.CatalogData.ChunkDataSize)
	require.Equal(t, uint16(32), data.CatalogData.CatalogSubsystemStringsOffset)
	require.Equal(t, uint16(96), data.CatalogData.CatalogProcessInfoEntriesOffset)
	require.Equal(t, uint16(1), data.CatalogData.NumberProcessInformationEntries)
	require.Equal(t, uint16(160), data.CatalogData.CatalogOffsetSubChunks)
	require.Equal(t, uint16(7), data.CatalogData.NumberSubChunks)
	require.Equal(t, []byte{0, 0, 0, 0, 0, 0}, data.CatalogData.Unknown)
	require.Equal(t, uint64(820223379547412), data.CatalogData.EarliestFirehoseTimestamp)
	require.Equal(t, []string{"2BEFD20C18EC3838814F2B4E5AF3BCEC", "3D05845F3F65358F9EBF2236E772AC01"}, data.CatalogData.CatalogUUIDs)
	require.Equal(t, []byte{
		99, 111, 109, 46, 97, 112, 112, 108, 101, 46, 83, 107, 121, 76, 105, 103, 104, 116,
		0, 112, 101, 114, 102, 111, 114, 109, 97, 110, 99, 101, 95, 105, 110, 115, 116,
		114, 117, 109, 101, 110, 116, 97, 116, 105, 111, 110, 0, 116, 114, 97, 99, 105,
		110, 103, 46, 115, 116, 97, 108, 108, 115, 0, 0, 0,
	}, data.CatalogData.CatalogSubsystemStrings)
	require.Equal(t, 1, len(data.CatalogData.ProcessInfoEntries))
	var processEntry *models.ProcessInfoEntry
	for _, entry := range data.CatalogData.ProcessInfoEntries {
		processEntry = entry
		break
	}
	require.NotNil(t, processEntry)
	require.Equal(t, "2BEFD20C18EC3838814F2B4E5AF3BCEC", processEntry.MainUUID)
	require.Equal(t, "3D05845F3F65358F9EBF2236E772AC01", processEntry.DSCUUID)
	require.Equal(t, 7, len(data.CatalogData.CatalogSubchunks))
}

var getULHeaderDataTestData = []byte{
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

var getULCatalogDataTestData = []byte{
	11, 96, 0, 0, 17, 0, 0, 0, 208, 1, 0, 0, 0, 0, 0, 0, 32, 0, 96, 0, 1, 0, 160, 0, 7, 0,
	0, 0, 0, 0, 0, 0, 20, 165, 44, 35, 253, 233, 2, 0, 43, 239, 210, 12, 24, 236, 56, 56,
	129, 79, 43, 78, 90, 243, 188, 236, 61, 5, 132, 95, 63, 101, 53, 143, 158, 191, 34, 54,
	231, 114, 172, 1, 99, 111, 109, 46, 97, 112, 112, 108, 101, 46, 83, 107, 121, 76, 105,
	103, 104, 116, 0, 112, 101, 114, 102, 111, 114, 109, 97, 110, 99, 101, 95, 105, 110,
	115, 116, 114, 117, 109, 101, 110, 116, 97, 116, 105, 111, 110, 0, 116, 114, 97, 99,
	105, 110, 103, 46, 115, 116, 97, 108, 108, 115, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 158,
	0, 0, 0, 0, 0, 0, 0, 55, 1, 0, 0, 158, 0, 0, 0, 88, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 87, 0, 0, 0, 19, 0, 78, 0, 0, 0, 47, 0, 0, 0, 0, 0,
	246, 113, 118, 43, 250, 233, 2, 0, 62, 195, 90, 26, 9, 234, 2, 0, 120, 255, 0, 0, 0, 1,
	0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 48, 89, 60, 28, 9, 234, 2, 0,
	99, 50, 207, 40, 18, 234, 2, 0, 112, 240, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0,
	0, 0, 0, 19, 0, 47, 0, 153, 6, 208, 41, 18, 234, 2, 0, 0, 214, 108, 78, 32, 234, 2, 0,
	0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 128, 0, 87,
	79, 32, 234, 2, 0, 137, 5, 2, 205, 41, 234, 2, 0, 88, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0,
	0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 185, 11, 2, 205, 41, 234, 2, 0, 172, 57, 107,
	20, 56, 234, 2, 0, 152, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19,
	0, 47, 0, 53, 172, 105, 21, 56, 234, 2, 0, 170, 167, 194, 43, 68, 234, 2, 0, 144, 255,
	0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 220, 202, 171, 57,
	68, 234, 2, 0, 119, 171, 170, 119, 76, 234, 2, 0, 240, 254, 0, 0, 0, 1, 0, 0, 1, 0, 0,
	0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0,
}
