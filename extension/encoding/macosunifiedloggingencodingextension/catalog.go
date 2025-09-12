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
	"strings"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
)

// GlobalCatalog stores catalog data for use by other chunk parsers
var GlobalCatalog *types.CatalogChunk

// ParseCatalogChunk parses a Catalog chunk (0x600b) containing catalog metadata
// Based on the rust implementation in catalog.rs
func ParseCatalogChunk(data []byte, entry *TraceV3Entry) {
	if len(data) < 56 { // Minimum size: 16-byte header + 40-byte catalog header
		entry.Message = fmt.Sprintf("Catalog chunk too small: %d bytes (need at least 56)", len(data))
		return
	}

	var catalog types.CatalogChunk
	offset := 0

	// Parse chunk header (16 bytes total)
	catalog.ChunkTag = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	catalog.ChunkSubtag = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	catalog.ChunkDataSize = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Validate chunk tag
	if catalog.ChunkTag != 0x600b {
		entry.Message = fmt.Sprintf("Invalid catalog chunk tag: expected 0x600b, got 0x%x", catalog.ChunkTag)
		return
	}

	// Parse catalog header fields (40 bytes)
	if len(data) < offset+40 {
		entry.Message = fmt.Sprintf("Catalog chunk too small for header: %d bytes", len(data))
		return
	}

	catalog.CatalogSubsystemStringsOffset = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	catalog.CatalogProcessInfoEntriesOffset = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	catalog.NumberProcessInformationEntries = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	catalog.CatalogOffsetSubChunks = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	catalog.NumberSubChunks = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	// Unknown/padding bytes (6 bytes)
	catalog.Unknown = make([]byte, 6)
	copy(catalog.Unknown, data[offset:offset+6])
	offset += 6

	catalog.EarliestFirehoseTimestamp = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse UUIDs (16 bytes each)
	const uuidLength = 16
	numberCatalogUUIDs := int(catalog.CatalogSubsystemStringsOffset) / uuidLength

	if numberCatalogUUIDs > 0 {
		if len(data) < offset+numberCatalogUUIDs*uuidLength {
			entry.Message = fmt.Sprintf("Catalog chunk too small for UUIDs: %d bytes", len(data))
			return
		}

		catalog.CatalogUUIDs = make([]string, numberCatalogUUIDs)
		for i := 0; i < numberCatalogUUIDs; i++ {
			uuidBytes := data[offset : offset+uuidLength]
			// Parse as big-endian uint128 and format as uppercase hex (matching rust)
			catalog.CatalogUUIDs[i] = fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
				uuidBytes[0], uuidBytes[1], uuidBytes[2], uuidBytes[3],
				uuidBytes[4], uuidBytes[5], uuidBytes[6], uuidBytes[7],
				uuidBytes[8], uuidBytes[9], uuidBytes[10], uuidBytes[11],
				uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])
			offset += uuidLength
		}
	}

	// Parse subsystem strings
	subsystemStringsLength := int(catalog.CatalogProcessInfoEntriesOffset - catalog.CatalogSubsystemStringsOffset)
	if subsystemStringsLength > 0 && len(data) >= offset+subsystemStringsLength {
		catalog.CatalogSubsystemStrings = make([]byte, subsystemStringsLength)
		copy(catalog.CatalogSubsystemStrings, data[offset:offset+subsystemStringsLength])
		offset += subsystemStringsLength
	}

	// Parse process info entries with complete subsystem parsing
	catalog.ProcessInfoEntries = make([]types.ProcessInfoEntry, 0, catalog.NumberProcessInformationEntries)

	// Parse all process entries to build complete subsystem mapping
	for i := 0; i < int(catalog.NumberProcessInformationEntries); i++ {
		if len(data) < offset+44 { // Minimum entry size
			break
		}

		var procEntry types.ProcessInfoEntry

		// Parse basic process info fields (44 bytes)
		procEntry.Index = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
		procEntry.Unknown = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
		procEntry.CatalogMainUUIDIndex = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
		procEntry.CatalogDSCUUIDIndex = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
		procEntry.FirstNumberProcID = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8
		procEntry.SecondNumberProcID = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		procEntry.PID = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		procEntry.EffectiveUserID = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		procEntry.Unknown2 = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		procEntry.NumberUUIDsEntries = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		procEntry.Unknown3 = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Parse UUID info entries
		procEntry.UUIDInfoEntries = make([]types.ProcessUUIDEntry, procEntry.NumberUUIDsEntries)
		for j := 0; j < int(procEntry.NumberUUIDsEntries); j++ {
			if len(data) < offset+20 { // UUID entry size: 4+4+2+8+2=20 bytes
				break
			}

			var uuidEntry types.ProcessUUIDEntry
			uuidEntry.Size = binary.LittleEndian.Uint32(data[offset : offset+4])
			offset += 4
			uuidEntry.Unknown = binary.LittleEndian.Uint32(data[offset : offset+4])
			offset += 4
			uuidEntry.CatalogUUIDIndex = binary.LittleEndian.Uint16(data[offset : offset+2])
			offset += 2
			uuidEntry.LoadAddress = binary.LittleEndian.Uint64(data[offset : offset+8])
			offset += 8

			// Get UUID from catalog array
			if int(uuidEntry.CatalogUUIDIndex) < len(catalog.CatalogUUIDs) {
				uuidEntry.UUID = catalog.CatalogUUIDs[uuidEntry.CatalogUUIDIndex]
			}

			procEntry.UUIDInfoEntries[j] = uuidEntry
		}

		// Parse subsystem count and unknown field
		if len(data) < offset+8 {
			break
		}
		procEntry.NumberSubsystems = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		procEntry.Unknown4 = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Parse subsystem entries - this is critical for proper subsystem/category mapping
		procEntry.SubsystemEntries = make([]types.ProcessInfoSubsystem, procEntry.NumberSubsystems)
		for j := 0; j < int(procEntry.NumberSubsystems); j++ {
			if len(data) < offset+6 { // Subsystem entry size: 2+2+2=6 bytes
				break
			}

			var subsysEntry types.ProcessInfoSubsystem
			subsysEntry.Identifier = binary.LittleEndian.Uint16(data[offset : offset+2])
			offset += 2
			subsysEntry.SubsystemOffset = binary.LittleEndian.Uint16(data[offset : offset+2])
			offset += 2
			subsysEntry.CategoryOffset = binary.LittleEndian.Uint16(data[offset : offset+2])
			offset += 2

			procEntry.SubsystemEntries[j] = subsysEntry
		}

		// Get UUIDs from catalog array
		if int(procEntry.CatalogMainUUIDIndex) < len(catalog.CatalogUUIDs) {
			procEntry.MainUUID = catalog.CatalogUUIDs[procEntry.CatalogMainUUIDIndex]
		}
		if int(procEntry.CatalogDSCUUIDIndex) < len(catalog.CatalogUUIDs) {
			procEntry.DSCUUID = catalog.CatalogUUIDs[procEntry.CatalogDSCUUIDIndex]
		}

		catalog.ProcessInfoEntries = append(catalog.ProcessInfoEntries, procEntry)

		// Limit to reasonable number for performance and to avoid excessive parsing
		if i >= 50 {
			break
		}
	}

	// Extract some subsystem strings for display
	var subsystemStrings []string
	if len(catalog.CatalogSubsystemStrings) > 0 {
		// Split on null bytes and extract first few strings
		parts := strings.Split(string(catalog.CatalogSubsystemStrings), "\x00")
		for _, part := range parts {
			if strings.TrimSpace(part) != "" {
				subsystemStrings = append(subsystemStrings, strings.TrimSpace(part))
				if len(subsystemStrings) >= 3 { // Limit to first 3
					break
				}
			}
		}
	}

	// Update entry with parsed catalog information
	entry.ProcessID = uint32(catalog.EarliestFirehoseTimestamp & 0xFFFFFFFF) // Use timestamp as pseudo process ID
	entry.ThreadID = uint64(catalog.NumberProcessInformationEntries)
	entry.Level = "Debug"
	entry.MessageType = "Default"
	entry.EventType = "logEvent"

	// Create detailed message about catalog contents
	message := fmt.Sprintf("Catalog: %d UUIDs, %d processes, %d subchunks, earliest_time=%d",
		len(catalog.CatalogUUIDs), catalog.NumberProcessInformationEntries, catalog.NumberSubChunks,
		catalog.EarliestFirehoseTimestamp)

	if len(subsystemStrings) > 0 {
		message += fmt.Sprintf(", subsystems=[%s]", strings.Join(subsystemStrings, ", "))
	}

	if len(catalog.ProcessInfoEntries) > 0 {
		proc := catalog.ProcessInfoEntries[0]
		message += fmt.Sprintf(", sample_proc={pid=%d,uid=%d,main_uuid=%s}",
			proc.PID, proc.EffectiveUserID, proc.MainUUID[:8]+"...")
	}

	entry.Message = message

	// Parse catalog subchunks if we have offset information
	if catalog.CatalogOffsetSubChunks > 0 && len(data) > int(56+catalog.CatalogOffsetSubChunks) {
		subchunksOffset := int(56 + catalog.CatalogOffsetSubChunks)
		subchunksData := data[subchunksOffset:]
		catalog.CatalogSubchunks = parseCatalogSubchunks(subchunksData, catalog.NumberSubChunks)
	}

	// Store this catalog globally for use by other parsers
	GlobalCatalog = &catalog
}

