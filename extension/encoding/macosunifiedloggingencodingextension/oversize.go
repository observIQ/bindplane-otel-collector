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
	firehoseItemData, _ = ParsePrivateData(oversizePrivateData, firehoseItemData)
	oversizeResult.messageItems = firehoseItemData
}

// // ParseOversizeChunk parses an Oversize chunk (0x6002) containing large log entries
// // Returns a slice of individual TraceV3Entry objects for each log message found in the oversize data
// // Based on rust implementation: uses FirehosePreamble::collect_items approach
// func ParseOversizeChunk(data []byte, templateEntry *TraceV3Entry, header *TraceV3Header, timesyncData map[string]*TimesyncBoot) []*TraceV3Entry {
// 	if len(data) < 48 { // Need at least 48 bytes for oversize header
// 		templateEntry.Message = fmt.Sprintf("Oversize chunk too small: %d bytes", len(data))
// 		return []*TraceV3Entry{templateEntry}
// 	}

// 	// Parse oversize chunk header (based on rust implementation)
// 	firstProcID := binary.LittleEndian.Uint64(data[16:24])
// 	secondProcID := binary.LittleEndian.Uint32(data[24:28])
// 	ttl := data[28]
// 	// Skip 3 bytes of unknown_reserved data (bytes 29-31)
// 	continuousTime := binary.LittleEndian.Uint64(data[32:40])
// 	dataRefIndex := binary.LittleEndian.Uint32(data[40:44])
// 	publicDataSize := binary.LittleEndian.Uint16(data[44:46])
// 	privateDataSize := binary.LittleEndian.Uint16(data[46:48])

// 	// Calculate total data size for oversize content
// 	totalDataSize := int(publicDataSize + privateDataSize)
// 	if len(data) < 48+totalDataSize {
// 		templateEntry.Message = fmt.Sprintf("Oversize chunk truncated: expected %d bytes, got %d", 48+totalDataSize, len(data))
// 		return []*TraceV3Entry{templateEntry}
// 	}

// 	// Extract the message data section starting at offset 48
// 	messageData := data[48 : 48+totalDataSize]

// 	// Parse the message items like firehose (based on rust implementation)
// 	// The first two bytes indicate structure: first byte unknown, second byte is item count
// 	if len(messageData) < 2 {
// 		templateEntry.Message = fmt.Sprintf("Oversize data too small for item header: %d bytes", len(messageData))
// 		return []*TraceV3Entry{templateEntry}
// 	}

// 	// Parse individual log messages from oversize data using rust approach
// 	// Skip first byte (unknown), second byte is item count, then use firehose item parsing
// 	itemCount := messageData[1]
// 	if itemCount == 0 {
// 		templateEntry.Message = fmt.Sprintf("Oversize chunk: ttl=%d dataRef=%d publicSize=%d privateSize=%d (no items)",
// 			ttl, dataRefIndex, publicDataSize, privateDataSize)
// 		return []*TraceV3Entry{templateEntry}
// 	}

// 	// Parse firehose items using the same logic as the rust implementation
// 	items := parseFirehoseItemsFromOversize(messageData[2:], int(itemCount))

// 	// Convert parsed items to individual log entries
// 	var entries []*TraceV3Entry
// 	for _, item := range items {
// 		// Create entries for items with actual content
// 		if item.MessageStrings != "" {
// 			entry := &TraceV3Entry{
// 				Type:         templateEntry.Type,
// 				Size:         uint32(item.ItemSize),
// 				ThreadID:     firstProcID,
// 				ProcessID:    secondProcID,
// 				Subsystem:    "com.apple.oversize.decompressed",
// 				Category:     "", // Empty like rust implementation
// 				Level:        "Info",
// 				MessageType:  "Default",
// 				EventType:    "logEvent",
// 				ChunkType:    "oversize",
// 				TimezoneName: templateEntry.TimezoneName,
// 				Message:      item.MessageStrings,
// 			}

// 			// Calculate timestamp using oversize's continuous time
// 			if timesyncData != nil && header != nil {
// 				entry.Timestamp = convertMachTimeToUnixNanosWithTimesync(continuousTime, header.BootUUID, 0, timesyncData)
// 			} else {
// 				entry.Timestamp = continuousTime
// 			}

// 			entries = append(entries, entry)
// 		}
// 	}

// 	// If no entries were created, create a summary entry
// 	if len(entries) == 0 {
// 		entry := &TraceV3Entry{
// 			Type:         templateEntry.Type,
// 			Size:         uint32(totalDataSize),
// 			ThreadID:     firstProcID,
// 			ProcessID:    secondProcID,
// 			Subsystem:    "com.apple.oversize.decompressed",
// 			Category:     "",
// 			Level:        "Info",
// 			MessageType:  "Default",
// 			EventType:    "logEvent",
// 			ChunkType:    "oversize",
// 			TimezoneName: templateEntry.TimezoneName,
// 			Message: fmt.Sprintf("Oversize chunk: ttl=%d dataRef=%d publicSize=%d privateSize=%d items=%d (no content extracted)",
// 				ttl, dataRefIndex, publicDataSize, privateDataSize, itemCount),
// 		}

// 		// Calculate timestamp using oversize's continuous time
// 		if timesyncData != nil && header != nil {
// 			entry.Timestamp = convertMachTimeToUnixNanosWithTimesync(continuousTime, header.BootUUID, 0, timesyncData)
// 		} else {
// 			entry.Timestamp = continuousTime
// 		}

// 		entries = append(entries, entry)
// 	}

// 	return entries
// }

