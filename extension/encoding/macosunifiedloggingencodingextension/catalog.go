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
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/models"
)

const (
	// UUIDLength is the length in bytes of a catalog UUID (128 bits)
	UUIDLength = uint16(16)
	// LZ4Compression is the catalog compression algorithm identifier for LZ4
	LZ4Compression = uint32(256)
	// OffsetSize is the size in bytes of an offset unit in catalog structures
	OffsetSize = uint64(2)
)

var (
	// UnknownLength holds the variable length for the unknown field
	UnknownLength = []byte{6}
	// SubsystemSize holds the size of a subsystem entry length field
	SubsystemSize = []byte{6}
	// LoadAddressSize is the byte-length for load address fields before padding
	LoadAddressSize = []byte{6}
)

// GlobalCatalog stores catalog data for use by other chunk parsers
var GlobalCatalog *models.CatalogChunk

// ParseCatalogChunk parses a Catalog chunk (0x600b) containing catalog metadata
func ParseCatalogChunk(data []byte) (models.CatalogChunk, []byte, error) {
	var catalog models.CatalogChunk
	var preamble LogPreamble
	var err error
	originalData := data

	// Save original data for offset-based access
	preamble, data, _ = ParsePreamble(data)

	var catalogSubsystemStringsOffsetBytes, catalogProcessInfoEntriesOffsetBytes, numberProcessInformationEntriesBytes,
		catalogOffsetSubChunksBytes, numberSubChunksBytes, unknown, earliestFirehoseTimestampBytes []byte

	// Parse header fields
	data, catalogSubsystemStringsOffsetBytes, _ = helpers.Take(data, 2)
	catalogSubsystemStringsOffset := binary.LittleEndian.Uint16(catalogSubsystemStringsOffsetBytes)

	data, catalogProcessInfoEntriesOffsetBytes, _ = helpers.Take(data, 2)
	catalogProcessInfoEntriesOffset := binary.LittleEndian.Uint16(catalogProcessInfoEntriesOffsetBytes)

	data, numberProcessInformationEntriesBytes, _ = helpers.Take(data, 2)
	numberProcessInformationEntries := binary.LittleEndian.Uint16(numberProcessInformationEntriesBytes)

	data, catalogOffsetSubChunksBytes, _ = helpers.Take(data, 2)
	catalogOffsetSubChunks := binary.LittleEndian.Uint16(catalogOffsetSubChunksBytes)

	data, numberSubChunksBytes, _ = helpers.Take(data, 2)
	numberSubChunks := binary.LittleEndian.Uint16(numberSubChunksBytes)

	data, unknown, _ = helpers.Take(data, int(UnknownLength[0]))

	data, earliestFirehoseTimestampBytes, _ = helpers.Take(data, 8)
	earliestFirehoseTimestamp := binary.LittleEndian.Uint64(earliestFirehoseTimestampBytes)

	// Parse UUIDs
	numberCatalogUUIDs := catalogSubsystemStringsOffset / UUIDLength
	catalogUUIDs := make([]string, numberCatalogUUIDs)
	for i := 0; i < int(numberCatalogUUIDs); i++ {
		var uuidBytes []byte

		data, uuidBytes, err = helpers.Take(data, int(UUIDLength))
		if err != nil {
			return catalog, data, fmt.Errorf("failed to parse catalog UUID %d: %w", i, err)
		}

		// Convert 16 bytes to uppercase hex string (big-endian for consistency with firehose parsing)
		uuidStr := fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
			uuidBytes[0], uuidBytes[1], uuidBytes[2], uuidBytes[3],
			uuidBytes[4], uuidBytes[5], uuidBytes[6], uuidBytes[7],
			uuidBytes[8], uuidBytes[9], uuidBytes[10], uuidBytes[11],
			uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])
		catalogUUIDs[i] = uuidStr
	}

	var subsystemsStringsData []byte

	// Parse subsystem strings
	subsystemsStringsLength := int(catalogProcessInfoEntriesOffset - catalogSubsystemStringsOffset)
	data, subsystemsStringsData, err = helpers.Take(data, subsystemsStringsLength)
	if err != nil {
		return catalog, data, fmt.Errorf("failed to parse catalog subsystems strings: %w", err)
	}

	// Parse process entries
	catalogProcessInfoEntriesVec := make([]models.ProcessInfoEntry, numberProcessInformationEntries)
	for i := 0; i < int(numberProcessInformationEntries); i++ {
		var entry models.ProcessInfoEntry
		entry, data, err = parseCatalogProcessEntry(data, catalogUUIDs)
		if err != nil {
			return catalog, data, fmt.Errorf("failed to parse catalog process entry %d: %w", i, err)
		}
		catalogProcessInfoEntriesVec[i] = entry
	}

	catalogProcessInfoEntries := make(map[string]*models.ProcessInfoEntry)
	for i := range catalogProcessInfoEntriesVec {
		entry := &catalogProcessInfoEntriesVec[i]
		key := fmt.Sprintf("%d_%d", entry.FirstNumberProcID, entry.SecondNumberProcID)
		catalogProcessInfoEntries[key] = entry
	}

	// Calculate position to start parsing subchunks
	// The catalogOffsetSubChunks is relative to the start of chunk data (after preamble)
	// Add 24-byte padding to account for alignment between catalog header and subchunks
	preambleSize := 16 // LogPreamble size
	chunkDataStart := originalData[preambleSize:]
	paddingSize := 24
	data = chunkDataStart[catalogOffsetSubChunks+uint16(paddingSize):]

	var catalogSubchunks []models.CatalogSubchunk
	for i := 0; i < int(numberSubChunks); i++ {
		var subchunk models.CatalogSubchunk
		subchunk, data, err = parseCatalogSubchunk(data)
		if err != nil {
			return catalog, data, fmt.Errorf("failed to parse catalog subchunk %d: %w", i, err)
		}
		catalogSubchunks = append(catalogSubchunks, subchunk)
	}

	catalog.ChunkTag = preamble.ChunkTag
	catalog.ChunkSubtag = preamble.ChunkSubtag
	catalog.ChunkDataSize = preamble.ChunkDataSize
	catalog.CatalogSubsystemStringsOffset = catalogSubsystemStringsOffset
	catalog.CatalogProcessInfoEntriesOffset = catalogProcessInfoEntriesOffset
	catalog.NumberProcessInformationEntries = numberProcessInformationEntries
	catalog.CatalogOffsetSubChunks = catalogOffsetSubChunks
	catalog.NumberSubChunks = numberSubChunks
	catalog.Unknown = unknown
	catalog.EarliestFirehoseTimestamp = earliestFirehoseTimestamp
	catalog.CatalogUUIDs = catalogUUIDs
	catalog.CatalogSubsystemStrings = subsystemsStringsData
	catalog.ProcessInfoEntries = catalogProcessInfoEntries
	catalog.CatalogSubchunks = catalogSubchunks

	// Store this catalog globally for use by other parsers
	GlobalCatalog = &catalog
	return catalog, data, nil
}

