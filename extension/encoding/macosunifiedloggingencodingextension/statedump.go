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
	result.UUID = formatUUID(uuidBytes[:])

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
	case "CLClientManagerStateTracker":
		return getClientManagerStateTrackerData(objectData)
	case "CLDaemonStatusStateTracker":
		return getDaemonStatusStateTrackerData(objectData)
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

// formatUUID formats a UUID byte array into standard UUID string format
func formatUUID(uuidBytes []byte) string {
	if len(uuidBytes) != 16 {
		return hex.EncodeToString(uuidBytes)
	}

	// Format as uppercase hex string without dashes (Rust-compatible format)
	// Using big-endian for proper UUID formatting
	return fmt.Sprintf("%08X%04X%04X%04X%012X",
		binary.BigEndian.Uint32(uuidBytes[0:4]),
		binary.BigEndian.Uint16(uuidBytes[4:6]),
		binary.BigEndian.Uint16(uuidBytes[6:8]),
		binary.BigEndian.Uint16(uuidBytes[8:10]),
		uuidBytes[10:16])
}

// encodeStandard provides hex encoding for unsupported objects
func encodeStandard(data []byte) string {
	return hex.EncodeToString(data)
}

func getClientManagerStateTrackerData(data []byte) string {
	if len(data) == 0 {
		return "Empty CLClientManagerStateTracker data"
	}

	// Try to parse basic structure
	if len(data) >= 8 {
		// Common pattern: first 4 bytes might be a count or type indicator
		count := binary.LittleEndian.Uint32(data[0:4])
		flags := binary.LittleEndian.Uint32(data[4:8])
		return fmt.Sprintf("CLClientManagerStateTracker: count=%d flags=0x%x data_size=%d", count, flags, len(data))
	}

	return string(data)
}

func getDaemonStatusStateTrackerData(data []byte) string {
	if len(data) == 0 {
		return "Empty CLDaemonStatusStateTracker data"
	}

	return string(data)
}

func getLocationTrackerState(data []byte) string {
	if len(data) == 0 {
		return "Empty CLLocationManagerStateTracker data"
	}

	// Try to parse location manager state structure
	if len(data) >= 16 {
		// Location managers often have status flags and coordinates
		status := binary.LittleEndian.Uint32(data[0:4])
		accuracy := binary.LittleEndian.Uint32(data[4:8])
		timestamp := binary.LittleEndian.Uint64(data[8:16])

		result := fmt.Sprintf("CLLocationManagerStateTracker: status=0x%x accuracy=%d timestamp=%d", status, accuracy, timestamp)

		// Try to extract more location data if available
		if len(data) >= 32 {
			lat := binary.LittleEndian.Uint64(data[16:24])
			lon := binary.LittleEndian.Uint64(data[24:32])
			result += fmt.Sprintf(" lat_raw=0x%x lon_raw=0x%x", lat, lon)
		}

		return result
	}

	return string(data)
}

func getDNSConfig(data []byte) string {
	if len(data) == 0 {
		return "Empty DNS Configuration data"
	}

	// DNS configurations often contain string data
	dnsInfo := make([]string, 0)
	offset := 0

	// Try to extract null-terminated strings from the data
	for offset < len(data) {
		// Look for null-terminated strings
		end := offset
		for end < len(data) && data[end] != 0 {
			end++
		}

		if end > offset {
			str := string(data[offset:end])
			// Check if this looks like a DNS server or domain
			if strings.Contains(str, ".") || strings.Contains(str, ":") {
				dnsInfo = append(dnsInfo, str)
			}
		}

		offset = end + 1

		// Safety limit
		if len(dnsInfo) >= 10 {
			break
		}
	}

	if len(dnsInfo) > 0 {
		return fmt.Sprintf("DNS Configuration: servers/domains=%s", strings.Join(dnsInfo, ", "))
	}

	return fmt.Sprintf("DNS Configuration data (%d bytes): %x", len(data), data)
}

