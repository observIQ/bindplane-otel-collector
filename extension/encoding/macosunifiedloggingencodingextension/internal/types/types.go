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

package types

import "strings"

// CatalogChunk represents a parsed Catalog chunk (0x600b)
type CatalogChunk struct {
	// Header fields (16 bytes)
	ChunkTag      uint32
	ChunkSubtag   uint32
	ChunkDataSize uint64

	// Catalog structure fields
	CatalogSubsystemStringsOffset   uint16
	CatalogProcessInfoEntriesOffset uint16
	NumberProcessInformationEntries uint16
	CatalogOffsetSubChunks          uint16
	NumberSubChunks                 uint16
	Unknown                         []byte // 6 bytes padding
	EarliestFirehoseTimestamp       uint64

	// Parsed data
	CatalogUUIDs            []string
	CatalogSubsystemStrings []byte
	ProcessInfoEntries      []ProcessInfoEntry
	CatalogSubchunks        []CatalogSubchunk
}

// GetSubsystem resolves subsystem ID to human-readable subsystem and category
// Based on the rust implementation's get_subsystem method
func (c *CatalogChunk) GetSubsystem(subsystemValue uint16, firstProcID uint64, secondProcID uint32) SubsystemInfo {
	if c == nil {
		return SubsystemInfo{Subsystem: "Unknown subsystem", Category: ""}
	}

	// Look for the process entry that matches the proc IDs
	for _, procEntry := range c.ProcessInfoEntries {
		if procEntry.FirstNumberProcID == firstProcID && procEntry.SecondNumberProcID == secondProcID {
			// Search through the subsystem entries for this process
			for _, subsysEntry := range procEntry.SubsystemEntries {
				if subsystemValue == subsysEntry.Identifier {
					// Extract subsystem string using offset
					subsystemString := extractStringAtOffset(c.CatalogSubsystemStrings, int(subsysEntry.SubsystemOffset))

					// Extract category string using offset
					categoryString := extractStringAtOffset(c.CatalogSubsystemStrings, int(subsysEntry.CategoryOffset))

					return SubsystemInfo{
						Subsystem: subsystemString,
						Category:  categoryString,
					}
				}
			}
			break
		}
	}

	return SubsystemInfo{Subsystem: "Unknown subsystem", Category: ""}
}

// GetPID resolves internal process IDs to actual PID
// Based on the rust implementation's get_pid method
func (c *CatalogChunk) GetPID(firstProcID uint64, secondProcID uint32) uint32 {
	if c == nil {
		return 0
	}

	// Look for the process entry
	for _, procEntry := range c.ProcessInfoEntries {
		if procEntry.FirstNumberProcID == firstProcID && procEntry.SecondNumberProcID == secondProcID {
			return procEntry.PID
		}
	}

	return 0
}

// GetEUID resolves internal process IDs to effective user ID
// Based on the rust implementation's get_euid method
func (c *CatalogChunk) GetEUID(firstProcID uint64, secondProcID uint32) uint32 {
	if c == nil {
		return 0
	}

	// Look for the process entry
	for _, procEntry := range c.ProcessInfoEntries {
		if procEntry.FirstNumberProcID == firstProcID && procEntry.SecondNumberProcID == secondProcID {
			return procEntry.EffectiveUserID
		}
	}

	return 0
}

// ProcessInfoEntry represents process information in the catalog
type ProcessInfoEntry struct {
	Index                uint16
	Unknown              uint16
	CatalogMainUUIDIndex uint16
	CatalogDSCUUIDIndex  uint16
	FirstNumberProcID    uint64
	SecondNumberProcID   uint32
	PID                  uint32
	EffectiveUserID      uint32
	Unknown2             uint32
	NumberUUIDsEntries   uint32
	Unknown3             uint32
	UUIDInfoEntries      []ProcessUUIDEntry
	NumberSubsystems     uint32
	Unknown4             uint32
	SubsystemEntries     []ProcessInfoSubsystem
	MainUUID             string
	DSCUUID              string
}

// ProcessUUIDEntry represents UUID information in the catalog
type ProcessUUIDEntry struct {
	Size             uint32
	Unknown          uint32
	CatalogUUIDIndex uint16
	LoadAddress      uint64
	UUID             string
}

// ProcessInfoSubsystem represents subsystem metadata in the catalog
// This helps get the subsystem (App Bundle ID) and the log entry category
type ProcessInfoSubsystem struct {
	Identifier      uint16 // Subsystem identifier to match against log entries
	SubsystemOffset uint16 // Offset to subsystem string in catalog_subsystem_strings
	CategoryOffset  uint16 // Offset to category string in catalog_subsystem_strings
}

// CatalogSubchunk represents metadata for compressed log data
type CatalogSubchunk struct {
	Start                uint64
	End                  uint64
	UncompressedSize     uint32
	CompressionAlgorithm uint32
	NumberIndex          uint32
	Indexes              []uint16
	NumberStringOffsets  uint32
	StringOffsets        []uint16
}

// SubsystemInfo holds resolved subsystem and category information
type SubsystemInfo struct {
	Subsystem string
	Category  string
}

// MessageData represents parsed message data
type MessageData struct {
	FormatString string
	Process      string
	Library      string
	LibraryUUID  string
	ProcessUUID  string
}

// FileProvider represents a file provider interface
type FileProvider interface {
	// Add methods as needed
}

// extractStringAtOffset extracts a null-terminated string from a byte array at a specific offset
// This matches the rust implementation's approach for extracting subsystem and category strings
func extractStringAtOffset(data []byte, offset int) string {
	if offset >= len(data) {
		return ""
	}

	// Find null terminator starting from offset
	start := offset
	end := len(data)
	for i := start; i < len(data); i++ {
		if data[i] == 0 {
			end = i
			break
		}
	}

	if start >= end {
		return ""
	}

	return strings.TrimSpace(string(data[start:end]))
}
