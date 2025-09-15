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

func parseChunkset(data []byte) (chunk ChunksetChunk, remainingData []byte, err error) {
	chunk = ChunksetChunk{}
	chunk := ChunksetChunk{}

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
	return entries, nil
}

func getChunksetData(data []byte, chunkTag uint32, ulData *UnifiedLogData) error {
	switch chunkTag {
	case firehoseChunk:
		firehosePreamble, _ := firehose.ParseFirehosePreamble(data)
		ulData.FirehoseData = append(ulData.FirehoseData, firehosePreamble)
	case oversizeChunk:
		oversizeChunk, _ := ParseOversizeChunk(data)
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