// parseCatalogSubchunk parses the catalog subchunk
func parseCatalogSubchunk(data []byte) (models.CatalogSubchunk, []byte, error) {
	var subchunk models.CatalogSubchunk
	var start, end, uncompressedSize, compressionAlgorithm, numberIndex []byte
	var err error

	data, start, err = helpers.Take(data, 8)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk start: %w", err)
	}
	data, end, err = helpers.Take(data, 8)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk end: %w", err)
	}
	data, uncompressedSize, err = helpers.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk uncompressed size: %w", err)
	}
	data, compressionAlgorithm, err = helpers.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk compression algorithm: %w", err)
	}
	data, numberIndex, err = helpers.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk number index: %w", err)
	}

	if binary.LittleEndian.Uint32(compressionAlgorithm) != LZ4Compression {
		return subchunk, data, fmt.Errorf("unsupported compression algorithm: 0x%x (expected LZ4 0x%x)", compressionAlgorithm, LZ4Compression)
	}

	// Parse indexes
	indexes := make([]uint16, binary.LittleEndian.Uint32(numberIndex))
	for i := 0; i < int(binary.LittleEndian.Uint32(numberIndex)); i++ {
		var indexBytes []byte

		data, indexBytes, err = helpers.Take(data, 2)
		if err != nil {
			return subchunk, data, fmt.Errorf("failed to parse index %d: %w", i, err)
		}
		indexes[i] = binary.LittleEndian.Uint16(indexBytes)
	}

	// Parse number of string offsets
	var numberStringOffsetsBytes []byte

	data, numberStringOffsetsBytes, err = helpers.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse number of string offsets: %w", err)
	}
	numberStringOffsets := binary.LittleEndian.Uint32(numberStringOffsetsBytes)

	// Parse string offsets
	stringOffsets := make([]uint16, int(numberStringOffsets))
	for i := 0; i < int(numberStringOffsets); i++ {

		var offsetBytes []byte

		data, offsetBytes, err = helpers.Take(data, 2)
		if err != nil {
			return subchunk, data, fmt.Errorf("failed to parse string offset %d: %w", i, err)
		}
		stringOffsets[i] = binary.LittleEndian.Uint16(offsetBytes)
	}

	padding := helpers.AnticipatedPaddingSize(uint64(binary.LittleEndian.Uint32(numberIndex))+uint64(numberStringOffsets), OffsetSize, 8)
	if padding > uint64(^uint(0)>>1) {
		return subchunk, data, fmt.Errorf("u64 is bigger than system usize")
	}

	data, _, err = helpers.Take(data, int(padding))
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to consume padding bytes: %w", err)
	}

	subchunk.Start = binary.LittleEndian.Uint64(start)
	subchunk.End = binary.LittleEndian.Uint64(end)
	subchunk.UncompressedSize = binary.LittleEndian.Uint32(uncompressedSize)
	subchunk.CompressionAlgorithm = binary.LittleEndian.Uint32(compressionAlgorithm)
	subchunk.NumberIndex = binary.LittleEndian.Uint32(numberIndex)
	subchunk.Indexes = indexes
	subchunk.NumberStringOffsets = numberStringOffsets
	subchunk.StringOffsets = stringOffsets

	return subchunk, data, nil
}

