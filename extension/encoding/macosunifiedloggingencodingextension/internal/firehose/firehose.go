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

package firehose

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
)

// Preamble represents a parsed firehose preamble
type Preamble struct {
	chunkTag                 uint32
	chunkSubTag              uint32
	chunkDataSize            uint64
	firstProcID              uint64
	secondProcID             uint32
	ttl                      uint8
	collapsed                uint8
	unknown                  []byte
	publicDataSize           uint16
	privateDataVirtualOffset uint16
	unknown2                 uint16
	unknown3                 uint16
	baseContinuousTime       uint64
	PublicData               []Entry
}

const (
	// UnknownRemnantData is the unknown remnant data type
	UnknownRemnantData uint8 = 0x0
	// PrivateNumber is the private number type
	PrivateNumber uint8 = 0x1
	// Base64Raw is the base64 raw type
	Base64Raw uint8 = 0xf2
)

var (
	logTypes       = []uint8{0x2, 0x6, 0x4, 0x7, 0x3}
	stringItem     = []uint8{0x20, 0x21, 0x22, 0x25, 0x40, 0x41, 0x42, 0x30, 0x31, 0x32, 0xf2, 0x35, 0x81, 0xf1}
	privateStrings = []uint8{0x21, 0x25, 0x41, 0x35, 0x31, 0x81, 0xf1}
	precisionItems = []uint8{0x10, 0x12}
	sensitiveItems = []uint8{0x5, 0x45, 0x85}
	arbitrary      = []uint8{0x30, 0x31, 0x32}
)

// Entry represents a parsed firehose log entry
type Entry struct {
	ActivityType            uint8
	LogType                 uint8
	Flags                   uint16
	FormatStringLocation    uint32
	ThreadID                uint64
	ContinousTimeDelta      uint32
	ContinousTimeDeltaUpper uint16
	DataSize                uint16
	FirehoseActivity        Activity
	FirehoseNonActivity     NonActivity
	FirehoseLoss            Loss
	FirehoseSignpost        Signpost
	FirehoseTrace           Trace
	UnknownItem             uint8
	NumberItems             uint8
	Message                 ItemData
}

// ItemType represents a parsed firehose item type
type ItemType struct {
	ItemType          uint8
	ItemSize          uint8
	Offset            uint16
	MessageStringSize uint16
	MessageStrings    string
}

// ItemData represents a parsed firehose item data
type ItemData struct {
	ItemInfo         []ItemInfo
	BacktraceStrings []string
}

// ItemInfo represents a parsed message item from firehose entry data
type ItemInfo struct {
	MessageStrings string
	ItemType       uint8
	ItemSize       uint16
}

