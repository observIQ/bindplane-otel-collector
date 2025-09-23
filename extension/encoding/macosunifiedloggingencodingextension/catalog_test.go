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
// See the for the specific language governing permissions and
// limitations under the License.

package macosunifiedloggingencodingextension

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCatalogChunk(t *testing.T) {
	catalog, _, err := ParseCatalogChunk(catalogTestData)
	require.NoError(t, err)
	require.Equal(t, uint32(0x600b), catalog.ChunkTag)
	require.Equal(t, uint32(17), catalog.ChunkSubtag)
	require.Equal(t, uint64(464), catalog.ChunkDataSize)
	require.Equal(t, uint16(32), catalog.CatalogSubsystemStringsOffset)
	require.Equal(t, uint16(96), catalog.CatalogProcessInfoEntriesOffset)
	require.Equal(t, uint16(1), catalog.NumberProcessInformationEntries)
	require.Equal(t, uint16(160), catalog.CatalogOffsetSubChunks)
	require.Equal(t, uint16(7), catalog.NumberSubChunks)
	require.Equal(t, []byte{0, 0, 0, 0, 0, 0}, catalog.Unknown)
	require.Equal(t, uint64(820223379547412), catalog.EarliestFirehoseTimestamp)
	require.Equal(t, []string{
		"2BEFD20C18EC3838814F2B4E5AF3BCEC",
		"3D05845F3F65358F9EBF2236E772AC01",
	}, catalog.CatalogUUIDs)
	require.Equal(t, []byte{
		99, 111, 109, 46, 97, 112, 112, 108, 101, 46, 83, 107, 121, 76, 105, 103, 104, 116,
		0, 112, 101, 114, 102, 111, 114, 109, 97, 110, 99, 101, 95, 105, 110, 115, 116,
		114, 117, 109, 101, 110, 116, 97, 116, 105, 111, 110, 0, 116, 114, 97, 99, 105,
		110, 103, 46, 115, 116, 97, 108, 108, 115, 0, 0, 0,
	}, catalog.CatalogSubsystemStrings)

	require.Equal(t, 1, len(catalog.ProcessInfoEntries))
	require.Equal(t, "2BEFD20C18EC3838814F2B4E5AF3BCEC", catalog.ProcessInfoEntries["158_311"].MainUUID)
	require.Equal(t, "3D05845F3F65358F9EBF2236E772AC01", catalog.ProcessInfoEntries["158_311"].DSCUUID)
	require.Equal(t, 7, len(catalog.CatalogSubchunks))
}

func TestParseCatalogSubchunk(t *testing.T) {
	subchunk, data, err := parseCatalogSubchunk(catalogSubchunkTestData)
	require.NoError(t, err)
	require.Equal(t, uint64(820210633699830), subchunk.Start)
	require.Equal(t, uint64(820274771182398), subchunk.End)
	require.Equal(t, uint32(65400), subchunk.UncompressedSize)
	require.Equal(t, uint32(256), subchunk.CompressionAlgorithm)
	require.Equal(t, uint32(1), subchunk.NumberIndex)
	require.Equal(t, []uint16{0}, subchunk.Indexes)
	require.Equal(t, uint32(3), subchunk.NumberStringOffsets)
	require.Equal(t, []uint16{0, 19, 47}, subchunk.StringOffsets)

	subchunk, data, err = parseCatalogSubchunk(data)
	require.NoError(t, err)
	require.Equal(t, uint64(820274802743600), subchunk.Start)
	require.Equal(t, uint64(820313668399715), subchunk.End)
	require.Equal(t, uint32(61552), subchunk.UncompressedSize)
	require.Equal(t, uint32(256), subchunk.CompressionAlgorithm)
	require.Equal(t, uint32(1), subchunk.NumberIndex)
	require.Equal(t, []uint16{0}, subchunk.Indexes)
	require.Equal(t, uint32(3), subchunk.NumberStringOffsets)
	require.Equal(t, []uint16{0, 19, 47}, subchunk.StringOffsets)

	subchunk, _, err = parseCatalogSubchunk(data)
	require.NoError(t, err)
	require.Equal(t, uint64(820313685231257), subchunk.Start)
	require.Equal(t, uint64(820374429029888), subchunk.End)
	require.Equal(t, uint32(65536), subchunk.UncompressedSize)
	require.Equal(t, uint32(256), subchunk.CompressionAlgorithm)
	require.Equal(t, uint32(1), subchunk.NumberIndex)
	require.Equal(t, []uint16{0}, subchunk.Indexes)
	require.Equal(t, uint32(3), subchunk.NumberStringOffsets)
	require.Equal(t, []uint16{0, 19, 47}, subchunk.StringOffsets)
}

