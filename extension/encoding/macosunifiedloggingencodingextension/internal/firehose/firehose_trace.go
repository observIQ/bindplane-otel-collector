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
	"slices"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/models"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
)

// Trace represents a parsed firehose trace entry
type Trace struct {
	UnknownPCID uint32
	MessageData ItemData
}

const minimumMessageSize = 4

// ParseFirehoseTrace parses a firehose trace entry
func ParseFirehoseTrace(data []byte) (Trace, []byte, error) {
	firehoseTrace := Trace{}
	var err error

	data, unknownPCID, err := helpers.Take(data, 4)
	if err != nil {
		return firehoseTrace, data, err
	}
	firehoseTrace.UnknownPCID = binary.LittleEndian.Uint32(unknownPCID)

	if len(data) < minimumMessageSize {
		data, _, err = helpers.Take(data, len(data))
		if err != nil {
			return firehoseTrace, data, err
		}
		return firehoseTrace, data, nil
	}

	// The rest of the trace log entry appears to be related to log message values
	// But the data is stored differently from other log entries
	// The data appears to be stored backwards? Ex: Data value, Data size, number of data entries, instead normal: number of data entries, data size, data value
	slices.Reverse(data)
	data, message, err := GetMessage(data)
	if err != nil {
		return firehoseTrace, data, err
	}
	firehoseTrace.MessageData = message

	return firehoseTrace, data, nil
}

// GetMessage gets the message data for a firehose trace entry
func GetMessage(data []byte) ([]byte, ItemData, error) {
	itemData := ItemData{}

	if len(data) < minimumMessageSize {
		return data, itemData, nil
	}

	remainingData, entries, err := helpers.Take(data, 1)
	if err != nil {
		return data, itemData, err
	}
	count := 0
	sizesCount := []uint8{}
	for count < int(entries[0]) {
		remainder, size, err := helpers.Take(remainingData, 1)
		if err != nil {
			return data, itemData, err
		}
		sizesCount = append(sizesCount, size[0])
		count++
		remainingData = remainder
	}

	for _, size := range sizesCount {
		itemInfo := ItemInfo{}
		remainder, messageData, err := helpers.Take(remainingData, int(size))
		if err != nil {
			return data, itemData, err
		}
		switch size {
		case 1:
			itemInfo.MessageStrings = fmt.Sprintf("%d", messageData[0])
		case 2:
			itemInfo.MessageStrings = fmt.Sprintf("%d", binary.BigEndian.Uint16(messageData))
		case 4:
			itemInfo.MessageStrings = fmt.Sprintf("%d", binary.BigEndian.Uint32(messageData))
		case 8:
			itemInfo.MessageStrings = fmt.Sprintf("%d", binary.BigEndian.Uint64(messageData))
		default:
			// TODO: warn
			itemInfo.MessageStrings = fmt.Sprintf("unknown size: %d", messageData[0])
		}
		itemData.ItemInfo = append(itemData.ItemInfo, itemInfo)
		remainingData = remainder
	}

	// Reverse the data back to expected format
	slices.Reverse(itemData.ItemInfo)

	return remainingData, itemData, nil
}

// GetFirehoseTraceStrings gets the message data for a firehose trace entry
func GetFirehoseTraceStrings(provider *uuidtext.CacheProvider, stringOffset uint64, firstProcID uint64, secondProcID uint32, catalogs *models.CatalogChunk) (models.MessageData, error) {
	messageData, err := ExtractFormatStrings(provider, stringOffset, firstProcID, secondProcID, catalogs, 0)
	if err != nil {
		return models.MessageData{}, err
	}
	return messageData, nil
}
