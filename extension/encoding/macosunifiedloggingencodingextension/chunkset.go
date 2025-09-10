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
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
	"github.com/pierrec/lz4/v4"
)

// ParseChunksetChunk parses a Chunkset chunk (0x600d) containing compressed log data
// Based on the Rust implementation from mandiant/macos-UnifiedLogs
// Enhanced with subchunk metadata for intelligent decompression
func ParseChunksetChunk(data []byte, entry *TraceV3Entry, header *TraceV3Header, timesyncData map[string]*TimesyncBoot) []*TraceV3Entry {
	if len(data) < 32 { // Need at least 32 bytes for chunkset header
		entry.Message = fmt.Sprintf("Chunkset chunk too small: %d bytes", len(data))
		return []*TraceV3Entry{entry}
	}

	// Parse chunkset header structure (similar to rust parse_chunkset)
	offset := 16 // Skip preamble which was already parsed
	signature := binary.LittleEndian.Uint32(data[offset:])
	uncompressSize := binary.LittleEndian.Uint32(data[offset+4:])
	blockSize := binary.LittleEndian.Uint32(data[offset+8:])

	offset += 12

	// Validate chunkset signature using enhanced validation
	isCompressed, isValid, err := ValidateChunksetSignature(signature)
	if !isValid {
		entry.Message = fmt.Sprintf("Invalid chunkset signature: %v", err)
		return []*TraceV3Entry{entry}
	}

	var decompressedData []byte
	var decompressionInfo *SubchunkDecompressionInfo

	if !isCompressed {
		// Data is already uncompressed
		if len(data) < offset+int(uncompressSize) {
			entry.Message = fmt.Sprintf("Chunkset uncompressed data too small: need %d, have %d",
				offset+int(uncompressSize), len(data))
			return []*TraceV3Entry{entry}
		}
		decompressedData = data[offset : offset+int(uncompressSize)]

		// Update statistics for uncompressed data
		GlobalCompressionStats.UncompressedChunks++
		GlobalCompressionStats.TotalBytesDecompressed += uint64(len(decompressedData))

	} else {
		// Data is compressed, need to decompress using enhanced LZ4 decompression
		if len(data) < offset+int(blockSize) {
			entry.Message = fmt.Sprintf("Chunkset compressed data too small: need %d, have %d",
				offset+int(blockSize), len(data))
			return []*TraceV3Entry{entry}
		}

		compressedData := data[offset : offset+int(blockSize)]

		// Use enhanced decompression with subchunk metadata if available
		relevantSubchunk := findRelevantSubchunkForSize(uncompressSize)
		if relevantSubchunk != nil {
			var err error
			decompressedData, decompressionInfo, err = DecompressWithSubchunkInfo(compressedData, relevantSubchunk)
			if err != nil {
				entry.Message = fmt.Sprintf("Failed to decompress chunkset data with subchunk info: %v", err)
				return []*TraceV3Entry{entry}
			}
		} else {
			// Fallback to standard decompression
			const lz4Compression = 0x100
			decompressedData, err = DecompressChunksetData(compressedData, uncompressSize, lz4Compression)
			if err != nil {
				entry.Message = fmt.Sprintf("Failed to decompress chunkset data: %v", err)
				return []*TraceV3Entry{entry}
			}
		}
	}

	// Parse individual log entries from decompressed data
	// The decompressed data contains multiple log chunks that need to be parsed
	entries := parseDecompressedChunksetData(decompressedData, header, entry, timesyncData)

	// Add decompression information if available
	if decompressionInfo != nil && len(entries) > 0 {
		// Update the first entry with decompression details
		firstEntry := entries[0]
		if firstEntry.ChunkType == "chunkset_summary" {
			firstEntry.Message += fmt.Sprintf(" | Decompression: success=%t time=%v actual_size=%d",
				decompressionInfo.DecompressionSuccess, decompressionInfo.DecompressionTime, decompressionInfo.ActualSize)
		}
	}

	return entries
}

const (
	bv41             = 825521762
	bv41Uncompressed = 758412898

	firehoseChunk   = 0x6001
	oversizeChunk   = 0x6002
	statedumpChunk  = 0x6003
	simpledumpChunk = 0x6004
)

