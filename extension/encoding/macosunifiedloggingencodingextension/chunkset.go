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

func ParseChunkset(data []byte) (chunk ChunksetChunk, remainingData []byte, err error) {
	chunk = ChunksetChunk{}

	data, chunkTag, err := helpers.Take(data, 4)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read chunk tag: %w", err)
	}
	data, chunkSubtag, err := helpers.Take(data, 4)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read chunk subtag: %w", err)
	}
	data, chunkDataSize, err := helpers.Take(data, 8)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read chunk data size: %w", err)
	}
	data, signature, err := helpers.Take(data, 4)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read signature: %w", err)
	}
	data, uncompressedSize, err := helpers.Take(data, 4)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read uncompressed size: %w", err)
	}

	chunk.ChunkTag = binary.LittleEndian.Uint32(chunkTag)
	chunk.ChunkSubtag = binary.LittleEndian.Uint32(chunkSubtag)
	chunk.ChunkDataSize = binary.LittleEndian.Uint64(chunkDataSize)
	chunk.Signature = binary.LittleEndian.Uint32(signature)
	chunk.UncompressedSize = binary.LittleEndian.Uint32(uncompressedSize)

	// Data is already uncompressed
	if chunk.Signature == bv41Uncompressed {
		data, decompressedData, err := helpers.Take(data, int(chunk.UncompressedSize))
		if err != nil {
			return chunk, data, fmt.Errorf("failed to read decompressed data: %w", err)
		}
		chunk.DecompressedData = decompressedData
		data, footer, err := helpers.Take(data, 4)
		if err != nil {
			return chunk, data, fmt.Errorf("failed to read footer: %w", err)
		}
		chunk.Footer = binary.LittleEndian.Uint32(footer)
		return chunk, data, nil
	}

	if chunk.Signature != bv41 {
		return chunk, data, fmt.Errorf("invalid chunkset signature: %x, expected %x", chunk.Signature, bv41)
	}

	data, blockSize, err := helpers.Take(data, 4)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read block size: %w", err)
	}
	chunk.BlockSize = binary.LittleEndian.Uint32(blockSize)
	data, compressedData, err := helpers.Take(data, int(chunk.BlockSize))
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read compressed data: %w", err)
	}

	decompressedData := make([]byte, chunk.UncompressedSize)

	// Decompress using LZ4
	n, err := lz4.UncompressBlock(compressedData, decompressedData)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to decompress chunkset data: %w", err)
	}

	chunk.DecompressedData = decompressedData[:n]
	data, footer, err := helpers.Take(data, 4)
	if err != nil {
		return chunk, data, fmt.Errorf("failed to read footer: %w", err)
	}
	chunk.Footer = binary.LittleEndian.Uint32(footer)

	return chunk, data, nil
}

// ParseChunksetData parses each log in the decompressed Chunkset data
func ParseChunksetData(data []byte, ulData *UnifiedLogCatalogData) error {
	for len(data) > 0 {
		// read preamble
		data, chunkTag, err := helpers.Take(data, 4)
		if err != nil {
			return entries, fmt.Errorf("failed to read chunk tag: %w", err)
		}
		data, _, err = helpers.Take(data, 4) // chunkSubTag
		if err != nil {
			return entries, fmt.Errorf("failed to read chunk sub tag: %w", err)
		}
		data, chunkDataSize, err := helpers.Take(data, 8)
		if err != nil {
			return entries, fmt.Errorf("failed to read chunk data size: %w", err)
		}

		if uint64(chunkDataSize[0]) > uint64(^uint(0)>>1) { // Check if larger than max int
			return fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
		}

		data, chunkData, err := helpers.Take(data, int(chunkDataSize[0]))
		if err != nil {
			return entries, fmt.Errorf("failed to read chunk data: %w", err)
		}
		if err := getChunksetData(chunkData, binary.LittleEndian.Uint32(chunkTag), ulData); err != nil {
			return err
		}

		// skip zero padding
		offset := 0
		for offset < len(data) && data[offset] == 0 {
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
	return nil
}
