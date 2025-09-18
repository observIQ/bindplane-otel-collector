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
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
)

const (
	UUID_LENGTH     = uint16(16)
	LZ4_COMPRESSION = uint32(256)
	OFFSET_SIZE     = uint64(2)
)

var (
	UNKNOWN_LENGTH    = []byte{6}
	SUBSYSTEM_SIZE    = []byte{6}
	LOAD_ADDRESS_SIZE = []byte{6}
)

// GlobalCatalog stores catalog data for use by other chunk parsers
var GlobalCatalog *types.CatalogChunk

// ParseCatalogChunk parses a Catalog chunk (0x600b) containing catalog metadata
func ParseCatalogChunk(originalData []byte) (types.CatalogChunk, []byte, error) {
	var catalog types.CatalogChunk
	var preamble LogPreamble

	// Save original data for offset-based access
	data := originalData
	preamble, data, _ = ParsePreamble(data)

	// Parse header fields
	remainingData, catalogSubsystemStringsOffsetBytes, _ := utils.Take(data, 2)
	data = remainingData
	catalogSubsystemStringsOffset := binary.LittleEndian.Uint16(catalogSubsystemStringsOffsetBytes)

	remainingData, catalogProcessInfoEntriesOffsetBytes, _ := utils.Take(data, 2)
	data = remainingData
	catalogProcessInfoEntriesOffset := binary.LittleEndian.Uint16(catalogProcessInfoEntriesOffsetBytes)

	remainingData, numberProcessInformationEntriesBytes, _ := utils.Take(data, 2)
	data = remainingData
	numberProcessInformationEntries := binary.LittleEndian.Uint16(numberProcessInformationEntriesBytes)

	remainingData, catalogOffsetSubChunksBytes, _ := utils.Take(data, 2)
	data = remainingData
	catalogOffsetSubChunks := binary.LittleEndian.Uint16(catalogOffsetSubChunksBytes)

	remainingData, numberSubChunksBytes, _ := utils.Take(data, 2)
	data = remainingData
	numberSubChunks := binary.LittleEndian.Uint16(numberSubChunksBytes)

	remainingData, unknown, _ := utils.Take(data, int(UNKNOWN_LENGTH[0]))
	data = remainingData

	remainingData, earliestFirehoseTimestampBytes, _ := utils.Take(data, 8)
	data = remainingData
	earliestFirehoseTimestamp := binary.LittleEndian.Uint64(earliestFirehoseTimestampBytes)

	// Parse UUIDs
	numberCatalogUUIDs := catalogSubsystemStringsOffset / UUID_LENGTH
	catalogUUIDs := make([]string, numberCatalogUUIDs)
	for i := 0; i < int(numberCatalogUUIDs); i++ {
		remainingData, uuidBytes, err := utils.Take(data, int(UUID_LENGTH))
		if err != nil {
			return catalog, data, fmt.Errorf("failed to parse catalog UUID %d: %w", i, err)
		}
		data = remainingData

		// Convert 16 bytes to uppercase hex string
		uuidStr := strings.ToUpper(hex.EncodeToString(uuidBytes))
		catalogUUIDs[i] = uuidStr
	}

	// Parse subsystem strings
	subsystemsStringsLength := int(catalogProcessInfoEntriesOffset - catalogSubsystemStringsOffset)
	remainingData, subsystemsStringsData, err := utils.Take(data, subsystemsStringsLength)
	if err != nil {
		return catalog, data, fmt.Errorf("failed to parse catalog subsystems strings: %w", err)
	}
	data = remainingData

	// Parse process entries
	catalogProcessInfoEntriesVec := make([]types.ProcessInfoEntry, numberProcessInformationEntries)
	for i := 0; i < int(numberProcessInformationEntries); i++ {
		var entry types.ProcessInfoEntry
		entry, data, err = parseCatalogProcessEntry(data, catalogUUIDs)
		if err != nil {
			return catalog, data, fmt.Errorf("failed to parse catalog process entry %d: %w", i, err)
		}
		catalogProcessInfoEntriesVec[i] = entry
	}

	catalogProcessInfoEntries := make(map[string]*types.ProcessInfoEntry)
	for _, entry := range catalogProcessInfoEntriesVec {
		key := fmt.Sprintf("%d_%d", entry.FirstNumberProcID, entry.SecondNumberProcID)
		catalogProcessInfoEntries[key] = &entry
	}

	// Calculate position to start parsing subchunks
	// The catalogOffsetSubChunks is relative to the start of chunk data (after preamble)
	// Add 24-byte padding to account for alignment between catalog header and subchunks
	preambleSize := 16 // LogPreamble size
	chunkDataStart := originalData[preambleSize:]
	paddingSize := 24
	subchunkData := chunkDataStart[catalogOffsetSubChunks+uint16(paddingSize):]

	var catalogSubchunks []types.CatalogSubchunk
	for i := 0; i < int(numberSubChunks); i++ {
		var subchunk types.CatalogSubchunk
		subchunk, subchunkData, err = parseCatalogSubchunk(subchunkData)
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
func parseCatalogSubchunk(data []byte) (types.CatalogSubchunk, []byte, error) {
	var subchunk types.CatalogSubchunk
	data, start, err := utils.Take(data, 8)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk start: %w", err)
	}
	data, end, err := utils.Take(data, 8)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk end: %w", err)
	}
	data, uncompressedSize, err := utils.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk uncompressed size: %w", err)
	}
	data, compressionAlgorithm, err := utils.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk compression algorithm: %w", err)
	}
	data, numberIndex, err := utils.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse catalog subchunk number index: %w", err)
	}

	if binary.LittleEndian.Uint32(compressionAlgorithm) != LZ4_COMPRESSION {
		return subchunk, data, fmt.Errorf("unsupported compression algorithm: 0x%x (expected LZ4 0x%x)", compressionAlgorithm, LZ4_COMPRESSION)
	}

	// Parse indexes
	indexes := make([]uint16, binary.LittleEndian.Uint32(numberIndex))
	for i := 0; i < int(binary.LittleEndian.Uint32(numberIndex)); i++ {
		remainingData, indexBytes, err := utils.Take(data, 2)
		if err != nil {
			return subchunk, data, fmt.Errorf("failed to parse index %d: %w", i, err)
		}
		data = remainingData
		indexes[i] = binary.LittleEndian.Uint16(indexBytes)
	}

	// Parse number of string offsets
	remainingData2, numberStringOffsetsBytes, err := utils.Take(data, 4)
	if err != nil {
		return subchunk, data, fmt.Errorf("failed to parse number of string offsets: %w", err)
	}
	data = remainingData2
	numberStringOffsets := binary.LittleEndian.Uint32(numberStringOffsetsBytes)

	// Parse string offsets
	stringOffsets := make([]uint16, int(numberStringOffsets))
	for i := 0; i < int(numberStringOffsets); i++ {
		remainingData, offsetBytes, err := utils.Take(data, 2)
		if err != nil {
			return subchunk, data, fmt.Errorf("failed to parse string offset %d: %w", i, err)
		}
		data = remainingData
		stringOffsets[i] = binary.LittleEndian.Uint16(offsetBytes)
	}

	padding := utils.AnticipatedPaddingSize(uint64(binary.LittleEndian.Uint32(numberIndex))+uint64(numberStringOffsets), OFFSET_SIZE, 8)
	if padding > uint64(^uint(0)>>1) {
		return subchunk, data, fmt.Errorf("u64 is bigger than system usize")
	}

	data, _, err = utils.Take(data, int(padding))
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
func parseCatalogProcessEntry(data []byte, catalogUUIDs []string) (types.ProcessInfoEntry, []byte, error) {
	var entry types.ProcessInfoEntry
	remainingData, index, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry index: %w", err)
	}
	data = remainingData
	remainingData, unknown, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown: %w", err)
	}
	data = remainingData
	remainingData, catalogMainUUIDIndex, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry catalog main UUID index: %w", err)
	}
	data = remainingData
	remainingData, catalogDSCUUIDIndex, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry catalog DSC UUID index: %w", err)
	}
	data = remainingData
	remainingData, firstNumberProcID, err := utils.Take(data, 8)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry first number proc ID: %w", err)
	}
	data = remainingData
	remainingData, secondNumberProcID, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry second number proc ID: %w", err)
	}
	data = remainingData
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry second number proc ID: %w", err)
	}
	remainingData, pid, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry PID: %w", err)
	}
	data = remainingData
	remainingData, effectiveUserID, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry effective user ID: %w", err)
	}
	data = remainingData
	remainingData, unknown2, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown2: %w", err)
	}
	data = remainingData
	remainingData, numberUUIDsEntries, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry number UUIDs entries: %w", err)
	}
	data = remainingData
	remainingData, unknown3, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown3: %w", err)
	}
	data = remainingData

	var uuidInfoEntries []types.ProcessUUIDEntry
	for i := 0; i < int(binary.LittleEndian.Uint32(numberUUIDsEntries)); i++ {
		var UUIDEntry types.ProcessUUIDEntry
		UUIDEntry, data, err = parseProcessInfoUUIDEntry(data, catalogUUIDs)
		if err != nil {
			return entry, data, fmt.Errorf("failed to parse process info UUID entry %d: %w", i, err)
		}
		uuidInfoEntries = append(uuidInfoEntries, UUIDEntry)
	}

	remainingData, numberSubsystems, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry number subsystems: %w", err)
	}
	data = remainingData
	remainingData, unknown4, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse catalog process entry unknown4: %w", err)
	}
	data = remainingData

	var subsystemEntries []types.ProcessInfoSubsystem
	for i := 0; i < int(binary.LittleEndian.Uint32(numberSubsystems)); i++ {
		var subsystemEntry types.ProcessInfoSubsystem
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
		// log.Warn("[macos-unifiedlogs] Could not find main UUID in catalog")
		mainUUID = ""
	}

	var dscUUID string
	if int(binary.LittleEndian.Uint16(catalogDSCUUIDIndex)) < len(catalogUUIDs) {
		dscUUID = catalogUUIDs[binary.LittleEndian.Uint16(catalogDSCUUIDIndex)]
	} else {
		// log.Warn("[macos-unifiedlogs] Could not find DSC UUID in catalog")
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

	return entry, data, nil
}