func getNetworkInterface(data []byte) string {
	if len(data) == 0 {
		return "Empty Network information data"
	}

	// Network interface data often contains interface names and addresses
	interfaces := make([]string, 0)
	offset := 0

	// Try to extract interface information
	for offset < len(data) {
		// Look for null-terminated strings
		end := offset
		for end < len(data) && data[end] != 0 {
			end++
		}

		if end > offset {
			str := string(data[offset:end])
			// Check if this looks like an interface name (en0, lo0, etc.) or IP address
			if (len(str) >= 2 && len(str) <= 16) &&
				(strings.HasPrefix(str, "en") || strings.HasPrefix(str, "lo") ||
					strings.HasPrefix(str, "wlan") || strings.Contains(str, ".") || strings.Contains(str, ":")) {
				interfaces = append(interfaces, str)
			}
		}

		offset = end + 1

		// Safety limit
		if len(interfaces) >= 20 {
			break
		}
	}

	if len(interfaces) > 0 {
		return fmt.Sprintf("Network information: interfaces=%s", strings.Join(interfaces, ", "))
	}

	return fmt.Sprintf("Network information data (%d bytes): %x", len(data), data)
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

	// Add statedump-specific attributes
	entry.Subsystem = "com.apple.statedump"
	entry.Category = statedump.TitleName

	// Set additional statedump metadata as attributes if needed
	// This can be used by consumers to filter or process statedumps specifically
}

// StatedumpCollection represents a collection of statedumps loaded from tracev3 files
type StatedumpCollection struct {
	Statedumps []StateDump
	BootUUID   string
	Timestamp  uint64
}

// LoadStatedumpsFromTraceV3 loads all statedumps from a tracev3 file
// This function specifically targets statedump chunks (0x6003) for extraction
func LoadStatedumpsFromTraceV3(data []byte) (*StatedumpCollection, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty tracev3 data")
	}

	// Parse the header first to get boot UUID and timing info
	header, headerSize, err := ParseTraceV3Header(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tracev3 header: %w", err)
	}

	collection := &StatedumpCollection{
		Statedumps: make([]StateDump, 0),
		BootUUID:   header.BootUUID,
		Timestamp:  header.ContinuousTime,
	}

	// Skip header and parse data section looking specifically for statedump chunks
	remainingData := data[headerSize:]
	offset := 0
	chunkPreambleSize := 16

	for offset < len(remainingData) {
		// Need at least 16 bytes for preamble
		if offset+chunkPreambleSize > len(remainingData) {
			break
		}

		// Parse preamble
		chunkTag := binary.LittleEndian.Uint32(remainingData[offset:])
		chunkDataSize := binary.LittleEndian.Uint64(remainingData[offset+8:])

		// Validate chunk data size
		if chunkDataSize == 0 || chunkDataSize > uint64(len(remainingData)) {
			offset += 4
			continue
		}

		totalChunkSize := chunkPreambleSize + int(chunkDataSize)
		if offset+totalChunkSize > len(remainingData) {
			break
		}

		// Check if this is a statedump chunk (0x6003)
		if chunkTag == 0x6003 {
			// Extract the statedump data (skip the 16-byte preamble)
			statedumpData := remainingData[offset+chunkPreambleSize : offset+totalChunkSize]

			statedump, err := ParseStateDump(statedumpData)
			if err == nil {
				collection.Statedumps = append(collection.Statedumps, *statedump)
			}
		}

		// Move to next chunk with 8-byte alignment padding
		offset += totalChunkSize
		paddingBytes := paddingSize8(chunkDataSize)
		offset += int(paddingBytes)

		// Safety limit
		if len(collection.Statedumps) >= 1000 {
			break
		}
	}

	return collection, nil
}

// GetStatedumpsByType filters statedumps by their data type
func (sc *StatedumpCollection) GetStatedumpsByType(dataType uint32) []StateDump {
	var filtered []StateDump
	for _, sd := range sc.Statedumps {
		if sd.UnknownDataType == dataType {
			filtered = append(filtered, sd)
		}
	}
	return filtered
}

// GetStatedumpsByTitle filters statedumps by their title name
func (sc *StatedumpCollection) GetStatedumpsByTitle(title string) []StateDump {
	var filtered []StateDump
	for _, sd := range sc.Statedumps {
		if sd.TitleName == title {
			filtered = append(filtered, sd)
		}
	}
	return filtered
}

// GetPlistStatedumps returns all statedumps containing plist data (type 1)
func (sc *StatedumpCollection) GetPlistStatedumps() []StateDump {
	return sc.GetStatedumpsByType(1)
}

// GetCustomObjectStatedumps returns all statedumps containing custom objects (type 3)
func (sc *StatedumpCollection) GetCustomObjectStatedumps() []StateDump {
	return sc.GetStatedumpsByType(3)
}
