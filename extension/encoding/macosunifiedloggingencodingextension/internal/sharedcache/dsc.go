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

package sharedcache // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/sharedcache"

import (
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
)

// Strings represents parsed DSC (shared cache) string data
type Strings struct {
	Signature    uint32
	MajorVersion uint16
	MinorVersion uint16
	NumberRanges uint32
	NumberUUIDs  uint32
	DSCUUID      string
	UUIDs        []UUIDDescriptor
	Ranges       []RangeDescriptor
}

// RangeDescriptor represents a string range in the shared cache
type RangeDescriptor struct {
	RangeOffset      uint64 // In Major version 2 (Monterey+) this is 8 bytes, in version 1 (up to Big Sur) its 4 bytes
	DataOffset       uint32
	RangeSize        uint32
	UnknownUUIDIndex uint64 // Unknown value, added in Major version: 2. Appears to be UUID index. In version 1 the index is 4 bytes and is at the start of the range descriptor
	Strings          []byte
}

// UUIDDescriptor represents a UUID entry in the shared cache
type UUIDDescriptor struct {
	TextOffset uint64 // Size appears to be 8 bytes in Major version: 2. 4 bytes in Major Version 1
	TextSize   uint32
	UUID       string
	PathOffset uint32
	PathString string // Not part of format
}

// DSCCache stores parsed DSC files for shared string resolution
var DSCCache = make(map[string]*Strings)

// ParseDSC parses a DSC (shared cache) file containing shared format strings
func ParseDSC(data []byte, uuid string) (*Strings, error) {
	input, signature, _ := utils.Take(data, 4)
	if binary.LittleEndian.Uint32(signature) != 0x64736368 {
		return nil, fmt.Errorf("invalid DSC signature: expected 0x%x, got 0x%x", 0x64736368, binary.LittleEndian.Uint32(signature))
	}

	sharedCacheStrings := &Strings{
		Signature: binary.LittleEndian.Uint32(signature),
	}

	input, major, _ := utils.Take(input, 2)
	input, minor, _ := utils.Take(input, 2)
	input, numberRanges, _ := utils.Take(input, 4)
	input, numberUUIDs, _ := utils.Take(input, 4)

	sharedCacheStrings.MajorVersion = binary.LittleEndian.Uint16(major)
	sharedCacheStrings.MinorVersion = binary.LittleEndian.Uint16(minor)
	sharedCacheStrings.NumberRanges = binary.LittleEndian.Uint32(numberRanges)
	sharedCacheStrings.NumberUUIDs = binary.LittleEndian.Uint32(numberUUIDs)

	for i := 0; i < int(sharedCacheStrings.NumberRanges); i++ {
		remainingInput, rangeData, err := getRanges(input, &sharedCacheStrings.MajorVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to get ranges while parsing DSC: %w", err)
		}
		sharedCacheStrings.Ranges = append(sharedCacheStrings.Ranges, rangeData)
		input = remainingInput
	}

	for i := 0; i < int(sharedCacheStrings.NumberUUIDs); i++ {
		remainingInput, uuidData, err := getUUIDs(input, &sharedCacheStrings.MajorVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to get UUIDs while parsing DSC: %w", err)
		}
		sharedCacheStrings.UUIDs = append(sharedCacheStrings.UUIDs, uuidData)
		input = remainingInput
	}

	for _, uuid := range sharedCacheStrings.UUIDs {
		pathString, err := getPaths(input, uuid.PathOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to get paths while parsing DSC: %w", err)
		}
		uuid.PathString = pathString
	}

	for _, rangeData := range sharedCacheStrings.Ranges {
		strings, err := getEntryStrings(input, rangeData.DataOffset, rangeData.RangeSize)
		if err != nil {
			return nil, fmt.Errorf("failed to get strings while parsing DSC: %w", err)
		}
		rangeData.Strings = strings
	}

	return sharedCacheStrings, nil
}

// ExtractSharedString extracts a format string from shared cache using the given offset
func (d *Strings) ExtractSharedString(stringOffset uint64) (types.MessageData, error) {
	messageData := types.MessageData{}

	// Handle dynamic formatters (offset with high bit set means "%s")
	if stringOffset&0x80000000 != 0 {
		messageData.FormatString = "%s"
		if len(d.UUIDs) > 0 {
			messageData.Library = d.UUIDs[0].PathString
			messageData.LibraryUUID = d.UUIDs[0].UUID
		}
		return messageData, nil
	}

	// Search through ranges to find the correct string
	for _, r := range d.Ranges {
		if stringOffset >= r.RangeOffset && stringOffset < (r.RangeOffset+uint64(r.RangeSize)) {
			offset := stringOffset - r.RangeOffset

			if int(offset) >= len(r.Strings) {
				continue
			}

			// Extract the format string
			startIdx := int(offset)
			endIdx := startIdx
			for endIdx < len(r.Strings) && r.Strings[endIdx] != 0 {
				endIdx++
			}

			if endIdx > startIdx {
				messageData.FormatString = string(r.Strings[startIdx:endIdx])

				// Get library information from UUID
				if int(r.UnknownUUIDIndex) < len(d.UUIDs) {
					messageData.Library = d.UUIDs[r.UnknownUUIDIndex].PathString
					messageData.LibraryUUID = d.UUIDs[r.UnknownUUIDIndex].UUID
				}

				return messageData, nil
			}
		}
	}

	return messageData, fmt.Errorf("shared string not found at offset %d", stringOffset)
}

