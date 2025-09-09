// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

type StateDump struct {
	ChunkTag        uint32
	ChunkSubtag     uint32
	ChunkDataSize   uint64
	FirstProcID     uint64
	SecondProcID    uint32
	TTL             uint8
	UnknownReserved []uint8
	ContinuousTime  uint64
	ActivityID      uint64
	UUID            string
	UnknownDataType uint32 // 1 = plist, 3 = custom object?, 2 = (protocol buffer?)
	UnknownDataSize uint32 // Size of statedump data
	DecoderLibrary  string
	DecoderType     string
	TitleName       string
	Data            []uint8
}

// ParseStateDump parses Statedump log entry. Statedumps are special log entries that may contain a plist file, custom object, or protocol buffer
func ParseStateDump(data []byte) (*StateDump, error) {
	if len(data) < 72 { // Minimum size for basic structure
		return nil, fmt.Errorf("statedump data too small: %d bytes", len(data))
	}

	reader := bytes.NewReader(data)
	result := &StateDump{}

	// Parse the header fields
	if err := binary.Read(reader, binary.LittleEndian, &result.ChunkTag); err != nil {
		return nil, fmt.Errorf("failed to read chunk tag: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.ChunkSubtag); err != nil {
		return nil, fmt.Errorf("failed to read chunk subtag: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.ChunkDataSize); err != nil {
		return nil, fmt.Errorf("failed to read chunk data size: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.FirstProcID); err != nil {
		return nil, fmt.Errorf("failed to read first proc id: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.SecondProcID); err != nil {
		return nil, fmt.Errorf("failed to read second proc id: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.TTL); err != nil {
		return nil, fmt.Errorf("failed to read ttl: %w", err)
	}

	// Read 3 bytes of unknown reserved data
	unknownReserved := make([]byte, 3)
	if _, err := reader.Read(unknownReserved); err != nil {
		return nil, fmt.Errorf("failed to read unknown reserved: %w", err)
	}
	result.UnknownReserved = unknownReserved

	if err := binary.Read(reader, binary.LittleEndian, &result.ContinuousTime); err != nil {
		return nil, fmt.Errorf("failed to read continuous time: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.ActivityID); err != nil {
		return nil, fmt.Errorf("failed to read activity id: %w", err)
	}

	// Read UUID as 16 bytes
	var uuidBytes [16]byte
	if err := binary.Read(reader, binary.LittleEndian, &uuidBytes); err != nil {
		return nil, fmt.Errorf("failed to read uuid: %w", err)
	}
	result.UUID = cleanUUID(fmt.Sprintf("%02X", uuidBytes))

	if err := binary.Read(reader, binary.LittleEndian, &result.UnknownDataType); err != nil {
		return nil, fmt.Errorf("failed to read unknown data type: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.UnknownDataSize); err != nil {
		return nil, fmt.Errorf("failed to read unknown data size: %w", err)
	}

	const customDecoder = 3
	const stringSize = 64

	// Handle custom decoder case or skip unknown data
	if result.UnknownDataType != customDecoder {
		// Skip 2 * 64 bytes of unknown data
		if _, err := reader.Seek(2*stringSize, 1); err != nil {
			return nil, fmt.Errorf("failed to skip unknown data: %w", err)
		}
	} else {
		// Read library and type data for custom decoder
		libraryData := make([]byte, stringSize)
		if _, err := reader.Read(libraryData); err != nil {
			return nil, fmt.Errorf("failed to read library data: %w", err)
		}
		result.DecoderLibrary = extractString(libraryData)

		typeData := make([]byte, stringSize)
		if _, err := reader.Read(typeData); err != nil {
			return nil, fmt.Errorf("failed to read type data: %w", err)
		}
		result.DecoderType = extractString(typeData)
	}

	// Read title data
	titleData := make([]byte, stringSize)
	if _, err := reader.Read(titleData); err != nil {
		return nil, fmt.Errorf("failed to read title data: %w", err)
	}
	result.TitleName = extractString(titleData)

	// Read the actual statedump data
	if result.UnknownDataSize > 0 {
		statedumpData := make([]byte, result.UnknownDataSize)
		if _, err := reader.Read(statedumpData); err != nil {
			return nil, fmt.Errorf("failed to read statedump data: %w", err)
		}
		result.Data = statedumpData
	}

	return result, nil
}

// ParseStateDumpPlist parses the binary plist file in the log. The plist may be empty
func ParseStateDumpPlist(plistData []byte) string {
	if len(plistData) == 0 {
		return "Empty plist data"
	}
	return fmt.Sprintf("Plist data (hex): %x", plistData)
}

// ParseStateDumpObject parses custom Apple objects
func ParseStateDumpObject(objectData []byte, name string) string {
	switch name {
	case "CLDaemonStatusStateTracker":
		return "TODO"
	case "CLClientManagerStateTracker":
		return getStateTrackerData(objectData)
	case "CLLocationManagerStateTracker":
		return getLocationTrackerState(objectData)
	case "DNS Configuration":
		return getDNSConfig(objectData)
	case "Network information":
		return getNetworkInterface(objectData)
	default:
		return fmt.Sprintf("Unsupported Statedump object: %s-%s", name, encodeStandard(objectData))
	}
}

// cleanUUID formats a UUID string by removing brackets and formatting properly
func cleanUUID(uuidString string) string {
	// Remove any brackets and clean up the string
	cleaned := strings.ReplaceAll(uuidString, "[", "")
	cleaned = strings.ReplaceAll(cleaned, "]", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return cleaned
}

// encodeStandard provides hex encoding for unsupported objects
func encodeStandard(data []byte) string {
	return hex.EncodeToString(data)
}

func getStateTrackerData(data []byte) string {
	return fmt.Sprintf("StateTracker data: %x", data)
}

func getLocationTrackerState(data []byte) string {
	return fmt.Sprintf("LocationTracker state: %x", data)
}

func getDNSConfig(data []byte) string {
	return fmt.Sprintf("DNS config: %x", data)
}

func getNetworkInterface(data []byte) string {
	return fmt.Sprintf("Network interface: %x", data)
}

// ParseStatedumpChunk parses a Statedump chunk (0x6003) containing system state information
func ParseStatedumpChunk(data []byte, entry *TraceV3Entry) {
	statedump, err := ParseStateDump(data)
	if err != nil {
		entry.Message = fmt.Sprintf("Failed to parse statedump: %v", err)
		entry.Level = "Error"
		return
	}

	var message string
	switch statedump.UnknownDataType {
	case 1: // plist
		message = ParseStateDumpPlist(statedump.Data)
	case 3: // custom object
		message = ParseStateDumpObject(statedump.Data, statedump.TitleName)
	default:
		message = fmt.Sprintf("Unknown statedump type %d: %x", statedump.UnknownDataType, statedump.Data)
	}

	entry.Message = fmt.Sprintf("Statedump [%s]: %s", statedump.TitleName, message)
	entry.Level = "Debug"
}
