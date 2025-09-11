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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
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

// func getFirehoseSignpost(signpost Signpost) (Signpost, []byte) {

// }