func getRanges(input []byte, major *uint16) ([]byte, RangeDescriptor, error) {
	versionNumber := uint16(2)
	rangeData := RangeDescriptor{}

	if major == &versionNumber {
		remainingInput, valueRangeOffset, err := utils.Take(input, 8)
		if err != nil {
			return nil, rangeData, err
		}
		rangeData.RangeOffset = binary.LittleEndian.Uint64(valueRangeOffset)
		input = remainingInput
	} else {
		input, uuidDescriptorIndex, err := utils.Take(input, 4)
		if err != nil {
			return nil, rangeData, err
		}
		rangeData.UnknownUUIDIndex = uint64(binary.LittleEndian.Uint32(uuidDescriptorIndex))

		input, valueRangeOffset, err := utils.Take(input, 4)
		if err != nil {
			return nil, rangeData, err
		}
		rangeData.RangeOffset = uint64(binary.LittleEndian.Uint32(valueRangeOffset))
	}

	input, dataOffset, err := utils.Take(input, 4)
	if err != nil {
		return nil, rangeData, err
	}
	rangeData.DataOffset = binary.LittleEndian.Uint32(dataOffset)

	input, rangeSize, err := utils.Take(input, 4)
	if err != nil {
		return nil, rangeData, err
	}
	rangeData.RangeSize = binary.LittleEndian.Uint32(rangeSize)

	if major == &versionNumber {
		remainingInput, unknown, err := utils.Take(input, 8)
		if err != nil {
			return nil, rangeData, err
		}
		rangeData.UnknownUUIDIndex = binary.LittleEndian.Uint64(unknown)
		input = remainingInput
	}

	return input, rangeData, nil
}

func getUUIDs(input []byte, major *uint16) ([]byte, UUIDDescriptor, error) {
	versionNumber := uint16(2)
	uuidData := UUIDDescriptor{}

	if major == &versionNumber {
		remainingInput, valueTextOffset, err := utils.Take(input, 8)
		if err != nil {
			return nil, uuidData, err
		}
		uuidData.TextOffset = binary.LittleEndian.Uint64(valueTextOffset)
		input = remainingInput
	} else {
		remainingInput, valueTextOffset, err := utils.Take(input, 4)
		if err != nil {
			return nil, uuidData, err
		}
		uuidData.TextOffset = uint64(binary.LittleEndian.Uint32(valueTextOffset))
		input = remainingInput
	}

	input, valueTextSize, err := utils.Take(input, 4)
	if err != nil {
		return nil, uuidData, err
	}
	uuidData.TextSize = binary.LittleEndian.Uint32(valueTextSize)

	input, valueUUID, err := utils.Take(input, 16)
	if err != nil {
		return nil, uuidData, err
	}
	uuidData.UUID = fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
		valueUUID[0], valueUUID[1], valueUUID[2], valueUUID[3], valueUUID[4], valueUUID[5], valueUUID[6], valueUUID[7],
		valueUUID[8], valueUUID[9], valueUUID[10], valueUUID[11], valueUUID[12], valueUUID[13], valueUUID[14], valueUUID[15])

	input, valuePathOffset, err := utils.Take(input, 4)
	if err != nil {
		return nil, uuidData, err
	}
	uuidData.PathOffset = binary.LittleEndian.Uint32(valuePathOffset)

	return input, uuidData, nil
}

func getPaths(input []byte, pathOffset uint32) (string, error) {
	pathData, _, err := utils.Take(input, int(pathOffset))
	if err != nil {
		return "", err
	}
	pathString, err := utils.ExtractString(pathData)
	if err != nil {
		return "", err
	}
	return pathString, nil
}

// getEntryStrings gets the base log entry strings after the ranges and UUIDs are parsed out
func getEntryStrings(input []byte, stringOffset uint32, stringRange uint32) ([]byte, error) {
	stringsData, _, err := utils.Take(input, int(stringOffset))
	if err != nil {
		return nil, err
	}
	_, strings, err := utils.Take(stringsData, int(stringRange))
	if err != nil {
		return nil, err
	}
	return strings, nil
}
