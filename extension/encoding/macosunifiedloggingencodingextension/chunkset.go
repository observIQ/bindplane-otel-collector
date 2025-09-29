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
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
	"github.com/pierrec/lz4/v4"
)

// ChunksetChunk represents a parsed chunkset chunk
type ChunksetChunk struct {
	ChunkTag         uint32
	ChunkSubtag      uint32
	ChunkDataSize    uint64
	Signature        uint32
	UncompressedSize uint32
	BlockSize        uint32
	DecompressedData []byte
	Footer           uint32
}

const (
	bv41             = 825521762
	bv41Uncompressed = 758412898

	firehoseChunk   = 0x6001
	oversizeChunk   = 0x6002
	statedumpChunk  = 0x6003
	simpledumpChunk = 0x6004
)

// ParseChunkset parses a chunkset chunk into its header fields, decompresses
// the payload when necessary, and returns the parsed chunk alongside the
// remaining unconsumed data.
func ParseChunkset(chunksetData []byte) (chunk ChunksetChunk, remainingData []byte, err error) {
	chunk = ChunksetChunk{}

	var chunkTag []byte
	chunksetData, chunkTag, err = helpers.Take(chunksetData, 4)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read chunk tag: %w", err)
	}

	var chunkSubtag []byte
	chunksetData, chunkSubtag, err = helpers.Take(chunksetData, 4)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read chunk subtag: %w", err)
	}

	var chunkDataSize []byte
	chunksetData, chunkDataSize, err = helpers.Take(chunksetData, 8)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read chunk data size: %w", err)
	}

	var signature []byte
	chunksetData, signature, err = helpers.Take(chunksetData, 4)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read signature: %w", err)
	}

	var uncompressedSize []byte
	chunksetData, uncompressedSize, err = helpers.Take(chunksetData, 4)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read uncompressed size: %w", err)
	}

	chunk.ChunkTag = binary.LittleEndian.Uint32(chunkTag)
	chunk.ChunkSubtag = binary.LittleEndian.Uint32(chunkSubtag)
	chunk.ChunkDataSize = binary.LittleEndian.Uint64(chunkDataSize)
	chunk.Signature = binary.LittleEndian.Uint32(signature)
	chunk.UncompressedSize = binary.LittleEndian.Uint32(uncompressedSize)

	// Data is already uncompressed
	if chunk.Signature == bv41Uncompressed {
		var decompressedData []byte
		chunksetData, decompressedData, err = helpers.Take(chunksetData, int(chunk.UncompressedSize))
		if err != nil {
			return chunk, chunksetData, fmt.Errorf("failed to read decompressed data: %w", err)
		}
		chunk.DecompressedData = decompressedData

		var footer []byte
		chunksetData, footer, err = helpers.Take(chunksetData, 4)
		if err != nil {
			return chunk, chunksetData, fmt.Errorf("failed to read footer: %w", err)
		}
		chunk.Footer = binary.LittleEndian.Uint32(footer)
		return chunk, chunksetData, nil
	}

	if chunk.Signature != bv41 {
		return chunk, chunksetData, fmt.Errorf("invalid chunkset signature: %x, expected %x", chunk.Signature, bv41)
	}

	var blockSize []byte
	chunksetData, blockSize, err = helpers.Take(chunksetData, 4)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read block size: %w", err)
	}
	chunk.BlockSize = binary.LittleEndian.Uint32(blockSize)

	var compressedData []byte
	chunksetData, compressedData, err = helpers.Take(chunksetData, int(chunk.BlockSize))
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read compressed data: %w", err)
	}

	decompressedData := make([]byte, chunk.UncompressedSize)

	// Decompress using LZ4
	n, err := lz4.UncompressBlock(compressedData, decompressedData)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to decompress chunkset data: %w", err)
	}

	chunk.DecompressedData = decompressedData[:n]

	var footer []byte
	chunksetData, footer, err = helpers.Take(chunksetData, 4)
	if err != nil {
		return chunk, chunksetData, fmt.Errorf("failed to read footer: %w", err)
	}
	chunk.Footer = binary.LittleEndian.Uint32(footer)

	return chunk, chunksetData, nil
}

// ParseChunksetData parses each log in the decompressed Chunkset data
func ParseChunksetData(decompressedChucksetData []byte, ulData *UnifiedLogCatalogData) error {
	chunkPreambleSize := 16

	for len(decompressedChucksetData) > 0 {
		// read preamble
		preamble, err := DetectPreamble(decompressedChucksetData)
		if err != nil {
			return fmt.Errorf("failed to detect preamble: %w", err)
		}
		chunkDataSize := preamble.ChunkDataSize

		if chunkDataSize > uint64(^uint(0)>>1) { // Check if larger than max int
			return fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
		}
		// Consumes chunkData and Preamble bytes off data and returns the chunkData
		var chunkData []byte
		decompressedChucksetData, chunkData, err = helpers.Take(decompressedChucksetData, int(chunkDataSize)+chunkPreambleSize)
		if err != nil {
			return fmt.Errorf("failed to read chunk data: %w", err)
		}

		if err := getChunksetData(chunkData, preamble.ChunkTag, ulData); err != nil {
			return fmt.Errorf("failed to get chunkset data: %w", err)
		}

		// skip zero padding
		offset := 0
		for offset < len(decompressedChucksetData) && decompressedChucksetData[offset] == 0 {
			offset++
		}
		// Consumes the zero padding off currentData and updates currentData
		decompressedChucksetData = decompressedChucksetData[offset:]
		if len(decompressedChucksetData) == 0 {
			break
		}

		if len(decompressedChucksetData) < 16 {
			// TODO: Warn, not enough data for Chunkset preamble header, needed 16 bytes. Got: %d
			break
		}
	}
	return nil
}

func getChunksetData(data []byte, chunkTag uint32, ulData *UnifiedLogCatalogData) error {
	switch chunkTag {
	case firehoseChunk:
		firehosePreamble, _, err := firehose.ParseFirehosePreamble(data)
		if err != nil {
			return fmt.Errorf("failed to parse firehose preamble: %w", err)
		}
		ulData.FirehoseData = append(ulData.FirehoseData, firehosePreamble)
	case oversizeChunk:
		oversizeChunk, _, err := ParseOversizeChunk(data)
		if err != nil {
			return err
		}
		ulData.OversizeData = append(ulData.OversizeData, oversizeChunk)
	case statedumpChunk:
		fmt.Printf("Found Statedump chunk type: 0x%x, size: %d\\n", chunkTag, len(data))
		statedumpChunkData, err := ParseStateDump(data)
		if err != nil {
			return err
		}
		ulData.StatedumpData = append(ulData.StatedumpData, *statedumpChunkData)
	case simpledumpChunk:
		fmt.Printf("Found Simpledump chunk type: 0x%x, size: %d\\n", chunkTag, len(data))
		simpledumpChunkData, _, err := ParseSimpleDumpChunk(data)
		if err != nil {
			return err
		}
		ulData.SimpledumpData = append(ulData.SimpledumpData, simpledumpChunkData)

	default:
		return fmt.Errorf("unknown chunk tag: %x", chunkTag)
	}
	return nil
}