// parseCatalogProcessEntry parses the process information entry
func parseCatalogProcessEntry(data []byte, catalogUUIDs []string) (models.ProcessInfoEntry, []byte, error) {
	var entry models.ProcessInfoEntry
	var index, unknown, catalogMainUUIDIndex, catalogDSCUUIDIndex, firstNumberProcID,
		secondNumberProcID, pid, effectiveUserID, unknown2, numberUUIDsEntries, unknown3, numberSubsystems, unknown4 []byte

	var err error

	data, index, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry index: %w", err)
	}
	data, unknown, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown: %w", err)
	}
	data, catalogMainUUIDIndex, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry catalog main UUID index: %w", err)
	}
	data, catalogDSCUUIDIndex, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry catalog DSC UUID index: %w", err)
	}
	data, firstNumberProcID, err = helpers.Take(data, 8)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry first number proc ID: %w", err)
	}
	data, secondNumberProcID, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry second number proc ID: %w", err)
	}
	data, pid, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry PID: %w", err)
	}
	data, effectiveUserID, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry effective user ID: %w", err)
	}
	data, unknown2, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown2: %w", err)
	}
	data, numberUUIDsEntries, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry number UUIDs entries: %w", err)
	}
	data, unknown3, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown3: %w", err)
	}

	var uuidInfoEntries []models.ProcessUUIDEntry
	for i := 0; i < int(binary.LittleEndian.Uint32(numberUUIDsEntries)); i++ {
		var UUIDEntry models.ProcessUUIDEntry
		UUIDEntry, data, err = parseProcessInfoUUIDEntry(data, catalogUUIDs)
		if err != nil {
			return entry, data, fmt.Errorf("failed to parse process info UUID entry %d: %w", i, err)
		}
		uuidInfoEntries = append(uuidInfoEntries, UUIDEntry)
	}

	data, numberSubsystems, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry number subsystems: %w", err)
	}
	data, unknown4, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown4: %w", err)
	}

	var subsystemEntries []models.ProcessInfoSubsystem
	for i := 0; i < int(binary.LittleEndian.Uint32(numberSubsystems)); i++ {
		var subsystemEntry models.ProcessInfoSubsystem
		subsystemEntry, data, err = parseProcessInfoSubsystem(data)
		if err != nil {
			return entry, data, fmt.Errorf("failed to parse process info subsystem %d: %w", i, err)
		}
		subsystemEntries = append(subsystemEntries, subsystemEntry)
	}

	var mainUUID string
	if int(binary.LittleEndian.Uint16(catalogMainUUIDIndex)) < len(catalogUUIDs) {
		mainUUID = catalogUUIDs[binary.LittleEndian.Uint16(catalogMainUUIDIndex)]
	} else {
		mainUUID = ""
	}

	var dscUUID string
	if int(binary.LittleEndian.Uint16(catalogDSCUUIDIndex)) < len(catalogUUIDs) {
		dscUUID = catalogUUIDs[binary.LittleEndian.Uint16(catalogDSCUUIDIndex)]
	} else {
		dscUUID = ""
	}

	entry.Index = binary.LittleEndian.Uint16(index)
	entry.Unknown = binary.LittleEndian.Uint16(unknown)
	entry.CatalogMainUUIDIndex = binary.LittleEndian.Uint16(catalogMainUUIDIndex)
	entry.CatalogDSCUUIDIndex = binary.LittleEndian.Uint16(catalogDSCUUIDIndex)
	entry.FirstNumberProcID = binary.LittleEndian.Uint64(firstNumberProcID)
	entry.SecondNumberProcID = binary.LittleEndian.Uint32(secondNumberProcID)
	entry.PID = binary.LittleEndian.Uint32(pid)
	entry.EffectiveUserID = binary.LittleEndian.Uint32(effectiveUserID)
	entry.Unknown2 = binary.LittleEndian.Uint32(unknown2)
	entry.NumberUUIDsEntries = binary.LittleEndian.Uint32(numberUUIDsEntries)
	entry.Unknown3 = binary.LittleEndian.Uint32(unknown3)
	entry.UUIDInfoEntries = uuidInfoEntries
	entry.NumberSubsystems = binary.LittleEndian.Uint32(numberSubsystems)
	entry.Unknown4 = binary.LittleEndian.Uint32(unknown4)
	entry.SubsystemEntries = subsystemEntries
	entry.MainUUID = mainUUID
	entry.DSCUUID = dscUUID

	// Calculate and skip padding bytes after subsystem entries
	const subsystemSize uint64 = 6
	padding := helpers.AnticipatedPaddingSize(uint64(entry.NumberSubsystems), subsystemSize, 8)
	if padding > uint64(^uint(0)>>1) {
		return entry, data, fmt.Errorf("padding size %d is bigger than system int", padding)
	}

	// Skip padding bytes
	if padding > 0 {
		data, _, err = helpers.Take(data, int(padding))
		if err != nil {
			return entry, data, fmt.Errorf("failed to skip padding bytes: %w", err)
		}
	}

	return entry, data, nil
}

