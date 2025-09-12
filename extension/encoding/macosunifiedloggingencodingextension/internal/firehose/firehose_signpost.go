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

package firehose

import (
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
)

// Signpost represents a parsed firehose signpost entry
type Signpost struct {
	UnknownPCID          uint32
	UnknownActivityID    uint32
	UnknownSentinel      uint32
	Subsystem            uint16
	SignpostID           uint64
	SignpostName         uint32
	PrivateStringsOffset uint16
	PrivateStringsSize   uint16
	TTLValue             uint8
	DataRefValue         uint32
	FirehoseFormatters   FirehoseFormatters
}

// ParseFirehoseSignpost parses a firehose signpost entry
func ParseFirehoseSignpost(data []byte, flags uint16) (Signpost, []byte) {
	signpost := Signpost{}
	var unknownActivityID []byte
	var unknownSentinel []byte
	var privateStringsOffset []byte
	var privateStringsSize []byte
	var subsystem []byte
	var ttlValue []byte
	var dataRefValue []byte
	var signpostName []byte

	activityIDCurrentFlag := uint16(0x1)
	if (flags & activityIDCurrentFlag) != 0 {
		data, unknownActivityID, _ = utils.Take(data, 4)
		signpost.UnknownActivityID = binary.LittleEndian.Uint32(unknownActivityID)
		data, unknownSentinel, _ = utils.Take(data, 4)
		signpost.UnknownSentinel = binary.LittleEndian.Uint32(unknownSentinel)
	}
	privateStringRangeFlag := uint16(0x100)
	if (flags & privateStringRangeFlag) != 0 {
		data, privateStringsOffset, _ = utils.Take(data, 2)
		signpost.PrivateStringsOffset = binary.LittleEndian.Uint16(privateStringsOffset)
		data, privateStringsSize, _ = utils.Take(data, 2)
		signpost.PrivateStringsSize = binary.LittleEndian.Uint16(privateStringsSize)
	}

	data, unknownPCID, _ := utils.Take(data, 4)
	signpost.UnknownPCID = binary.LittleEndian.Uint32(unknownPCID)

	formatters, data := firehoseFormatterFlags(data, flags)
	signpost.FirehoseFormatters = formatters

	subsystemFlag := uint16(0x200)
	if (flags & subsystemFlag) != 0 {
		data, subsystem, _ = utils.Take(data, 2)
		signpost.Subsystem = binary.LittleEndian.Uint16(subsystem)
	}

	data, signpostID, _ := utils.Take(data, 8)
	signpost.SignpostID = binary.LittleEndian.Uint64(signpostID)

	hasRulesFlag := uint16(0x400)
	if (flags & hasRulesFlag) != 0 {
		data, ttlValue, _ = utils.Take(data, 1)
		signpost.TTLValue = ttlValue[0]
	}

	dataRefFlag := uint16(0x800)
	if (flags & dataRefFlag) != 0 {
		data, dataRefValue, _ = utils.Take(data, 4)
		signpost.DataRefValue = binary.LittleEndian.Uint32(dataRefValue)
	}

	hasNameFlag := uint16(0x8000)
	if (flags & hasNameFlag) != 0 {
		data, signpostName, _ = utils.Take(data, 4)
		signpost.SignpostName = binary.LittleEndian.Uint32(signpostName)
		// If the signpost log has large_shared_cache flag
		// Then the signpost name has the same value after as the large_shared_cache
		if signpost.FirehoseFormatters.LargeSharedCache != 0 {
			data, _, _ = utils.Take(data, 2)
		}
	}

	return signpost, data
}

// TODO: Update this to use the new uuidtext package methods
func getFirehoseSignpost(signpost Signpost, provider types.FileProvider, stringOffset uint64, firstProcID uint64, secondProcID uint32, catalogs types.CatalogChunk) (types.MessageData, error) {
	if signpost.FirehoseFormatters.SharedCache || (signpost.FirehoseFormatters.LargeSharedCache != 0 && signpost.FirehoseFormatters.HasLargeOffset != 0) {
		if signpost.FirehoseFormatters.HasLargeOffset != 0 {
			largeOffset := signpost.FirehoseFormatters.HasLargeOffset
			var extraOffsetValue string
			// large_shared_cache should be double the value of has_large_offset
			// Ex: has_large_offset = 1, large_shared_cache = 2
			// If the value do not match then there is an issue with shared string offset
			// Can recover by using large_shared_cache
			// Apple records this as an error: "error: ~~> <Invalid shared cache code pointer offset>"
			//   But is still able to get string formatter
			if largeOffset != signpost.FirehoseFormatters.LargeSharedCache/2 && !signpost.FirehoseFormatters.SharedCache {
				largeOffset = signpost.FirehoseFormatters.LargeSharedCache / 2
				extraOffsetValue = fmt.Sprintf("%x%x", largeOffset, stringOffset)
			} else if signpost.FirehoseFormatters.SharedCache {
				largeOffset = 8
				extraOffsetValue = fmt.Sprintf("%x%x", largeOffset, stringOffset)
			} else {
				extraOffsetValue = fmt.Sprintf("%x%x", largeOffset, stringOffset)
			}
			extraOffsetValueResult, err := uuidtext.ExtractSharedStrings(provider, uint32(extraOffsetValue), firstProcID, secondProcID, catalogs, stringOffset)
			messageData, err := uuidtext.GetSharedFormatString(uint32(extraOffsetValueResult), firstProcID, secondProcID)
			if err != nil {
				return messageData, fmt.Errorf("error getting shared format string: %w", err)
			}
			return messageData, nil
		}
	}
	return types.MessageData{}, nil
}
