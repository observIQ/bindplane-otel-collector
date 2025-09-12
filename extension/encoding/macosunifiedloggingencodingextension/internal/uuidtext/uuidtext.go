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

// UUIDText represents a parsed UUID text file containing format strings
type UUIDText struct {
	UUID                string
	Signature           uint32
	UnknownMajorVersion uint32
	UnknownMinorVersion uint32
	NumberEntries       uint32
	EntryDescriptors    []UUIDTextEntry
	FooterData          []byte // Collection of strings containing format strings
}

// UUIDTextEntry represents an entry descriptor in a UUID text file
type UUIDTextEntry struct {
	RangeStartOffset uint32
	EntrySize        uint32
}

func ParseUUIDText(data []byte, uuid string) (*UUIDText, error) {
	uuidText := &UUIDText{}
	expectedSignature := uint32(0x66778899)

	data, signature, _ := utils.Take(data, 4)
	if binary.LittleEndian.Uint32(signature) != expectedSignature {
		return nil, fmt.Errorf("invalid UUID text signature: expected 0x%x, got 0x%x", expectedSignature, signature)
	}
	uuidText.Signature = binary.LittleEndian.Uint32(signature)

	data, unknownMajorVersion, _ := utils.Take(data, 4)
	data, unknownMinorVersion, _ := utils.Take(data, 4)
	data, numberEntries, _ := utils.Take(data, 4)

	uuidText.UnknownMajorVersion = binary.LittleEndian.Uint32(unknownMajorVersion)
	uuidText.UnknownMinorVersion = binary.LittleEndian.Uint32(unknownMinorVersion)
	uuidText.NumberEntries = binary.LittleEndian.Uint32(numberEntries)

	uuidText.EntryDescriptors = make([]UUIDTextEntry, uuidText.NumberEntries)
	for i := 0; i < int(uuidText.NumberEntries); i++ {
		data, rangeStartOffset, _ := utils.Take(data, 4)
		data, entrySize, _ := utils.Take(data, 4)
		entryData := UUIDTextEntry{
			RangeStartOffset: binary.LittleEndian.Uint32(rangeStartOffset),
			EntrySize:        binary.LittleEndian.Uint32(entrySize),
		}
		uuidText.EntryDescriptors = append(uuidText.EntryDescriptors, entryData)
	}
	uuidText.FooterData = data

	return uuidText, nil
}