func TestParseCatalogProcessEntry(t *testing.T) {
	entry, _, err := parseCatalogProcessEntry(catalogProcessEntryTestData, []string{"MAIN", "DSC", "OTHER"})
	require.NoError(t, err)
	require.Equal(t, uint16(0), entry.Index)
	require.Equal(t, uint16(0), entry.Unknown)
	require.Equal(t, uint16(0), entry.CatalogMainUUIDIndex)
	require.Equal(t, uint16(1), entry.CatalogDSCUUIDIndex)
	require.Equal(t, uint64(158), entry.FirstNumberProcID)
	require.Equal(t, uint32(311), entry.SecondNumberProcID)
	require.Equal(t, uint32(158), entry.PID)
	require.Equal(t, uint32(88), entry.EffectiveUserID)
	require.Equal(t, uint32(0), entry.Unknown2)
	require.Equal(t, uint32(0), entry.NumberUUIDsEntries)
	require.Equal(t, uint32(0), entry.Unknown3)
	require.Equal(t, 0, len(entry.UUIDInfoEntries))
	require.Equal(t, uint32(2), entry.NumberSubsystems)
	require.Equal(t, uint32(0), entry.Unknown4)
	require.Equal(t, 2, len(entry.SubsystemEntries))
	require.Equal(t, "MAIN", entry.MainUUID)
	require.Equal(t, "DSC", entry.DSCUUID)
}

func TestParseProcessInfoSubsystem(t *testing.T) {
	entry, data, err := parseProcessInfoSubsystem(processInfoSubsystemTestData)
	require.NoError(t, err)
	require.Equal(t, uint16(87), entry.Identifier)
	require.Equal(t, uint16(0), entry.SubsystemOffset)
	require.Equal(t, uint16(19), entry.CategoryOffset)

	entry, _, err = parseProcessInfoSubsystem(data)
	require.NoError(t, err)
	require.Equal(t, uint16(78), entry.Identifier)
	require.Equal(t, uint16(0), entry.SubsystemOffset)
	require.Equal(t, uint16(47), entry.CategoryOffset)
}

var processInfoSubsystemTestData = []byte{
	87, 0, 0, 0, 19, 0, 78, 0, 0, 0, 47, 0, 0, 0, 0, 0, 246, 113, 118, 43, 250, 233, 2, 0,
	62, 195, 90, 26, 9, 234, 2, 0, 120, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0,
	0, 0, 0, 19, 0, 47, 0, 48, 89, 60, 28, 9, 234, 2, 0, 99, 50, 207, 40, 18, 234, 2, 0,
	112, 240, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 153, 6,
	208, 41, 18, 234, 2, 0, 0, 214, 108, 78, 32, 234, 2, 0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0,
	0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 128, 0, 87, 79, 32, 234, 2, 0, 137, 5, 2,
	205, 41, 234, 2, 0, 88, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19,
	0, 47, 0, 185, 11, 2, 205, 41, 234, 2, 0, 172, 57, 107, 20, 56, 234, 2, 0, 152, 255, 0,
	0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 53, 172, 105, 21, 56,
	234, 2, 0, 170, 167, 194, 43, 68, 234, 2, 0, 144, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0,
	0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 220, 202, 171, 57, 68, 234, 2, 0, 119, 171, 170,
	119, 76, 234, 2, 0, 240, 254, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19,
	0, 47, 0,
}

var catalogProcessEntryTestData = []byte{
	0, 0, 0, 0, 0, 0, 1, 0, 158, 0, 0, 0, 0, 0, 0, 0, 55, 1, 0, 0, 158, 0, 0, 0, 88, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 87, 0, 0, 0, 19, 0, 78,
	0, 0, 0, 47, 0, 0, 0, 0, 0, 246, 113, 118, 43, 250, 233, 2, 0, 62, 195, 90, 26, 9, 234,
	2, 0, 120, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 48,
	89, 60, 28, 9, 234, 2, 0, 99, 50, 207, 40, 18, 234, 2, 0, 112, 240, 0, 0, 0, 1, 0, 0,
	1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 153, 6, 208, 41, 18, 234, 2, 0, 0,
	214, 108, 78, 32, 234, 2, 0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0,
	0, 19, 0, 47, 0, 128, 0, 87, 79, 32, 234, 2, 0, 137, 5, 2, 205, 41, 234, 2, 0, 88, 255,
	0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 185, 11, 2, 205,
	41, 234, 2, 0, 172, 57, 107, 20, 56, 234, 2, 0, 152, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0,
	0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0, 53, 172, 105, 21, 56, 234, 2, 0, 170, 167, 194,
	43, 68, 234, 2, 0, 144, 255, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19,
	0, 47, 0, 220, 202, 171, 57, 68, 234, 2, 0, 119, 171, 170, 119, 76, 234, 2, 0, 240,
	254, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 19, 0, 47, 0,
}

var catalogSubchunkTestData = []byte{
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

var catalogTestData = []byte{
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
