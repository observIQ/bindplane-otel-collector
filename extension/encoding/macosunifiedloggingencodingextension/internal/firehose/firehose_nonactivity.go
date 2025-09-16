// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package firehose

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
)

type FirehoseNonActivity struct {
	UnknownActivityID       uint32
	UnknownSentinel         uint32
	PrivateStringsOffset    uint16
	PrivateStringsSize      uint16
	UnknownMessageStringRef uint32
	SubsystemValue          uint16
	TtlValue                uint8
	DataRefValue            uint32
	UnknownPCID             uint32
	FirehoseFormatters      FirehoseFormatters
}

func ParseFirehoseNonActivity(data []byte, flags uint16) (FirehoseNonActivity, []byte) {
	var nonActivity FirehoseNonActivity
	activityIDCurrent := uint16(0x1) // has_current_aid flag
	if (flags & activityIDCurrent) != 0 {
		firehoseData, unknownActivityID, _ := utils.Take(data, 4)
		firehoseData, unknownSentinel, _ := utils.Take(firehoseData, 4)
		nonActivity.UnknownActivityID = binary.LittleEndian.Uint32(unknownActivityID)
		nonActivity.UnknownSentinel = binary.LittleEndian.Uint32(unknownSentinel)
		data = firehoseData
	}

	privateStringRange := uint16(0x100) // has_private_data flag
	if (flags & privateStringRange) != 0 {
		firehoseData, privateStringsOffset, _ := utils.Take(data, 2)
		firehoseData, privateStringsSize, _ := utils.Take(firehoseData, 2)
		nonActivity.PrivateStringsOffset = binary.LittleEndian.Uint16(privateStringsOffset)
		nonActivity.PrivateStringsSize = binary.LittleEndian.Uint16(privateStringsSize)
		data = firehoseData
	}

	data, unknownPCID, _ := utils.Take(data, 4)
	nonActivity.UnknownPCID = binary.LittleEndian.Uint32(unknownPCID)

	formatters, data := firehoseFormatterFlags(data, flags)
	nonActivity.FirehoseFormatters = formatters

	subsystem := uint16(0x200) // has_subsystem flag
	if (flags & subsystem) != 0 {
		firehoseData, subsystemValue, _ := utils.Take(data, 2)
		nonActivity.SubsystemValue = binary.LittleEndian.Uint16(subsystemValue)
		data = firehoseData
	}

	ttl := uint16(0x400) // has_rules flag
	if (flags & ttl) != 0 {
		firehoseData, ttlValue, _ := utils.Take(data, 1)
		nonActivity.TtlValue = ttlValue[0]
		data = firehoseData
	}

	dataRef := uint16(0x800) // has_oversize flag
	if (flags & dataRef) != 0 {
		firehoseData, dataRefValue, _ := utils.Take(data, 4)
		nonActivity.DataRefValue = binary.LittleEndian.Uint32(dataRefValue)
		data = firehoseData
	}

	return nonActivity, data
}

func GetFirehoseNonActivityStrings(
	firehose FirehoseNonActivity,
	provider *uuidtext.CacheProvider,
	stringOffset uint64,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *types.CatalogChunk,
) (types.MessageData, error) {
	if firehose.FirehoseFormatters.SharedCache || firehose.FirehoseFormatters.LargeSharedCache != 0 {
		if firehose.FirehoseFormatters.HasLargeOffset != 0 {
			largeOffset := firehose.FirehoseFormatters.HasLargeOffset
			var extraOffsetValue string
			// large_shared_cache should be double the value of has_large_offset
			// Ex: has_large_offset = 1, large_shared_cache = 2
			// If the value do not match then there is an issue with shared string offset
			// Can recover by using large_shared_cache
			// Apple/log records this as an error: "error: ~~> <Invalid shared cache code pointer offset>"
			// But is still able to get string formatter
			if largeOffset != firehose.FirehoseFormatters.LargeSharedCache/2 && !firehose.FirehoseFormatters.SharedCache {
				largeOffset = firehose.FirehoseFormatters.LargeSharedCache / 2
				extraOffsetValue = fmt.Sprintf("%x%x", largeOffset, stringOffset)
			} else if firehose.FirehoseFormatters.SharedCache {
				largeOffset = 8
				addOffset := uint64(0x10000000) * uint64(largeOffset)
				extraOffsetValue = fmt.Sprintf("%x", addOffset+stringOffset)
			} else {
				extraOffsetValue = fmt.Sprintf("%x%x", largeOffset, stringOffset)
			}

			extraOffsetValueResult, err := strconv.ParseUint(extraOffsetValue, 16, 64)
			if err != nil {
				return types.MessageData{}, fmt.Errorf("failed to get shared string offset to format string for non-activity firehose entry: %w", err)
			}
			return ExtractSharedStrings(provider, uint64(extraOffsetValueResult), firstProcID, secondProcID, catalogs, stringOffset)
		}

		return ExtractSharedStrings(provider, stringOffset, firstProcID, secondProcID, catalogs, stringOffset)
	}

	if firehose.FirehoseFormatters.Absolute {
		extraOffsetValue := fmt.Sprintf("%x%x", firehose.FirehoseFormatters.MainExeAltIndex, firehose.UnknownPCID)
		extraOffsetValueResult, err := strconv.ParseUint(extraOffsetValue, 16, 64)
		if err != nil {
			return types.MessageData{}, fmt.Errorf("failed to get absolute offset to format string for non-activity firehose entry: %w", err)
		}
		return ExtractAbsoluteStrings(provider, extraOffsetValueResult, stringOffset, firstProcID, secondProcID, catalogs, stringOffset)
	}

	if firehose.FirehoseFormatters.UUIDRelative != "" {
		return ExtractAltUUIDStrings(provider, stringOffset, firehose.FirehoseFormatters.UUIDRelative, firstProcID, secondProcID, catalogs, stringOffset)
	}

	return ExtractFormatStrings(provider, stringOffset, firstProcID, secondProcID, catalogs, stringOffset)
}