// ParseFirehosePreamble parses a firehose chunk preamble
func ParseFirehosePreamble(data []byte) (Preamble, []byte, error) {
	var preamble Preamble

	var chunkTag, chunkSubTag, chunkDataSize, firstProcID, secondProcID, ttl,
		collapsed, unknown, publicDataSize, privateDataVirtualOffset, unknown2, unknown3, baseContinuousTime []byte
	var err error

	data, chunkTag, err = helpers.Take(data, 4)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read chunk tag: %w", err)
	}
	data, chunkSubTag, err = helpers.Take(data, 4)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read chunk sub tag: %w", err)
	}
	data, chunkDataSize, err = helpers.Take(data, 8)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read chunk data size: %w", err)
	}
	data, firstProcID, err = helpers.Take(data, 8)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read first proc ID: %w", err)
	}
	data, secondProcID, err = helpers.Take(data, 4)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read second proc ID: %w", err)
	}
	data, ttl, err = helpers.Take(data, 1)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read TTL: %w", err)
	}
	data, collapsed, err = helpers.Take(data, 1)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read collapsed: %w", err)
	}
	data, unknown, err = helpers.Take(data, 2)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read unknown: %w", err)
	}
	data, publicDataSize, err = helpers.Take(data, 2)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read public data size: %w", err)
	}
	data, privateDataVirtualOffset, err = helpers.Take(data, 2)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read private data virtual offset: %w", err)
	}
	data, unknown2, err = helpers.Take(data, 2)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read unknown2: %w", err)
	}
	data, unknown3, err = helpers.Take(data, 2)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read unknown3: %w", err)
	}
	data, baseContinuousTime, err = helpers.Take(data, 8)
	if err != nil {
		return preamble, data, fmt.Errorf("failed to read base continuous time: %w", err)
	}

	// Save the current data for later use
	logData := data

	preamble.chunkTag = binary.LittleEndian.Uint32(chunkTag)
	preamble.chunkSubTag = binary.LittleEndian.Uint32(chunkSubTag)
	preamble.chunkDataSize = binary.LittleEndian.Uint64(chunkDataSize)
	preamble.firstProcID = binary.LittleEndian.Uint64(firstProcID)
	preamble.secondProcID = binary.LittleEndian.Uint32(secondProcID)
	preamble.ttl = ttl[0]
	preamble.collapsed = collapsed[0]
	preamble.unknown = unknown
	preamble.publicDataSize = binary.LittleEndian.Uint16(publicDataSize)
	preamble.privateDataVirtualOffset = binary.LittleEndian.Uint16(privateDataVirtualOffset)
	preamble.unknown2 = binary.LittleEndian.Uint16(unknown2)
	preamble.unknown3 = binary.LittleEndian.Uint16(unknown3)
	preamble.baseContinuousTime = binary.LittleEndian.Uint64(baseContinuousTime)

	// firehose_public_data_size includes the 16 bytes before the public data offset
	publicDataSizeOffset := 16
	var publicData []byte

	data, publicData, err = helpers.Take(data, int(preamble.publicDataSize)-publicDataSizeOffset)

	if err != nil {
		return preamble, data, fmt.Errorf("failed to read public data: %w", err)
	}

	// Go through all the public data associated with log Firehose entry
	for len(publicData) > 0 {
		var firehoseEntry Entry
		firehoseEntry, publicData, err = ParseFirehoseEntry(publicData)
		if err != nil {
			return preamble, data, fmt.Errorf("failed to parse firehose entry: %w", err)
		}

		if !slices.Contains(logTypes[:], firehoseEntry.ActivityType) || len(publicData) < 24 {
			// If the activity type is unknown remnant data, break
			if firehoseEntry.ActivityType == UnknownRemnantData {
				break
			}
			if preamble.privateDataVirtualOffset != 0x1000 {
				privateDataOffset := 0x1000 - preamble.privateDataVirtualOffset
				// Calculate start of private data. If the remaining input is greater than private data offset.
				// Remove any padding/junk data in front of the private data
				if len(data) > int(privateDataOffset) && len(publicData) == 0 {
					leftoverDataSize := len(data) - int(privateDataOffset)
					// Removing padding in front of the private data
					data, _, err = helpers.Take(data, leftoverDataSize)
					if err != nil {
						return preamble, data, fmt.Errorf("failed to extract padding in front of private data: %w", err)
					}
				} else {
					// If log data and public data are the same size, use private data offset to calculate the private data
					if len(logData) == int(preamble.publicDataSize)-publicDataSizeOffset {
						data, _, err = helpers.Take(logData, int(preamble.privateDataVirtualOffset)-publicDataSizeOffset-len(publicData))
						if err != nil {
							return preamble, data, fmt.Errorf("failed to read private input data: %w", err)
						}
					} else {
						// If we have private data, then any leftover public data is actually prepended to the private data
						data, _, err = helpers.Take(logData, int(preamble.publicDataSize)-publicDataSizeOffset-len(publicData))
						if err != nil {
							return preamble, data, fmt.Errorf("failed to read private input data: %w", err)
						}
					}
				}
			}
			preamble.PublicData = append(preamble.PublicData, firehoseEntry)
			break
		}
		preamble.PublicData = append(preamble.PublicData, firehoseEntry)
	}

	// If there is private data, go through and update any logs that have private data items
	if preamble.privateDataVirtualOffset != 0x1000 {
		// Skip any null padding at the beginning
		offset := 0
		for offset < len(data) && data[offset] == 0 {
			offset++
		}
		privateInput := data[offset:]

		if len(privateInput) == 0 || preamble.collapsed == 1 {
			privateInput = data
		}

		for i := range preamble.PublicData {
			publicData := &preamble.PublicData[i] // Use index to avoid memory aliasing
			if publicData.FirehoseNonActivity.PrivateStringsSize == 0 {
				continue
			}
			stringOffset := publicData.FirehoseNonActivity.PrivateStringsOffset - preamble.privateDataVirtualOffset

			var privateStringStart []byte
			privateStringStart, _, err = helpers.Take(privateInput, int(stringOffset))
			if err != nil {
				return preamble, data, fmt.Errorf("failed to read private string data: %w", err)
			}
			_, err = ParsePrivateData(privateStringStart, &publicData.Message)
			if err != nil {
				return preamble, data, fmt.Errorf("failed to parse private data: %w", err)
			}
		}
		data = privateInput
	}

	return preamble, data, nil
}

