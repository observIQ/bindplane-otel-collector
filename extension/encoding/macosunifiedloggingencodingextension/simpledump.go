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
)

// SimpledumpChunk represents the parsed data from a simpledump chunk
type SimpledumpChunk struct {
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

// parseUUID converts a 16-byte UUID to string format matching the rust implementation
func parseUUID(data []byte) string {
	if len(data) < 16 {
		return ""
	}

	// Format as uppercase hex string to match rust implementation
	// The rust code does format!("{uuid:02X?}") which creates a debug format with uppercase hex
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
		data[0], data[1], data[2], data[3], data[4], data[5], data[6], data[7],
		data[8], data[9], data[10], data[11], data[12], data[13], data[14], data[15])
}

// extractString extracts a null-terminated string from binary data
func extractString(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// Find null terminator
	end := len(data)
	for i, b := range data {
		if b == 0 {
			end = i
			break
		}
	}

	return string(data[:end])
}

// ParseSimpledumpChunk parses a Simpledump chunk (0x6004) containing simple string data
// Based on the rust implementation in chunks/simpledump.rs
// Note: The data passed in includes the complete chunk with 16-byte header (tag, subtag, size)
func ParseSimpledumpChunk(data []byte, chunk *SimpledumpChunk) {
	offset := 0

	// Parse chunk header (16 bytes total)
	chunk.ChunkTag = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	chunk.ChunkSubTag = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	chunk.ChunkDataSize = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	// Parse simpledump payload fields
	chunk.FirstProcID = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	chunk.SecondProcID = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	chunk.ContinuousTime = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	chunk.ThreadID = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	chunk.UnknownOffset = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	chunk.UnknownTTL = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	chunk.UnknownType = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	chunk.SenderUUID = parseUUID(data[offset : offset+16])
	offset += 16
	chunk.DSCSharedCacheUUID = parseUUID(data[offset : offset+16])
	offset += 16
	chunk.UnknownNumberMessageStrings = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	chunk.UnknownSizeSubsystemString = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	chunk.UnknownSizeMessageString = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	chunk.Subsystem = extractString(data[offset : offset+int(chunk.UnknownSizeSubsystemString)])
	offset += int(chunk.UnknownSizeSubsystemString)
	chunk.MessageString = extractString(data[offset : offset+int(chunk.UnknownSizeMessageString)])
	offset += int(chunk.UnknownSizeMessageString)
}
