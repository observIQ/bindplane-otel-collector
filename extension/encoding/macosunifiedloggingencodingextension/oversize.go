// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"

import (
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
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
	messageItems    firehose.FirehoseItemData
}

// ParseOversizeChunk parses an oversize chunk
// Returns the parsed oversize chunk and the remaining data
func ParseOversizeChunk(data []byte) (OversizeChunk, []byte) {
	var oversizeResult OversizeChunk

	data, chunkTag, _ := utils.Take(data, 4)
	data, chunkSubTag, _ := utils.Take(data, 4)
	data, chunkDataSize, _ := utils.Take(data, 8)
	data, firstProcID, _ := utils.Take(data, 8)
	data, secondProcID, _ := utils.Take(data, 4)
	data, ttl, _ := utils.Take(data, 1)
	data, unknownReserved, _ := utils.Take(data, 3)
	data, continuousTime, _ := utils.Take(data, 8)
	data, dataRefIndex, _ := utils.Take(data, 4)
	data, publicDataSize, _ := utils.Take(data, 2)
	data, privateDataSize, _ := utils.Take(data, 2)

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

	data, publicData, _ := utils.Take(data, oversizeDataSize)
	messageData, _, _ := utils.Take(publicData, 1)
	messageData, itemCount, _ := utils.Take(messageData, 1)
	oversizeItemCount := itemCount[0]

	emptyFlags := uint16(0)
	// Grab all message items from oversize data
	firehoseItemData, oversizePrivateData := firehose.ParseFirehoseMessageItems(messageData, oversizeItemCount, emptyFlags)
	_, err := firehose.ParsePrivateData(oversizePrivateData, &firehoseItemData)
	if err != nil {
		// TODO: better error handling
		fmt.Printf("Error parsing private data: %v\n", err)
	}
	oversizeResult.messageItems = firehoseItemData

	return oversizeResult, data
}

// GetOversizeStrings gets the firehose item info from the oversize log entry based on oversize (data ref) id, first proc id, and second proc id
func GetOversizeStrings(dataRef uint32, firstProcID uint64, secondProcID uint32, oversizeData []*OversizeChunk) []firehose.FirehoseItemInfo {
	messageStrings := []firehose.FirehoseItemInfo{}
	for _, oversize := range oversizeData {
		if oversize.dataRefIndex == dataRef && oversize.firstProcID == firstProcID && oversize.secondProcID == secondProcID {
			for _, message := range oversize.messageItems.ItemInfo {
				oversizeFirehose := firehose.FirehoseItemInfo{
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