func parseProcessInfoUUIDEntry(data []byte, catalogUUIDs []string) (types.ProcessUUIDEntry, []byte, error) {
	var entry types.ProcessUUIDEntry
	data, size, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info UUID entry size: %w", err)
	}
	data, unknown, err := utils.Take(data, 4)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info UUID entry unknown: %w", err)
	}
	data, catalogUUIDIndex, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info UUID entry catalog UUID index: %w", err)
	}

	entry.Size = binary.LittleEndian.Uint32(size)
	entry.Unknown = binary.LittleEndian.Uint32(unknown)
	entry.CatalogUUIDIndex = binary.LittleEndian.Uint16(catalogUUIDIndex)

	data, loadAddressBytes, err := utils.Take(data, int(LOAD_ADDRESS_SIZE[0]))
	if err != nil {
		return entry, data, fmt.Errorf("failed to read load address: %w", err)
	}

	// Pad 6 bytes to 8 bytes for u64 parsing
	// The remaining 2 bytes are already zero from make()
	loadAddressVec := make([]byte, 8)
	copy(loadAddressVec, loadAddressBytes)

	entry.LoadAddress = binary.LittleEndian.Uint64(loadAddressVec)
	entry.UUID = catalogUUIDs[binary.LittleEndian.Uint16(catalogUUIDIndex)]

	return entry, data, nil
}

func parseProcessInfoSubsystem(data []byte) (types.ProcessInfoSubsystem, []byte, error) {
	var entry types.ProcessInfoSubsystem
	data, identifier, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info subsystem identifier: %w", err)
	}
	data, subsystemOffset, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info subsystem subsystem offset: %w", err)
	}
	data, categoryOffset, err := utils.Take(data, 2)
	if err != nil {
		return entry, data, fmt.Errorf("failed to parse process info subsystem category offset: %w", err)
	}

	entry.Identifier = binary.LittleEndian.Uint16(identifier)
	entry.SubsystemOffset = binary.LittleEndian.Uint16(subsystemOffset)
	entry.CategoryOffset = binary.LittleEndian.Uint16(categoryOffset)

	return entry, data, nil
}
