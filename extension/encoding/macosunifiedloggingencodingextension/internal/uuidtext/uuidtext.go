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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
)

// UUIDText represents a parsed UUID text file containing format strings
type UUIDText struct {
	UUID                string
	Signature           uint32
	UnknownMajorVersion uint32
	UnknownMinorVersion uint32
	NumberEntries       uint32
	EntryDescriptors    []Entry
	FooterData          []byte // Collection of strings containing format strings
}

// Entry represents an entry descriptor in a UUID text file
type Entry struct {
	RangeStartOffset uint32
	EntrySize        uint32
}

// ParseUUIDText parses a UUID text file and returns a UUIDText struct
func ParseUUIDText(data []byte) (*UUIDText, error) {
	uuidText := &UUIDText{}
	expectedSignature := uint32(0x66778899)
	var signature []byte
	var unknownMajorVersion []byte
	var unknownMinorVersion []byte
	var numberEntries []byte
	var entrySize []byte
	var rangeStartOffset []byte
	var err error

	data, signature, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}
	if binary.LittleEndian.Uint32(signature) != expectedSignature {
		return nil, fmt.Errorf("invalid UUID text signature: expected 0x%x, got 0x%x", expectedSignature, signature)
	}
	uuidText.Signature = binary.LittleEndian.Uint32(signature)

	data, unknownMajorVersion, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read unknown major version: %w", err)
	}
	data, unknownMinorVersion, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read unknown minor version: %w", err)
	}
	data, numberEntries, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read number entries: %w", err)
	}

	uuidText.UnknownMajorVersion = binary.LittleEndian.Uint32(unknownMajorVersion)
	uuidText.UnknownMinorVersion = binary.LittleEndian.Uint32(unknownMinorVersion)
	uuidText.NumberEntries = binary.LittleEndian.Uint32(numberEntries)

	uuidText.EntryDescriptors = make([]Entry, uuidText.NumberEntries)
	for i := 0; i < int(uuidText.NumberEntries); i++ {
		data, rangeStartOffset, err = helpers.Take(data, 4)
		if err != nil {
			return nil, fmt.Errorf("failed to read range start offset: %w", err)
		}
		data, entrySize, err = helpers.Take(data, 4)
		if err != nil {
			return nil, fmt.Errorf("failed to read entry size: %w", err)
		}
		entryData := Entry{
			RangeStartOffset: binary.LittleEndian.Uint32(rangeStartOffset),
			EntrySize:        binary.LittleEndian.Uint32(entrySize),
		}
		uuidText.EntryDescriptors[i] = entryData
	}
	uuidText.FooterData = data

	return uuidText, nil
}
