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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
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

func ParseOversizeChunkV2(data []byte) []*TraceV3Entry {
	var oversizeResult OversizeChunk

	chunkTag, data, _ := Take(data, 4)
	chunkSubTag, data, _ := Take(data, 4)
	chunkDataSize, data, _ := Take(data, 8)
	firstProcID, data, _ := Take(data, 8)
	secondProcID, data, _ := Take(data, 4)
	ttl, data, _ := Take(data, 1)
	unknownReserved, data, _ := Take(data, 3)
	continuousTime, data, _ := Take(data, 8)
	dataRefIndex, data, _ := Take(data, 4)
	publicDataSize, data, _ := Take(data, 2)
	privateDataSize, data, _ := Take(data, 2)

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

	publicData, data, _ := Take(data, oversizeDataSize)
	messageData, _, _ := Take(publicData, 1)
	messageData, itemCount, _ := Take(messageData, 1)
	oversizeItemCount := itemCount[0]

	emptyFlags := 0
	// Grab all message items from oversize data
	oversizePrivateData, firehoseItemData := ParseFirehoseMessageItems(messageData, oversizeItemCount, emptyFlags)
	firehoseItemData, _ = parsePrivateData(oversizePrivateData, firehoseItemData)
	oversizeResult.messageItems = firehoseItemData
}

// ParseOversizeChunk parses an Oversize chunk (0x6002) containing large log entries
// Returns a slice of individual TraceV3Entry objects for each log message found in the oversize data
// Based on rust implementation: uses FirehosePreamble::collect_items approach
func ParseOversizeChunk(data []byte, templateEntry *TraceV3Entry, header *TraceV3Header, timesyncData map[string]*TimesyncBoot) []*TraceV3Entry {
	if len(data) < 48 { // Need at least 48 bytes for oversize header
		templateEntry.Message = fmt.Sprintf("Oversize chunk too small: %d bytes", len(data))
		return []*TraceV3Entry{templateEntry}
	}

	// Parse oversize chunk header (based on rust implementation)
	firstProcID := binary.LittleEndian.Uint64(data[16:24])
	secondProcID := binary.LittleEndian.Uint32(data[24:28])
	ttl := data[28]
	// Skip 3 bytes of unknown_reserved data (bytes 29-31)
	continuousTime := binary.LittleEndian.Uint64(data[32:40])
	dataRefIndex := binary.LittleEndian.Uint32(data[40:44])
	publicDataSize := binary.LittleEndian.Uint16(data[44:46])
	privateDataSize := binary.LittleEndian.Uint16(data[46:48])

	// Calculate total data size for oversize content
	totalDataSize := int(publicDataSize + privateDataSize)
	if len(data) < 48+totalDataSize {
		templateEntry.Message = fmt.Sprintf("Oversize chunk truncated: expected %d bytes, got %d", 48+totalDataSize, len(data))
		return []*TraceV3Entry{templateEntry}
	}

	// Extract the message data section starting at offset 48
	messageData := data[48 : 48+totalDataSize]

	// Parse the message items like firehose (based on rust implementation)
	// The first two bytes indicate structure: first byte unknown, second byte is item count
	if len(messageData) < 2 {
		templateEntry.Message = fmt.Sprintf("Oversize data too small for item header: %d bytes", len(messageData))
		return []*TraceV3Entry{templateEntry}
	}

	// Parse individual log messages from oversize data using rust approach
	// Skip first byte (unknown), second byte is item count, then use firehose item parsing
	itemCount := messageData[1]
	if itemCount == 0 {
		templateEntry.Message = fmt.Sprintf("Oversize chunk: ttl=%d dataRef=%d publicSize=%d privateSize=%d (no items)",
			ttl, dataRefIndex, publicDataSize, privateDataSize)
		return []*TraceV3Entry{templateEntry}
	}

	// Parse firehose items using the same logic as the rust implementation
	items := parseFirehoseItemsFromOversize(messageData[2:], int(itemCount))

	// Convert parsed items to individual log entries
	var entries []*TraceV3Entry
	for _, item := range items {
		// Create entries for items with actual content
		if item.MessageStrings != "" {
			entry := &TraceV3Entry{
				Type:         templateEntry.Type,
				Size:         uint32(item.ItemSize),
				ThreadID:     firstProcID,
				ProcessID:    secondProcID,
				Subsystem:    "com.apple.oversize.decompressed",
				Category:     "", // Empty like rust implementation
				Level:        "Info",
				MessageType:  "Default",
				EventType:    "logEvent",
				ChunkType:    "oversize",
				TimezoneName: templateEntry.TimezoneName,
				Message:      item.MessageStrings,
			}

			// Calculate timestamp using oversize's continuous time
			if timesyncData != nil && header != nil {
				entry.Timestamp = convertMachTimeToUnixNanosWithTimesync(continuousTime, header.BootUUID, 0, timesyncData)
			} else {
				entry.Timestamp = continuousTime
			}

			entries = append(entries, entry)
		}
	}

	// If no entries were created, create a summary entry
	if len(entries) == 0 {
		entry := &TraceV3Entry{
			Type:         templateEntry.Type,
			Size:         uint32(totalDataSize),
			ThreadID:     firstProcID,
			ProcessID:    secondProcID,
			Subsystem:    "com.apple.oversize.decompressed",
			Category:     "",
			Level:        "Info",
			MessageType:  "Default",
			EventType:    "logEvent",
			ChunkType:    "oversize",
			TimezoneName: templateEntry.TimezoneName,
			Message: fmt.Sprintf("Oversize chunk: ttl=%d dataRef=%d publicSize=%d privateSize=%d items=%d (no content extracted)",
				ttl, dataRefIndex, publicDataSize, privateDataSize, itemCount),
		}

		// Calculate timestamp using oversize's continuous time
		if timesyncData != nil && header != nil {
			entry.Timestamp = convertMachTimeToUnixNanosWithTimesync(continuousTime, header.BootUUID, 0, timesyncData)
		} else {
			entry.Timestamp = continuousTime
		}

		entries = append(entries, entry)
	}

	return entries
}

// ParseOversizeChunk parses an oversize chunk
// Returns the parsed oversize chunk and the remaining data
func ParseOversizeChunk(data []byte) (OversizeChunk, []byte, error) {
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
		// TODO: Log this warning
		// fmt.Printf("Oversize data size greater than Oversize remaining string size. Using remaining string size\n")
		oversizeDataSize = len(data)
	}

	data, publicData, _ := utils.Take(data, oversizeDataSize)
	messageData, _, _ := utils.Take(publicData, 1)
	messageData, itemCount, _ := utils.Take(messageData, 1)
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
func GetOversizeStrings(dataRef uint32, firstProcID uint64, secondProcID uint32, oversizeData []*OversizeChunk) []firehose.ItemInfo {
	messageStrings := []firehose.ItemInfo{}
	for _, oversize := range oversizeData {
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
