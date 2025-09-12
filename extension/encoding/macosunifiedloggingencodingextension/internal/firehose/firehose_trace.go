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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
)

type FirehoseTrace struct {
	UnknownPCID uint32
	MessageData ItemData
}

const minimumMessageSize = 4

func ParseFirehoseTrace(data []byte) (FirehoseTrace, []byte, error) {
	firehoseTrace := FirehoseTrace{}

	data, unknownPCID, _ := utils.Take(data, 4)
	firehoseTrace.UnknownPCID = binary.LittleEndian.Uint32(unknownPCID)

	if len(data) < minimumMessageSize {
		data, _, _ = utils.Take(data, len(data))
		return firehoseTrace, data, nil
	}

	// The rest of the trace log entry appears to be related to log message values
	// But the data is stored differently from other log entries
	// The data appears to be stored backwards? Ex: Data value, Data size, number of data entries, instead normal: number of data entries, data size, data value
	slices.Reverse(data)
	data, message, error := GetMessage(data)
	if error != nil {
		return firehoseTrace, data, error
	}
	firehoseTrace.MessageData = message

	return firehoseTrace, data, nil
}

func GetMessage(data []byte) ([]byte, FirehoseItemData, error) {
	itemData := FirehoseItemData{}

	if len(data) < minimumMessageSize {
		return data, itemData, nil
	}

	remainingData, entries, _ := utils.Take(data, 1)
	count := 0
	sizesCount := []uint8{}
	for count < int(entries[0]) {
		remainder, size, _ := utils.Take(remainingData, 1)
		sizesCount = append(sizesCount, size[0])
		count++
		remainingData = remainder
	}

	for _, size := range sizesCount {
		itemInfo := FirehoseItemInfo{}
		remainder, messageData, _ := utils.Take(remainingData, int(size))
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

// TODO: Uncomment once dsc is updated
// func GetFirehoseTraceStrings(provider FileProvider, stringOffset uint64, firstProcID uint64, secondProcID uint32, catalogs *CatalogChunk) ([]byte, FirehoseItemData, error) {
// 	return ExtractFormatStrings(provider, stringOffset, firstProcID, secondProcID, catalogs, 0)
// }