// // parseFirehoseItemsFromOversize parses firehose items from oversize data using rust implementation approach
// // Based on FirehosePreamble::collect_items logic
// func parseFirehoseItemsFromOversize(data []byte, itemCount int) []FirehoseItemInfo {
// 	var items []FirehoseItemInfo
// 	offset := 0

// 	// String item types from rust implementation
// 	stringItems := map[uint8]bool{
// 		0x20: true, 0x21: true, 0x22: true, 0x25: true, 0x40: true, 0x41: true, 0x42: true,
// 		0x30: true, 0x31: true, 0x32: true, 0xf2: true, 0x35: true, 0x81: true, 0xf1: true,
// 	}

// 	// Number item types
// 	numberItems := map[uint8]bool{
// 		0x0: true, 0x2: true,
// 	}

// 	// Private number item
// 	privateNumber := uint8(0x1)

// 	// Object items
// 	objectItems := map[uint8]bool{
// 		0x40: true, 0x42: true,
// 	}

// 	// Parse each item following the firehose item structure
// 	for i := 0; i < itemCount && offset < len(data); i++ {
// 		if offset+2 > len(data) {
// 			break
// 		}

// 		item := FirehoseItemInfo{
// 			ItemType: data[offset],
// 			ItemSize: uint16(data[offset+1]),
// 		}
// 		offset += 2

// 		// String and private number items have 4 bytes of metadata (offset + size)
// 		if stringItems[item.ItemType] || item.ItemType == privateNumber {
// 			if offset+4 > len(data) {
// 				break
// 			}
// 			messageOffset := binary.LittleEndian.Uint16(data[offset : offset+2])
// 			messageSize := binary.LittleEndian.Uint16(data[offset+2 : offset+4])
// 			offset += 4

// 			// Store offset and size for later string extraction
// 			item.MessageOffset = messageOffset
// 			item.MessageStringSize = messageSize
// 		}

// 		items = append(items, item)
// 	}

// 	// Now extract the actual string data from the end of the buffer
// 	// The rust implementation shows that string data comes after all item metadata
// 	stringDataStart := offset
// 	for i := range items {
// 		item := &items[i]

// 		// Handle string items
// 		if stringItems[item.ItemType] || item.ItemType == privateNumber {
// 			if item.MessageOffset < uint16(len(data)-stringDataStart) &&
// 				item.MessageOffset+item.MessageStringSize <= uint16(len(data)-stringDataStart) {
// 				stringStart := stringDataStart + int(item.MessageOffset)
// 				stringEnd := stringStart + int(item.MessageStringSize)
// 				if stringEnd <= len(data) {
// 					messageBytes := data[stringStart:stringEnd]
// 					// Remove null terminators
// 					for len(messageBytes) > 0 && messageBytes[len(messageBytes)-1] == 0 {
// 						messageBytes = messageBytes[:len(messageBytes)-1]
// 					}
// 					if len(messageBytes) > 0 {
// 						item.MessageStrings = string(messageBytes)
// 					}
// 				}
// 			}
// 		} else if numberItems[item.ItemType] {
// 			// Number items - extract the number value
// 			if offset < len(data) {
// 				// Parse number based on item size
// 				if item.ItemSize >= 8 && offset+8 <= len(data) {
// 					value := binary.LittleEndian.Uint64(data[offset : offset+8])
// 					item.MessageStrings = fmt.Sprintf("%d", value)
// 					offset += 8
// 				} else if item.ItemSize >= 4 && offset+4 <= len(data) {
// 					value := binary.LittleEndian.Uint32(data[offset : offset+4])
// 					item.MessageStrings = fmt.Sprintf("%d", value)
// 					offset += 4
// 				} else if item.ItemSize >= 2 && offset+2 <= len(data) {
// 					value := binary.LittleEndian.Uint16(data[offset : offset+2])
// 					item.MessageStrings = fmt.Sprintf("%d", value)
// 					offset += 2
// 				} else if item.ItemSize >= 1 && offset+1 <= len(data) {
// 					item.MessageStrings = fmt.Sprintf("%d", data[offset])
// 					offset++
// 				}
// 			}
// 		} else if objectItems[item.ItemType] && item.ItemSize == 0 {
// 			// Object items with size 0 are "(null)"
// 			item.MessageStrings = "(null)"
// 		} else {
// 			// Private or unknown items
// 			item.MessageStrings = "<private>"
// 		}
// 	}

// 	return items
// }

// // extractReadableContent tries to extract any readable string content from binary data
// // This is a fallback method when structured parsing fails
// func extractReadableContent(data []byte) string {
// 	if len(data) == 0 {
// 		return ""
// 	}

// 	var readableStrings []string
// 	var currentString []byte

// 	// Scan through the data looking for printable ASCII sequences
// 	for _, b := range data {
// 		if b >= 32 && b <= 126 { // Printable ASCII
// 			currentString = append(currentString, b)
// 		} else {
// 			// End of current string, save if it's long enough to be meaningful
// 			if len(currentString) >= 4 { // At least 4 characters
// 				str := string(currentString)
// 				if isPrintableString(str) {
// 					readableStrings = append(readableStrings, str)
// 				}
// 			}
// 			currentString = currentString[:0]
// 		}
// 	}

// 	// Don't forget the last string if the data ends with printable characters
// 	if len(currentString) >= 4 {
// 		str := string(currentString)
// 		if isPrintableString(str) {
// 			readableStrings = append(readableStrings, str)
// 		}
// 	}

// 	// Combine found strings with separators
// 	if len(readableStrings) > 0 {
// 		return strings.Join(readableStrings, " | ")
// 	}

// 	return ""
// }