func parseProcessInfoUUIDEntry(data []byte, catalogUUIDs []string) (models.ProcessUUIDEntry, []byte, error) {
	var entry models.ProcessUUIDEntry
	var size, unknown, catalogUUIDIndex, loadAddressBytes []byte
	var err error

	data, size, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info UUID entry size: %w", err)
	}
	data, unknown, err = helpers.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info UUID entry unknown: %w", err)
	}
	data, catalogUUIDIndex, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info UUID entry catalog UUID index: %w", err)
	}

	entry.Size = binary.LittleEndian.Uint32(size)
	entry.Unknown = binary.LittleEndian.Uint32(unknown)
	entry.CatalogUUIDIndex = binary.LittleEndian.Uint16(catalogUUIDIndex)

	data, loadAddressBytes, err = helpers.Take(data, int(LoadAddressSize[0]))
	if err != nil {
		return entry, data, fmt.Errorf("failed to read load address: %w", err)
	}

	// Pad 6 bytes to 8 bytes for u64 parsing
	// The remaining 2 bytes are already zero from make()
	loadAddressVec := make([]byte, 8)
	copy(loadAddressVec, loadAddressBytes)

	entry.LoadAddress = binary.LittleEndian.Uint64(loadAddressVec)
	uuidIndex := entry.CatalogUUIDIndex
	if int(uuidIndex) >= len(catalogUUIDs) {
		return entry, data, fmt.Errorf("catalog UUID index %d out of range (max %d)", uuidIndex, len(catalogUUIDs)-1)
	}
	entry.UUID = catalogUUIDs[uuidIndex]

	return entry, data, nil
}

func parseProcessInfoSubsystem(data []byte) (models.ProcessInfoSubsystem, []byte, error) {
	var entry models.ProcessInfoSubsystem
	var identifier, subsystemOffset, categoryOffset []byte
	var err error

	data, identifier, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info subsystem identifier: %w", err)
	}
	data, subsystemOffset, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info subsystem subsystem offset: %w", err)
	}
	data, categoryOffset, err = helpers.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info subsystem category offset: %w", err)
	}

	entry.Identifier = binary.LittleEndian.Uint16(identifier)
	entry.SubsystemOffset = binary.LittleEndian.Uint16(subsystemOffset)
	entry.CategoryOffset = binary.LittleEndian.Uint16(categoryOffset)

	return entry, data, nil
}
