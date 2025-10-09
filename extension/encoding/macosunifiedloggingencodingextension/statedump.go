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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
)

// StatedumpChunk represents a Statedump log entry (also known as a "stateEvent" type of log entry)
type StatedumpChunk struct {
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
	UnknownDataType uint32 // 1 = plist, 2 = protocol buffer(?), 3 = custom object
	UnknownDataSize uint32 // Size of statedump data
	DecoderLibrary  string
	DecoderType     string
	TitleName       string
	Data            []uint8
}

// ParseStateDump parses a Statedump log entry. Statedumps are special log entries that may contain a plist file, custom object, or protocol buffer
func ParseStateDump(data []byte) (*StatedumpChunk, error) {
	var err error
	if len(data) < 72 { // Minimum size for basic structure
		return nil, fmt.Errorf("statedump data too small: %d bytes", len(data))
	}

	reader := bytes.NewReader(data)
	result := &StatedumpChunk{}

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
		if result.DecoderLibrary, err = helpers.ExtractString(libraryData); err != nil {
			return nil, fmt.Errorf("failed to extract library data: %w", err)
		}

		typeData := make([]byte, stringSize)
		if _, err := reader.Read(typeData); err != nil {
			return nil, fmt.Errorf("failed to read type data: %w", err)
		}
		if result.DecoderType, err = helpers.ExtractString(typeData); err != nil {
			return nil, fmt.Errorf("failed to extract type data: %w", err)
		}
	}

	// Read title data
	titleData := make([]byte, stringSize)
	if _, err := reader.Read(titleData); err != nil {
		return nil, fmt.Errorf("failed to read title data: %w", err)
	}
	result.TitleName, err = helpers.ExtractString(titleData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract title data: %w", err)
	}

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

	// Format as uppercase hex string without dashes using big-endian for consistency with other UUID parsing
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
		uuidBytes[0], uuidBytes[1], uuidBytes[2], uuidBytes[3],
		uuidBytes[4], uuidBytes[5], uuidBytes[6], uuidBytes[7],
		uuidBytes[8], uuidBytes[9], uuidBytes[10], uuidBytes[11],
		uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])
}

// encodeStandard provides hex encoding for unsupported objects
func encodeStandard(data []byte) string {
	return hex.EncodeToString(data)
}

func getClientManagerStateTrackerData(data []byte) string {
	if len(data) == 0 {
		return "Empty CLClientManagerStateTracker data"
	}

	if len(data) >= 8 {
		// Parse first 8 bytes as count and flags
		count := binary.LittleEndian.Uint32(data[0:4])
		flags := binary.LittleEndian.Uint32(data[4:8])
		return fmt.Sprintf("CLClientManagerStateTracker: count=%d flags=0x%x data_size=%d", count, flags, len(data))
	}

	return fmt.Sprintf("CLClientManagerStateTracker data (%d bytes): %x", len(data), data)
}

func getDaemonStatusStateTrackerData(data []byte) string {
	if len(data) == 0 {
		return "Empty CLDaemonStatusStateTracker data"
	}

	return fmt.Sprintf("CLDaemonStatusStateTracker data (%d bytes): %x", len(data), data)
}

func getLocationTrackerState(data []byte) string {
	if len(data) == 0 {
		return "Empty CLLocationManagerStateTracker data"
	}

	if len(data) >= 16 {
		// Parse location manager state structure with status, accuracy, and timestamp
		status := binary.LittleEndian.Uint32(data[0:4])
		accuracy := binary.LittleEndian.Uint32(data[4:8])
		timestamp := binary.LittleEndian.Uint64(data[8:16])

		result := fmt.Sprintf("CLLocationManagerStateTracker: status=0x%x accuracy=%d timestamp=%d", status, accuracy, timestamp)

		// Extract additional location data if available
		if len(data) >= 32 {
			lat := binary.LittleEndian.Uint64(data[16:24])
			lon := binary.LittleEndian.Uint64(data[24:32])
			result += fmt.Sprintf(" lat_raw=0x%x lon_raw=0x%x", lat, lon)
		}

		return result
	}

	return fmt.Sprintf("CLLocationManagerStateTracker data (%d bytes): %x", len(data), data)
}

func getDNSConfig(data []byte) string {
	if len(data) == 0 {
		return "Empty DNS Configuration data"
	}

	dnsInfo := make([]string, 0)
	offset := 0

	// Extract null-terminated strings from DNS configuration data
	for offset < len(data) {
		// Look for null-terminated strings
		end := offset
		for end < len(data) && data[end] != 0 {
			end++
		}

		if end > offset {
			str := string(data[offset:end])
			// Filter for DNS server addresses or domain names
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

	interfaces := make([]string, 0)
	offset := 0

	// Extract interface names and addresses from network data
	for offset < len(data) {
		// Look for null-terminated strings
		end := offset
		for end < len(data) && data[end] != 0 {
			end++
		}

		if end > offset {
			str := string(data[offset:end])
			// Filter for interface names (en0, lo0, wlan) or IP addresses
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

// StatedumpCollection represents a collection of statedumps loaded from tracev3 files
type StatedumpCollection struct {
	Statedumps []StatedumpChunk
	BootUUID   string
	Timestamp  uint64
}