// ParseFirehoseEntry parses a firehose entry
func ParseFirehoseEntry(data []byte) (Entry, []byte, error) {
	firehoseResult := Entry{}
	var unknownLogActivityType, unknownLogType, flags, formatStringLocation,
		threadID, continousTimeDelta, continousTimeDeltaUpper, dataSize []byte
	var err error

	data, unknownLogActivityType, err = helpers.Take(data, 1)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read log activity type: %w", err)
	}
	data, unknownLogType, err = helpers.Take(data, 1)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read log type: %w", err)
	}
	data, flags, err = helpers.Take(data, 2)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read flags: %w", err)
	}
	data, formatStringLocation, err = helpers.Take(data, 4)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read format string location: %w", err)
	}
	data, threadID, err = helpers.Take(data, 8)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read thread ID: %w", err)
	}
	data, continousTimeDelta, err = helpers.Take(data, 4)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read continuous time delta: %w", err)
	}
	data, continousTimeDeltaUpper, err = helpers.Take(data, 2)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read continuous time delta upper: %w", err)
	}
	data, dataSize, err = helpers.Take(data, 2)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read data size: %w", err)
	}

	firehoseResult.ActivityType = unknownLogActivityType[0]
	firehoseResult.LogType = unknownLogType[0]
	firehoseResult.Flags = binary.LittleEndian.Uint16(flags)
	firehoseResult.FormatStringLocation = binary.LittleEndian.Uint32(formatStringLocation)
	firehoseResult.ThreadID = binary.LittleEndian.Uint64(threadID)
	firehoseResult.ContinousTimeDelta = binary.LittleEndian.Uint32(continousTimeDelta)
	firehoseResult.ContinousTimeDeltaUpper = binary.LittleEndian.Uint16(continousTimeDeltaUpper)
	firehoseResult.DataSize = binary.LittleEndian.Uint16(dataSize)

	var firehoseData []byte
	data, firehoseData, err = helpers.Take(data, int(firehoseResult.DataSize))
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read firehose data: %w", err)
	}

	// Activity type
	const activity uint8 = 0x2
	const nonactivity uint8 = 0x4
	const signpost uint8 = 0x6
	const loss uint8 = 0x7
	const trace uint8 = 0x3

	if unknownLogActivityType[0] == activity {
		var activity Activity
		activity, firehoseData, err = ParseFirehoseActivity(firehoseData, firehoseResult.Flags, unknownLogType[0])
		if err != nil {
			return firehoseResult, data, err
		}
		firehoseResult.FirehoseActivity = activity

	} else if unknownLogActivityType[0] == nonactivity {
		var nonActivity NonActivity
		nonActivity, firehoseData, err = ParseFirehoseNonActivity(firehoseData, firehoseResult.Flags)
		if err != nil {
			return firehoseResult, data, err
		}
		firehoseResult.FirehoseNonActivity = nonActivity
	} else if unknownLogActivityType[0] == signpost {
		var signpost Signpost
		signpost, firehoseData, err = ParseFirehoseSignpost(firehoseData, firehoseResult.Flags)
		if err != nil {
			return firehoseResult, data, err
		}
		firehoseResult.FirehoseSignpost = signpost
	} else if unknownLogActivityType[0] == loss {
		var loss Loss
		loss, firehoseData, err = ParseFirehoseLoss(firehoseData)
		if err != nil {
			return firehoseResult, data, err
		}
		firehoseResult.FirehoseLoss = loss
	} else if unknownLogActivityType[0] == trace {
		var trace Trace
		trace, firehoseData, err = ParseFirehoseTrace(firehoseData)
		if err != nil {
			return firehoseResult, data, err
		}
		firehoseResult.FirehoseTrace = trace
		firehoseResult.Message = trace.MessageData
	} else if unknownLogActivityType[0] == UnknownRemnantData {
		return firehoseResult, data, nil
	} else {
		// TODO: Handle this warning
		// fmt.Sprintf("Unknown log activity type: %d", unknownLogActivityType[0])
		// Rust Logging of Warning and Debug data
		// warn!(
		// 	"[macos-unifiedlogs] Unknown log activity type: {} -  {} bytes left",
		// 	unknown_log_activity_type,
		// 	input.len()
		// );
		// debug!("[macos-unifiedlogs] Firehose data: {data:X?}");

		return firehoseResult, data, nil
	}

	// Minimum item size is 6 bytes
	if len(firehoseData) < 6 {
		// Skip any zero padding
		offset := 0
		for offset < len(data) && data[offset] == 0 {
			offset++
		}
		data = data[offset:]
		return firehoseResult, data, nil
	}

	var unknownItem []byte
	var numberItems []byte
	firehoseData, unknownItem, err = helpers.Take(firehoseData, 1)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read unknown item: %w", err)
	}
	firehoseData, numberItems, err = helpers.Take(firehoseData, 1)
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read number items: %w", err)
	}
	firehoseResult.UnknownItem = unknownItem[0]
	firehoseResult.NumberItems = numberItems[0]

	var messageData ItemData
	messageData, _, err = ParseFirehoseMessageItems(firehoseData, numberItems[0], firehoseResult.Flags)
	if err != nil {
		return firehoseResult, data, err
	}
	firehoseResult.Message = messageData

	// Skip any zero padding
	offset := 0
	for offset < len(data) && data[offset] == 0 {
		offset++
	}
	remainingData := data[offset:]

	// Verify we didn't remove padding into remnant/junk data
	paddingData := helpers.PaddingSize(uint64(firehoseResult.DataSize), 8)
	if paddingData > uint64(^uint(0)>>1) {
		return firehoseResult, data, fmt.Errorf("u64 is bigger than system int")
	}
	data, _, err = helpers.Take(data, int(paddingData))
	if err != nil {
		return firehoseResult, data, fmt.Errorf("failed to read padding data: %w", err)
	}
	if int(paddingData) > offset {
		data = remainingData
	}

	return firehoseResult, data, nil
}

