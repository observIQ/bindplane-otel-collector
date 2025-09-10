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

package macosunifiedloggingencodingextension // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"

import (
	"encoding/binary"
	"fmt"
)

type OversizeChunk struct {
	chunkTag        uint32
	chunkSubTag     uint32
	chunkDataSize   uint64
	firstProcID     uint64
	secondProcID    uint32
	ttl             uint8
	unknownReserved [3]uint8
	continuousTime  uint64
	dataRefIndex    uint32
	publicDataSize  uint16
	privateDataSize uint16
	messageItems    FirehoseItemData
}

func ParseOversizeChunk(data []byte) (OversizeChunk, []byte) {
	var oversizeResult OversizeChunk

	data, chunkTag, _ := Take(data, 4)
	data, chunkSubTag, _ := Take(data, 4)
	data, chunkDataSize, _ := Take(data, 8)
	data, firstProcID, _ := Take(data, 8)
	data, secondProcID, _ := Take(data, 4)
	data, ttl, _ := Take(data, 1)
	data, unknownReserved, _ := Take(data, 3)
	data, continuousTime, _ := Take(data, 8)
	data, dataRefIndex, _ := Take(data, 4)
	data, publicDataSize, _ := Take(data, 2)
	data, privateDataSize, _ := Take(data, 2)

	oversizeResult.chunkTag = binary.LittleEndian.Uint32(chunkTag)
	oversizeResult.chunkSubTag = binary.LittleEndian.Uint32(chunkSubTag)
	oversizeResult.chunkDataSize = binary.LittleEndian.Uint64(chunkDataSize)
	oversizeResult.firstProcID = binary.LittleEndian.Uint64(firstProcID)
	oversizeResult.secondProcID = binary.LittleEndian.Uint32(secondProcID)
	oversizeResult.ttl = ttl[0]
	copy(oversizeResult.unknownReserved[:], unknownReserved)
	oversizeResult.continuousTime = binary.LittleEndian.Uint64(continuousTime)
	oversizeResult.dataRefIndex = binary.LittleEndian.Uint32(dataRefIndex)
	oversizeResult.publicDataSize = binary.LittleEndian.Uint16(publicDataSize)
	oversizeResult.privateDataSize = binary.LittleEndian.Uint16(privateDataSize)

	oversizeDataSize := int(oversizeResult.publicDataSize + oversizeResult.privateDataSize)
	if oversizeDataSize > len(data) {
		fmt.Printf("Oversize data size greater than Oversize remaining string size. Using remaining string size\n")
		oversizeDataSize = len(data)
	}

	data, publicData, _ := Take(data, oversizeDataSize)
	messageData, _, _ := Take(publicData, 1)
	messageData, itemCount, _ := Take(messageData, 1)
	oversizeItemCount := itemCount[0]

	emptyFlags := uint16(0)
	// Grab all message items from oversize data
	firehoseItemData, oversizePrivateData := ParseFirehoseMessageItems(messageData, oversizeItemCount, emptyFlags)
	_, err := ParsePrivateData(oversizePrivateData, &firehoseItemData)
	if err != nil {
		// TODO: better error handling
		fmt.Printf("Error parsing private data: %v\n", err)
	}
	oversizeResult.messageItems = firehoseItemData

	return oversizeResult, data
}

// get the firehose item info from the oversize log entry based on oversize (data ref) id, first proc id, and second proc id
func GetOversizeStrings(dataRef uint32, firstProcID uint64, secondProcID uint32, oversizeData []*OversizeChunk) []FirehoseItemInfo {
	messageStrings := []FirehoseItemInfo{}
	for _, oversize := range oversizeData {
		if oversize.dataRefIndex == dataRef && oversize.firstProcID == firstProcID && oversize.secondProcID == secondProcID {
			for _, message := range oversize.messageItems.ItemInfo {
				oversizeFirehose := FirehoseItemInfo{
					MessageStrings: message.MessageStrings,
					ItemType:       message.ItemType,
					ItemSize:       message.ItemSize,
				}
				messageStrings = append(messageStrings, oversizeFirehose)
			}
			return messageStrings
		}
	}
	// TODO: Log that we didn't find any oversize data
	return messageStrings
}
