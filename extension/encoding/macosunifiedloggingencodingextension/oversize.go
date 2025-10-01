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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
)

// OversizeChunk represents a parsed oversize chunk
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
	messageItems    firehose.ItemData
}

// ParseOversizeChunk parses an oversize chunk
// Returns the parsed oversize chunk and the remaining data
func ParseOversizeChunk(data []byte) (OversizeChunk, []byte, error) {
	var oversizeResult OversizeChunk

	var chunkTag, chunkSubTag, chunkDataSize, firstProcID, secondProcID,
		ttl, unknownReserved, continuousTime, dataRefIndex, publicDataSize, privateDataSize []byte

	var err error

	data, chunkTag, err = helpers.Take(data, 4)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read chunk tag: %w", err)
	}
	data, chunkSubTag, err = helpers.Take(data, 4)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read chunk sub tag: %w", err)
	}
	data, chunkDataSize, err = helpers.Take(data, 8)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read chunk data size: %w", err)
	}
	data, firstProcID, err = helpers.Take(data, 8)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read first proc ID: %w", err)
	}
	data, secondProcID, err = helpers.Take(data, 4)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read second proc ID: %w", err)
	}
	data, ttl, err = helpers.Take(data, 1)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read TTL: %w", err)
	}
	data, unknownReserved, err = helpers.Take(data, 3)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read unknown reserved: %w", err)
	}
	data, continuousTime, err = helpers.Take(data, 8)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read continuous time: %w", err)
	}
	data, dataRefIndex, err = helpers.Take(data, 4)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read data ref index: %w", err)
	}
	data, publicDataSize, err = helpers.Take(data, 2)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read public data size: %w", err)
	}
	data, privateDataSize, err = helpers.Take(data, 2)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read private data size: %w", err)
	}

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
		// TODO: Log this warning
		// fmt.Printf("Oversize data size greater than Oversize remaining string size. Using remaining string size\n")
		oversizeDataSize = len(data)
	}

	var publicData []byte
	data, publicData, err = helpers.Take(data, oversizeDataSize)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read public data: %w", err)
	}

	var messageData []byte
	messageData, _, err = helpers.Take(publicData, 1)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read message data header: %w", err)
	}

	var itemCount []byte
	messageData, itemCount, err = helpers.Take(messageData, 1)
	if err != nil {
		return oversizeResult, data, fmt.Errorf("failed to read item count: %w", err)
	}
	oversizeItemCount := itemCount[0]

	emptyFlags := uint16(0)
	// Grab all message items from oversize data
	firehoseItemData, oversizePrivateData, err := firehose.ParseFirehoseMessageItems(messageData, oversizeItemCount, emptyFlags)
	if err != nil {
		return oversizeResult, data, err
	}
	_, err = firehose.ParsePrivateData(oversizePrivateData, &firehoseItemData)
	if err != nil {
		return oversizeResult, data, err
	}
	oversizeResult.messageItems = firehoseItemData

	return oversizeResult, data, nil
}

// GetOversizeStrings gets the firehose item info from the oversize log entry based on oversize (data ref) id, first proc id, and second proc id
func GetOversizeStrings(dataRef uint32, firstProcID uint64, secondProcID uint32, oversizeData *[]OversizeChunk) []firehose.ItemInfo {
	messageStrings := []firehose.ItemInfo{}
	for _, oversize := range *oversizeData {
		if oversize.dataRefIndex == dataRef && oversize.firstProcID == firstProcID && oversize.secondProcID == secondProcID {
			for _, message := range oversize.messageItems.ItemInfo {
				oversizeFirehose := firehose.ItemInfo{
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
