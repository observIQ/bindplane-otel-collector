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

package uuidtext

import (
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
)

// SharedCacheStrings represents parsed DSC (shared cache) string data
type SharedCacheStrings struct {
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

const expectedSignature = 0x68736964 // "dsih" in little endian

// DSCCache stores parsed DSC files for shared string resolution
var DSCCache = make(map[string]*SharedCacheStrings)

func ParseDSC(data []byte, uuid string) (*SharedCacheStrings, error) {
	sharedCacheStrings := &SharedCacheStrings{}
	data, signature, _ := utils.Take(data, 4)
	if binary.LittleEndian.Uint32(signature) != expectedSignature {
		return nil, fmt.Errorf("invalid DSC signature: expected 0x%x, got 0x%x", expectedSignature, signature)
	}
	sharedCacheStrings.Signature = binary.LittleEndian.Uint32(signature)
	data, majorVersion, _ := utils.Take(data, 2)
	sharedCacheStrings.MajorVersion = binary.LittleEndian.Uint16(majorVersion)
	data, minorVersion, _ := utils.Take(data, 2)
	sharedCacheStrings.MinorVersion = binary.LittleEndian.Uint16(minorVersion)
	data, numberRanges, _ := utils.Take(data, 4)
	sharedCacheStrings.NumberRanges = binary.LittleEndian.Uint32(numberRanges)
	data, numberUUIDs, _ := utils.Take(data, 4)
	sharedCacheStrings.NumberUUIDs = binary.LittleEndian.Uint32(numberUUIDs)

	for rangeCount := 0; rangeCount < int(sharedCacheStrings.NumberRanges); rangeCount++ {
		rangeData := RangeDescriptor{}
		data, rangeData, _ = getRanges(data, sharedCacheStrings.MajorVersion)
		sharedCacheStrings.Ranges = append(sharedCacheStrings.Ranges, rangeData)
	}

	for uuidCount := 0; uuidCount < int(sharedCacheStrings.NumberUUIDs); uuidCount++ {
		uuidDescriptor := UUIDDescriptor{}
		data, uuidDescriptor, _ = getUUIDs(data, sharedCacheStrings.MajorVersion)
		sharedCacheStrings.UUIDs = append(sharedCacheStrings.UUIDs, uuidDescriptor)
	}

	for i, uuids := range sharedCacheStrings.UUIDs {
		data, sharedCacheStrings.UUIDs[i].PathString, _ = getPaths(data, uuids.PathOffset)
	}

	for i, rangeItem := range sharedCacheStrings.Ranges {
		data, sharedCacheStrings.Ranges[i].Strings, _ = getStrings(data, rangeItem.DataOffset, rangeItem.RangeSize)
	}

	return sharedCacheStrings, nil

}

func getRanges(data []byte, version uint16) ([]byte, RangeDescriptor, error) {
	versionNumber := uint16(2)
	rangeData := RangeDescriptor{}
	dscRangeOffset := []byte{}
	uuidDescriptorIndex := []byte{}
	// Version 2 (Monterey+): range offset is 8 bytes, in version 1 (up to Big Sur) its 4 bytes
	// The uuid index was moved to end
	if version == versionNumber {
		data, dscRangeOffset, _ = utils.Take(data, 8)
		rangeData.RangeOffset = binary.LittleEndian.Uint64(dscRangeOffset)
	} else {
		data, uuidDescriptorIndex, _ = utils.Take(data, 4)
		rangeData.UnknownUUIDIndex = binary.LittleEndian.Uint64(uuidDescriptorIndex)
		data, dscRangeOffset, _ = utils.Take(data, 4)
		rangeData.RangeOffset = binary.LittleEndian.Uint64(dscRangeOffset)
	}

	data, dataOffset, _ := utils.Take(data, 4)
	rangeData.DataOffset = binary.LittleEndian.Uint32(dataOffset)
	data, rangeSize, _ := utils.Take(data, 4)
	rangeData.RangeSize = binary.LittleEndian.Uint32(rangeSize)

	if version == versionNumber {
		unknown := []byte{}
		data, unknown, _ = utils.Take(data, 8)
		rangeData.UnknownUUIDIndex = binary.LittleEndian.Uint64(unknown)
	}

	return data, rangeData, nil
}

func getUUIDs(data []byte, version uint16) ([]byte, UUIDDescriptor, error) {
	uuidDescriptor := UUIDDescriptor{}
	versionNumber := uint16(2)
	textOffset := []byte{}

	if version == versionNumber {
		data, textOffset, _ = utils.Take(data, 8)
		uuidDescriptor.TextOffset = binary.LittleEndian.Uint64(textOffset)
	} else {
		data, textOffset, _ = utils.Take(data, 4)
		uuidDescriptor.TextOffset = binary.LittleEndian.Uint64(textOffset)
	}

	data, textSize, _ := utils.Take(data, 4)
	data, uuid, _ := utils.Take(data, 16)
	data, pathOffset, _ := utils.Take(data, 4)

	uuidDescriptor.TextSize = binary.LittleEndian.Uint32(textSize)
	uuidDescriptor.UUID = fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
		uuid[0], uuid[1], uuid[2], uuid[3], uuid[4], uuid[5], uuid[6], uuid[7], uuid[8], uuid[9], uuid[10], uuid[11], uuid[12], uuid[13], uuid[14], uuid[15])
	uuidDescriptor.PathOffset = binary.LittleEndian.Uint32(pathOffset)

	return data, uuidDescriptor, nil
}

func getPaths(data []byte, pathOffset uint32) ([]byte, string, error) {
	data, offset, _ := utils.Take(data, int(pathOffset))
	path, err := utils.ExtractString(offset)
	if err != nil {
		return data, "", err
	}
	return data, path, nil
}

func getStrings(data []byte, stringOffset uint32, stringRange uint32) ([]byte, []byte, error) {
	data, _, _ = utils.Take(data, int(stringOffset))
	_, strings, _ := utils.Take(data, int(stringRange))
	return data, strings, nil
}