// ParsePrivateData parses private data for log entry
// Returns the remaining data and an error if the private data is not found
func ParsePrivateData(data []byte, firehoseItemData *ItemData) ([]byte, error) {
	privateStringStart := data
	var err error

	for i := range firehoseItemData.ItemInfo {
		firehoseInfo := &firehoseItemData.ItemInfo[i]
		if slices.Contains(privateStrings[:], firehoseInfo.ItemType) {
			if firehoseInfo.ItemType == privateStrings[3] || firehoseInfo.ItemType == privateStrings[4] {
				var privateData []byte
				var pointerObject []byte

				if len(privateStringStart) < int(firehoseInfo.ItemSize) {
					privateData, pointerObject, err = helpers.Take(privateStringStart, len(privateStringStart))
					if err != nil {
						return privateStringStart, fmt.Errorf("failed to read private data object: %w", err)
					}
					privateStringStart = privateData
					firehoseInfo.MessageStrings = base64.StdEncoding.EncodeToString(pointerObject)
					continue
				}

				privateData, pointerObject, err = helpers.Take(privateStringStart, int(firehoseInfo.ItemSize))
				if err != nil {
					return privateStringStart, fmt.Errorf("failed to read private data object: %w", err)
				}
				privateStringStart = privateData
				firehoseInfo.MessageStrings = base64.StdEncoding.EncodeToString(pointerObject)
				continue
			}
			nullPrivate := uint16(0)
			if firehoseInfo.ItemSize == nullPrivate {
				firehoseInfo.MessageStrings = "<private>"
			} else {
				privateData, messageString, err := helpers.ExtractStringSize(privateStringStart, uint64(firehoseInfo.ItemSize))
				if err != nil {
					return privateStringStart, fmt.Errorf("failed to extract private string: %w", err)
				}
				privateStringStart = privateData
				firehoseInfo.MessageStrings = messageString
			}
		} else if firehoseInfo.ItemType == PrivateNumber {
			nullPrivate := uint16(0x8000)
			if firehoseInfo.ItemSize == nullPrivate {
				firehoseInfo.MessageStrings = "<private>"
			} else {
				privateData, privateNumber := ParseItemNumber(privateStringStart, uint8(firehoseInfo.ItemSize))
				privateStringStart = privateData
				firehoseInfo.MessageStrings = fmt.Sprintf("%d", privateNumber)
			}
		}
	}
	return privateStringStart, nil
}