func parseChunkset(data []byte) (chunk ChunksetChunk, remainingData []byte, err error) {
	chunk = ChunksetChunk{}

	data, chunkTag, _ := utils.Take(data, 4)
	data, chunkSubtag, _ := utils.Take(data, 4)
	data, chunkDataSize, _ := utils.Take(data, 8)
	data, signature, _ := utils.Take(data, 4)
	data, uncompressedSize, _ := utils.Take(data, 4)

	chunk.ChunkTag = binary.LittleEndian.Uint32(chunkTag)
	chunk.ChunkSubtag = binary.LittleEndian.Uint32(chunkSubtag)
	chunk.ChunkDataSize = binary.LittleEndian.Uint64(chunkDataSize)
	chunk.Signature = binary.LittleEndian.Uint32(signature)
	chunk.UncompressedSize = binary.LittleEndian.Uint32(uncompressedSize)

	// Data is already uncompressed
	if chunk.Signature == bv41Uncompressed {
		data, decompressedData, _ := utils.Take(data, int(chunk.UncompressedSize))
		chunk.DecompressedData = decompressedData
		data, footer, _ := utils.Take(data, 4)
		chunk.Footer = binary.LittleEndian.Uint32(footer)
		return chunk, data, nil
	}

	if chunk.Signature != bv41 {
		return chunk, data, fmt.Errorf("invalid chunkset signature: %x, expected %x", chunk.Signature, bv41)
	}

	data, blockSize, _ := utils.Take(data, 4)
	chunk.BlockSize = binary.LittleEndian.Uint32(blockSize)
	data, compressedData, _ := utils.Take(data, int(chunk.BlockSize))

	decompressedData := make([]byte, chunk.UncompressedSize)

	// Decompress using LZ4
	n, err := lz4.UncompressBlock(compressedData, decompressedData)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to decompress chunkset data: %w", err)
	}

	chunk.DecompressedData = decompressedData[:n]
	data, footer, _ := utils.Take(data, 4)
	chunk.Footer = binary.LittleEndian.Uint32(footer)

	return chunk, data, nil
}

// ParseChunksetData parses each log in the decompressed Chunkset data
func ParseChunksetData(data []byte, ulData *UnifiedLogData) ([]*TraceV3Entry, error) {
	entries := []*TraceV3Entry{}

	for len(data) > 0 {
		// read preamble
		data, chunkTag, _ := utils.Take(data, 4)
		data, _, _ = utils.Take(data, 4) // chunkSubTag
		data, chunkDataSize, _ := utils.Take(data, 8)

		if uint64(chunkDataSize[0]) > uint64(^uint(0)>>1) { // Check if larger than max int
			return entries, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
		}

		data, chunkData, _ := utils.Take(data, int(chunkDataSize[0]))
		if err := getChunksetData(chunkData, binary.LittleEndian.Uint32(chunkTag), ulData); err != nil {
			return entries, err
		}

		// Calculate total chunk size (preamble + data)
		totalChunkSize := 16 + int(chunkDataSize)
		if offset+totalChunkSize > len(decompressedData) {
			break
		}

		// Extract chunk data
		chunkData := decompressedData[offset : offset+totalChunkSize]

		// Create base entry
		chunkEntry := &TraceV3Entry{
			Type:         chunkTag,
			Size:         uint32(chunkDataSize),
			Timestamp:    header.ContinuousTime + uint64(chunkCount)*1000000,
			ThreadID:     0,
			ProcessID:    header.LogdPID,
			Level:        "Info",
			MessageType:  "Default",
			EventType:    "logEvent",
			TimezoneName: extractTimezoneName(header.TimezonePath),
		}

		// Parse based on chunk type with enhanced processing
		switch chunkTag {
		// case 0x6001:
		// 	// Firehose chunk - contains individual log entries
		// 	chunkEntry.ChunkType = "firehose"
		// 	chunkEntry.Subsystem = "com.apple.firehose.decompressed"
		// 	chunkEntry.Category = "entry"

		// 	// Use enhanced firehose parsing with debugging
		// 	firehoseEntries := ParseFirehoseChunk(chunkData, chunkEntry, header, timesyncData)

		// 	// Add debug information if no entries were extracted
		// 	if len(firehoseEntries) == 0 {
		// 		debugEntry := &TraceV3Entry{
		// 			Type:         chunkTag,
		// 			Size:         uint32(chunkDataSize),
		// 			Timestamp:    header.ContinuousTime + uint64(chunkCount)*1000000,
		// 			ThreadID:     0,
		// 			ProcessID:    header.LogdPID,
		// 			Level:        "Debug",
		// 			MessageType:  "Debug",
		// 			EventType:    "logEvent",
		// 			TimezoneName: extractTimezoneName(header.TimezonePath),
		// 			ChunkType:    "firehose_debug",
		// 			Subsystem:    "com.apple.firehose.debug",
		// 			Category:     "parsing_debug",
		// 			Message:      fmt.Sprintf("Firehose chunk debug: size=%d data_preview=%x", chunkDataSize, chunkData[:min(32, len(chunkData))]),
		// 		}
		// 		entries = append(entries, debugEntry)
		// 	} else {
		// 		// Return only individual entries, no summary entry
		// 		// This replaces summary entries with actual log entries as requested
		// 	}

		// 	entries = append(entries, firehoseEntries...)

		// case 0x6002:
		// 	// Oversize chunk
		// 	chunkEntry.ChunkType = "oversize"
		// 	chunkEntry.Subsystem = "com.apple.oversize.decompressed"
		// 	chunkEntry.Category = "oversize_data"
		// 	ParseOversizeChunk(chunkData, chunkEntry, header, timesyncData)
		// 	entries = append(entries, chunkEntry)

		case 0x6003:
			// Statedump chunk
			chunkEntry.ChunkType = "statedump"
			chunkEntry.Subsystem = "com.apple.statedump.decompressed"
			chunkEntry.Category = "system_state"
			ParseStatedumpChunk(chunkData, chunkEntry)
			entries = append(entries, chunkEntry)

		case 0x6004:
			// SimpleDump chunk
			simpleDumpChunk := &SimpleDumpChunk{}

			ParseSimpleDumpChunk(chunkData, simpleDumpChunk)
			// TODO: Update chunckEntry with simpledump data
			// chunkEntry.ChunkType = "simpledump"
			// entries = append(entries, chunkEntry)

		default:
			// Unknown chunk type
			chunkEntry.ChunkType = "unknown_decompressed"
			chunkEntry.Subsystem = "com.apple.unknown.decompressed"
			chunkEntry.Category = fmt.Sprintf("unknown_0x%x", chunkTag)
			chunkEntry.Message = fmt.Sprintf("Unknown decompressed chunk: tag=0x%x sub_tag=0x%x size=%d",
				chunkTag, chunkSubTag, chunkDataSize)
			entries = append(entries, chunkEntry)
		}

		// Move to next chunk with 8-byte alignment padding
		offset += totalChunkSize
		paddingBytes := (8 - (chunkDataSize & 7)) & 7
		offset += int(paddingBytes)

		// Skip any zero padding
		for offset < len(decompressedData) && decompressedData[offset] == 0 {
			offset++
		}
		remainingData := data[offset:]
		if len(remainingData) == 0 {
			break
		}

		if len(remainingData) < 16 {
			// TODO: Warn, not enough data for Chunkset preamble header, needed 16 bytes. Got: %d
			break
		}
	}
	return entries, nil
}