// parseCatalogSubchunks parses the subchunk metadata used for decompression
// Based on the rust implementation's parse_catalog_subchunk method
func parseCatalogSubchunks(data []byte, numberSubChunks uint16) []types.CatalogSubchunk {
	subchunks := make([]types.CatalogSubchunk, 0, numberSubChunks)
	offset := 0

	for i := 0; i < int(numberSubChunks) && i < 10; i++ { // Limit for performance
		if len(data) < offset+32 { // Minimum subchunk size
			break
		}

		var subchunk types.CatalogSubchunk

		// Parse basic subchunk fields (32 bytes)
		subchunk.Start = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8
		subchunk.End = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8
		subchunk.UncompressedSize = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		subchunk.CompressionAlgorithm = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		subchunk.NumberIndex = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Validate compression algorithm (should be LZ4 = 0x100)
		const lz4Compression = 0x100
		if subchunk.CompressionAlgorithm != lz4Compression {
			// Skip invalid subchunk
			offset += 4 // Skip remaining field
			continue
		}

		// Parse indexes
		if len(data) >= offset+int(subchunk.NumberIndex)*2 {
			subchunk.Indexes = make([]uint16, subchunk.NumberIndex)
			for j := 0; j < int(subchunk.NumberIndex); j++ {
				subchunk.Indexes[j] = binary.LittleEndian.Uint16(data[offset : offset+2])
				offset += 2
			}
		}

		// Parse number of string offsets
		if len(data) >= offset+4 {
			subchunk.NumberStringOffsets = binary.LittleEndian.Uint32(data[offset : offset+4])
			offset += 4

			// Parse string offsets
			if len(data) >= offset+int(subchunk.NumberStringOffsets)*2 {
				subchunk.StringOffsets = make([]uint16, subchunk.NumberStringOffsets)
				for j := 0; j < int(subchunk.NumberStringOffsets); j++ {
					subchunk.StringOffsets[j] = binary.LittleEndian.Uint16(data[offset : offset+2])
					offset += 2
				}
			}
		}

		// Calculate 8-byte alignment padding (matching rust implementation)
		totalItems := subchunk.NumberIndex + subchunk.NumberStringOffsets
		const offsetSize = 2 // Each offset is 2 bytes
		padding := (8 - ((totalItems * offsetSize) & 7)) & 7
		offset += int(padding)

		subchunks = append(subchunks, subchunk)
	}

	return subchunks
}