// GetBacktraceData parses backtrace data for log entry (chunk). This only exists if `has_context_data` flag is set
func GetBacktraceData(data []byte) ([]byte, []string, error) {
	var uuidCountBytes []byte
	var offsetCountBytes []byte
	var uuidBytes []byte
	var offsetBytes []byte
	var indexBytes []byte
	var err error

	// Skip 3 unknown bytes
	data, _, err = helpers.Take(data, 3)
	if err != nil {
		return data, nil, fmt.Errorf("failed to read backtrace data: %w", err)
	}

	// Read counts
	data, uuidCountBytes, err = helpers.Take(data, 1)
	if err != nil {
		return data, nil, fmt.Errorf("failed to read uuid count: %w", err)
	}
	data, offsetCountBytes, err = helpers.Take(data, 2)
	if err != nil {
		return data, nil, fmt.Errorf("failed to read offset count: %w", err)
	}
	uuidCount := int(uuidCountBytes[0])
	offsetCount := int(binary.LittleEndian.Uint16(offsetCountBytes))

	// Read UUID vector (128-bit big-endian UUIDs)
	var uuidVec []string
	for i := 0; i < uuidCount; i++ {
		data, uuidBytes, err = helpers.Take(data, 16) // 128 bits = 16 bytes
		if err != nil {
			return data, nil, fmt.Errorf("failed to read uuid bytes: %w", err)
		}
		// Convert 128-bit UUID to uppercase hex string (big-endian)
		uuidStr := fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
			uuidBytes[0], uuidBytes[1], uuidBytes[2], uuidBytes[3],
			uuidBytes[4], uuidBytes[5], uuidBytes[6], uuidBytes[7],
			uuidBytes[8], uuidBytes[9], uuidBytes[10], uuidBytes[11],
			uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])
		uuidVec = append(uuidVec, uuidStr)
	}

	// Read offsets vector (32-bit little-endian)
	var offsetsVec []uint32
	for i := 0; i < offsetCount; i++ {
		data, offsetBytes, err = helpers.Take(data, 4)
		if err != nil {
			return data, nil, fmt.Errorf("failed to read offset bytes: %w", err)
		}
		offset := binary.LittleEndian.Uint32(offsetBytes)
		offsetsVec = append(offsetsVec, offset)
	}

	// Read indexes (8-bit)
	var indexes []uint8
	for i := 0; i < offsetCount; i++ {
		data, indexBytes, err = helpers.Take(data, 1)
		if err != nil {
			return data, nil, fmt.Errorf("failed to read index bytes: %w", err)
		}
		indexes = append(indexes, indexBytes[0])
	}

	// Build backtrace strings
	var backtraceData []string
	for i, idx := range indexes {
		var uuidStr string
		if int(idx) < len(uuidVec) {
			uuidStr = uuidVec[idx]
		}
		var offset uint32
		if i < len(offsetsVec) {
			offset = offsetsVec[i]
		}

		backtraceStr := fmt.Sprintf("\"%s\" +0x%x", uuidStr, offset)
		backtraceData = append(backtraceData, backtraceStr)
	}

	// Calculate padding size (align to 4-byte boundary)
	paddingSize := helpers.PaddingSizeFour(uint64(offsetCount))
	if paddingSize > uint64(^uint(0)>>1) { // Check if larger than max int
		return data, backtraceData, fmt.Errorf("u64 is bigger than system int")
	}

	// Skip padding bytes
	data, _, err = helpers.Take(data, int(paddingSize))
	if err != nil {
		return data, nil, fmt.Errorf("failed to read padding bytes: %w", err)
	}

	return data, backtraceData, nil
}