func getChunksetData(data []byte, chunkTag uint32, ulData *UnifiedLogData) error {
	switch chunkTag {
	case firehoseChunk:
		firehosePreamble, _ := firehose.ParseFirehosePreamble(data)
		ulData.FirehoseData = append(ulData.FirehoseData, firehosePreamble)
	case oversizeChunk:
		oversizeChunk, _, err := ParseOversizeChunk(data)
		if err != nil {
			return err
		}
		ulData.OversizeData = append(ulData.OversizeData, oversizeChunk)
	// TODO: uncomment once statedump and simpledump are merged
	// case statedumpChunk:
	// 	statedumpChunk, err := ParseStatedump(*data)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	ulData.StatedumpData = append(ulData.StatedumpData, statedumpChunk)
	// case simpledumpChunk:
	// 	simpledumpChunk, _ := ParseSimpledumpChunk(*data)
	// 	ulData.SimpledumpData = append(ulData.SimpledumpData, simpledumpChunk)
	default:
		return fmt.Errorf("unknown chunk tag: %x", chunkTag)
	}

	// If no exact match, find the closest one
	var bestMatch *CatalogSubchunk
	var smallestDiff = ^uint32(0) // Max uint32

	for _, subchunk := range GlobalCatalog.CatalogSubchunks {
		diff := uint32(0)
		if subchunk.UncompressedSize > dataSize {
			diff = subchunk.UncompressedSize - dataSize
		} else {
			diff = dataSize - subchunk.UncompressedSize
		}

		if diff < smallestDiff {
			smallestDiff = diff
			bestMatch = &subchunk
		}
	}

	return bestMatch
}

// findRelevantSubchunkForSize finds a catalog subchunk that matches the given uncompressed size
func findRelevantSubchunkForSize(uncompressedSize uint32) *CatalogSubchunk {
	if GlobalCatalog == nil || len(GlobalCatalog.CatalogSubchunks) == 0 {
		return nil
	}

	// Find exact match first
	for _, subchunk := range GlobalCatalog.CatalogSubchunks {
		if subchunk.UncompressedSize == uncompressedSize {
			return &subchunk
		}
	}

	return nil // Return nil if no exact match (could enhance with fuzzy matching if needed)
}
