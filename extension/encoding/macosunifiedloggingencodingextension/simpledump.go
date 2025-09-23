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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
)

// SimpleDumpChunk represents the parsed data from a SimpleDump chunk
type SimpleDumpChunk struct {
	ChunkTag                    uint32
	ChunkSubTag                 uint32
	ChunkDataSize               uint64
	FirstProcID                 uint64
	SecondProcID                uint64
	ContinuousTime              uint64
	ThreadID                    uint64
	UnknownOffset               uint32
	UnknownTTL                  uint16
	UnknownType                 uint16
	SenderUUID                  string
	DSCSharedCacheUUID          string
	UnknownNumberMessageStrings uint32
	UnknownSizeSubsystemString  uint32
	UnknownSizeMessageString    uint32
	Subsystem                   string
	MessageString               string
}

// parseUUID converts a 16-byte UUID to string format
func parseUUID(data []byte) string {
	if len(data) < 16 {
		return ""
	}

	// Format as uppercase hex string
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
		data[0], data[1], data[2], data[3], data[4], data[5], data[6], data[7],
		data[8], data[9], data[10], data[11], data[12], data[13], data[14], data[15])
}

// ParseSimpleDumpChunk parses a SimpleDump Chunk containing simple string data
// Returns the parsed SimpleDump chunk and the remaining data
// Note: The data passed in includes the complete chunk with 16-byte header (tag, subtag, size)
func ParseSimpleDumpChunk(data []byte) (SimpleDumpChunk, []byte, error) {
	var simpleDumpResult SimpleDumpChunk

	// Parse SimpleDump Chunk chunk header (16 bytes total)
	data, chunkTag, err := helpers.Take(data, 4)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read chunk tag: %w", err)
	}
	data, chunkSubTag, err := helpers.Take(data, 4)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read chunk sub tag: %w", err)
	}
	data, chunkDataSize, err := helpers.Take(data, 8)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read chunk data size: %w", err)
	}
	data, firstProcID, err := helpers.Take(data, 8)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read first proc ID: %w", err)
	}
	data, secondProcID, err := helpers.Take(data, 8)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read second proc ID: %w", err)
	}
	data, continuousTime, err := helpers.Take(data, 8)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read continuous time: %w", err)
	}
	data, threadID, err := helpers.Take(data, 8)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read thread ID: %w", err)
	}
	data, unknownOffset, err := helpers.Take(data, 4)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read unknown offset: %w", err)
	}
	data, unknownTTL, err := helpers.Take(data, 2)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read unknown TTL: %w", err)
	}
	data, unknownType, err := helpers.Take(data, 2)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read unknown type: %w", err)
	}
	data, senderUUID, err := helpers.Take(data, 16)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read sender UUID: %w", err)
	}
	data, dSCSharedCacheUUID, err := helpers.Take(data, 16)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read DSC shared cache UUID: %w", err)
	}
	data, unknownNumberMessageStrings, err := helpers.Take(data, 4)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read unknown number message strings: %w", err)
	}
	data, unknownSizeSubsystemString, err := helpers.Take(data, 4)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read unknown size subsystem string: %w", err)
	}
	data, unknownSizeMessageString, err := helpers.Take(data, 4)
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to read unknown size message string: %w", err)
	}
	data, subsystem, err := helpers.ExtractStringSize(data, uint64(binary.LittleEndian.Uint32(unknownSizeSubsystemString)))
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to extract subsystem string: %w", err)
	}
	data, messageString, err := helpers.ExtractStringSize(data, uint64(binary.LittleEndian.Uint32(unknownSizeMessageString)))
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to extract message string: %w", err)
	}
	data, messageString, err = helpers.ExtractStringSize(data, uint64(binary.LittleEndian.Uint32(unknownSizeMessageString)))
	if err != nil {
		return simpleDumpResult, data, fmt.Errorf("failed to extract message string: %w", err)
	}

	simpleDumpResult.ChunkTag = binary.LittleEndian.Uint32(chunkTag)
	simpleDumpResult.ChunkSubTag = binary.LittleEndian.Uint32(chunkSubTag)
	simpleDumpResult.ChunkDataSize = binary.LittleEndian.Uint64(chunkDataSize)
	simpleDumpResult.FirstProcID = binary.LittleEndian.Uint64(firstProcID)
	simpleDumpResult.SecondProcID = binary.LittleEndian.Uint64(secondProcID)
	simpleDumpResult.ContinuousTime = binary.LittleEndian.Uint64(continuousTime)
	simpleDumpResult.ThreadID = binary.LittleEndian.Uint64(threadID)
	simpleDumpResult.UnknownOffset = binary.LittleEndian.Uint32(unknownOffset)
	simpleDumpResult.UnknownTTL = binary.LittleEndian.Uint16(unknownTTL)
	simpleDumpResult.UnknownType = binary.LittleEndian.Uint16(unknownType)
	simpleDumpResult.SenderUUID = parseUUID(senderUUID)
	simpleDumpResult.DSCSharedCacheUUID = parseUUID(dSCSharedCacheUUID)
	simpleDumpResult.UnknownNumberMessageStrings = binary.LittleEndian.Uint32(unknownNumberMessageStrings)
	simpleDumpResult.UnknownSizeSubsystemString = binary.LittleEndian.Uint32(unknownSizeSubsystemString)
	simpleDumpResult.UnknownSizeMessageString = binary.LittleEndian.Uint32(unknownSizeMessageString)
	simpleDumpResult.Subsystem = subsystem
	simpleDumpResult.MessageString = messageString

	return simpleDumpResult, data, nil
}