// ParseFirehoseMessageItems parses message items from firehose entry data
func ParseFirehoseMessageItems(data []byte, numItems uint8, flags uint16) (ItemData, []byte, error) {
	itemCount := 0
	itemsData := []ItemType{}
	var err error
	var parsedBacktraceSig []byte

	firehoseInput := data
	firehoseItemData := ItemData{}
	numberItemType := []uint8{0x0, 0x2}
	objectItems := []uint8{0x40, 0x42}

	for itemCount < int(numItems) {
		item, itemInput := GetFirehoseItems(firehoseInput)
		firehoseInput = itemInput

		if slices.Contains(precisionItems, item.ItemType) {
			itemsData = append(itemsData, item)
			itemCount++
			continue
		}

		if slices.Contains(numberItemType, item.ItemType) {
			itemValueInput, messageNumber := ParseItemNumber(firehoseInput, item.ItemSize)
			item.MessageStrings = fmt.Sprintf("%d", messageNumber)
			firehoseInput = itemValueInput
			itemCount++
			itemsData = append(itemsData, item)
			continue
		}

		if item.MessageStringSize == 0 && slices.Contains(objectItems, item.ItemType) {
			item.MessageStrings = "(null)"
		}
		itemsData = append(itemsData, item)
		itemCount++
	}

	// Backtrace data appears before Firehose item strings
	// It only exists if log entry has_context_data flag set
	// Backtrace data can also exist in Oversize log entries. However, Oversize entries do not have has_context_data flags. Instead we check for possible signature
	hasContextData := uint16(0x1000)
	backtraceSignatureSize := 3

	if (flags & hasContextData) != 0 {
		backtraceInput, backtraceData, err := GetBacktraceData(firehoseInput)
		if err != nil {
			return firehoseItemData, firehoseInput, fmt.Errorf("failed to get backtrace data: %w", err)
		}
		firehoseInput = backtraceInput
		firehoseItemData.BacktraceStrings = backtraceData
	} else if len(firehoseInput) > backtraceSignatureSize {
		backtraceSignature := []uint8{1, 0, 18}
		_, parsedBacktraceSig, err = helpers.Take(firehoseInput, backtraceSignatureSize)
		if err != nil {
			return firehoseItemData, firehoseInput, fmt.Errorf("failed to read backtrace signature: %w", err)
		}
		if len(parsedBacktraceSig) == len(backtraceSignature) &&
			parsedBacktraceSig[0] == backtraceSignature[0] &&
			parsedBacktraceSig[1] == backtraceSignature[1] &&
			parsedBacktraceSig[2] == backtraceSignature[2] {
			backtraceInput, backtraceData, err := GetBacktraceData(firehoseInput)
			if err != nil {
				return firehoseItemData, firehoseInput, fmt.Errorf("failed to get backtrace data: %w", err)
			}
			firehoseInput = backtraceInput
			firehoseItemData.BacktraceStrings = backtraceData
		}
	}

	for i, item := range itemsData {
		if slices.Contains(numberItemType, item.ItemType) {
			continue
		}

		if slices.Contains(privateStrings, uint8(item.ItemType)) || slices.Contains(sensitiveItems, uint8(item.ItemType)) {
			itemsData[i].MessageStrings = "<private>"
			continue
		}

		if uint8(item.ItemType) == PrivateNumber {
			continue
		}

		if slices.Contains(precisionItems, uint8(item.ItemType)) {
			continue
		}

		if item.MessageStringSize == 0 && len(item.MessageStrings) != 0 {
			continue
		}

		if len(firehoseInput) == 0 {
			break
		}

		if slices.Contains(stringItem[:], uint8(item.ItemType)) {
			itemValueInput, messageString, _ := ParseItemString(firehoseInput, item.ItemType, item.MessageStringSize)
			firehoseInput = itemValueInput
			itemsData[i].MessageStrings = messageString
		} else {
			return firehoseItemData, firehoseInput, fmt.Errorf("unknown item type: %d", item.ItemType)
		}
	}

	for _, item := range itemsData {
		firehoseItemData.ItemInfo = append(firehoseItemData.ItemInfo, ItemInfo{
			MessageStrings: item.MessageStrings,
			ItemType:       item.ItemType,
			ItemSize:       item.MessageStringSize,
		})
	}
	return firehoseItemData, firehoseInput, nil
}

// GetFirehoseItems gets the firehose item type and size
func GetFirehoseItems(data []byte) (ItemType, []byte) {
	var remainingData, itemTypeBytes, itemSizeBytes []byte
	var err error

	item := ItemType{}
	remainingData, itemTypeBytes, err = helpers.Take(data, 1)
	if err != nil {
		return item, remainingData
	}
	remainingData, itemSizeBytes, err = helpers.Take(remainingData, 1)
	if err != nil {
		return item, remainingData
	}
	item = ItemType{
		ItemType: itemTypeBytes[0],
		ItemSize: itemSizeBytes[0],
	}

	if slices.Contains(stringItem[:], uint8(item.ItemType)) || uint8(item.ItemType) == PrivateNumber {
		var offsetBytes, sizeBytes []byte
		remainingData, offsetBytes, err = helpers.Take(remainingData, 2)
		if err != nil {
			return item, remainingData
		}
		remainingData, sizeBytes, err = helpers.Take(remainingData, 2)
		if err != nil {
			return item, remainingData
		}
		item.Offset = binary.LittleEndian.Uint16(offsetBytes)
		item.MessageStringSize = binary.LittleEndian.Uint16(sizeBytes)
	}

	// Precision items just contain the length for the actual item. Ex: %*s
	if slices.Contains(precisionItems, item.ItemType) {
		remainingData, _, err = helpers.Take(remainingData, int(item.ItemSize))
		if err != nil {
			return item, remainingData
		}
	}

	if slices.Contains(sensitiveItems, item.ItemType) {
		var sensitiveOffsetBytes, sensitiveSizeBytes []byte
		remainingData, sensitiveOffsetBytes, err = helpers.Take(remainingData, 2)
		if err != nil {
			return item, remainingData
		}
		remainingData, sensitiveSizeBytes, err = helpers.Take(remainingData, 2)
		if err != nil {
			return item, remainingData
		}
		item.Offset = binary.LittleEndian.Uint16(sensitiveOffsetBytes)
		item.MessageStringSize = binary.LittleEndian.Uint16(sensitiveSizeBytes)
	}

	return item, remainingData
}

// ParseItemString parses a string from the firehose data based on the item type and message size
func ParseItemString(data []byte, itemType uint8, messageSize uint16) ([]byte, string, error) {
	var messageData []byte
	var err error

	if messageSize > uint16(len(data)) {
		return helpers.ExtractStringSize(data, uint64(messageSize))
	}
	data, messageData, err = helpers.Take(data, int(messageSize))
	if err != nil {
		return data, "", err
	}

	// 0x30, 0x31, and 0x32 represent arbitrary data, need to be decoded again
	if slices.Contains(arbitrary, itemType) {
		return data, base64.StdEncoding.EncodeToString(messageData), nil
	}

	if itemType == Base64Raw {
		return data, base64.StdEncoding.EncodeToString(messageData), nil
	}

	_, messageString, err := helpers.ExtractStringSize(messageData, uint64(messageSize))
	if err != nil {
		return data, "", err
	}
	return data, messageString, nil
}

// ParseItemNumber parses a number from the firehose data based on the item size
func ParseItemNumber(data []byte, itemSize uint8) ([]byte, uint64) {
	switch itemSize {
	case 4:
		value := int64(binary.LittleEndian.Uint32(data[:4]))
		return data[4:], uint64(value)
	case 2:
		value := int64(binary.LittleEndian.Uint16(data[:2]))
		return data[2:], uint64(value)
	case 8:
		value := binary.LittleEndian.Uint64(data[:8])
		return data[8:], value
	case 1:
		value := int64(data[0])
		return data[1:], uint64(value)
	default:
		return data, uint64(0) // Unknown size
	}
}
