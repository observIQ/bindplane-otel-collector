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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
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
	publicData               []Entry
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
	stringItem     = []uint8{0x20, 0x22, 0x40, 0x42, 0x30, 0x31, 0x32, 0xf2}
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
func ParseFirehosePreamble(data []byte) (Preamble, []byte) {
	var preamble Preamble

	data, chunkTag, _ := utils.Take(data, 4)
	data, chunkSubTag, _ := utils.Take(data, 4)
	data, chunkDataSize, _ := utils.Take(data, 8)
	data, firstProcID, _ := utils.Take(data, 8)
	data, secondProcID, _ := utils.Take(data, 4)
	data, ttl, _ := utils.Take(data, 1)
	data, collapsed, _ := utils.Take(data, 1)
	data, unknown, _ := utils.Take(data, 2)
	data, publicDataSize, _ := utils.Take(data, 2)
	data, privateDataVirtualOffset, _ := utils.Take(data, 2)
	data, unknown2, _ := utils.Take(data, 2)
	data, unknown3, _ := utils.Take(data, 2)
	logData, baseContinuousTime, _ := utils.Take(data, 8)

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
	data, publicData, _ := utils.Take(logData, int(preamble.publicDataSize)-publicDataSizeOffset)

	// Go through all the public data associated with log Firehose entry
	for len(publicData) > 0 {
		firehosePublicData, firehoseInput, _ := ParseFirehoseEntry(publicData)
		publicData = firehoseInput

		if !slices.Contains(logTypes[:], firehosePublicData.ActivityType) || len(publicData) < 24 {
			// If the activity type is unknown remnant data, break
			if firehosePublicData.ActivityType == UnknownRemnantData {
				break
			}
			if preamble.privateDataVirtualOffset != 0x1000 {
				privateDataOffset := 0x1000 - preamble.privateDataVirtualOffset

				if len(data) > int(privateDataOffset) && len(publicData) == 0 {
					leftoverDataSize := len(data) - int(privateDataOffset)
					_, privateData, _ := utils.Take(data, leftoverDataSize)
					data = privateData
				} else {
					// If log data and public data are the same size, use private data offset to calculate the private data
					if len(logData) == int(preamble.publicDataSize)-publicDataSizeOffset {
						_, privateInputData, _ := utils.Take(logData, int(preamble.privateDataVirtualOffset)-publicDataSizeOffset-len(publicData))
						data = privateInputData
					} else {
						// If we have private data, then any leftover public data is actually prepended to the private data
						_, privateInputData, _ := utils.Take(logData, int(preamble.publicDataSize)-publicDataSizeOffset-len(publicData))
						data = privateInputData
					}
				}
			}
			preamble.publicData = append(preamble.publicData, firehosePublicData)
			break
		}
		preamble.publicData = append(preamble.publicData, firehosePublicData)
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

		for _, data := range preamble.publicData {
			if data.FirehoseNonActivity.PrivateStringsSize == 0 {
				continue
			}
			stringOffset := data.FirehoseNonActivity.PrivateStringsOffset - preamble.privateDataVirtualOffset
			_, privateStringStart, _ := utils.Take(privateInput, int(stringOffset))
			ParsePrivateData(privateStringStart, &data.Message)
		}
		data = privateInput
	}

	return preamble, data
}

// ParseFirehoseEntry parses a firehose entry
func ParseFirehoseEntry(data []byte) (Entry, []byte, error) {
	firehoseResult := Entry{}

	data, unknownLogActivityType, _ := utils.Take(data, 1)
	data, unknownLogType, _ := utils.Take(data, 1)
	data, flags, _ := utils.Take(data, 2)
	data, formatStringLocation, _ := utils.Take(data, 4)
	data, threadID, _ := utils.Take(data, 8)
	data, continousTimeDelta, _ := utils.Take(data, 4)
	data, continousTimeDeltaUpper, _ := utils.Take(data, 2)
	data, dataSize, _ := utils.Take(data, 2)

	firehoseResult.ActivityType = unknownLogActivityType[0]
	firehoseResult.LogType = unknownLogType[0]
	firehoseResult.Flags = binary.LittleEndian.Uint16(flags)
	firehoseResult.FormatStringLocation = binary.LittleEndian.Uint32(formatStringLocation)
	firehoseResult.ThreadID = binary.LittleEndian.Uint64(threadID)
	firehoseResult.ContinousTimeDelta = binary.LittleEndian.Uint32(continousTimeDelta)
	firehoseResult.ContinousTimeDeltaUpper = binary.LittleEndian.Uint16(continousTimeDeltaUpper)
	firehoseResult.DataSize = binary.LittleEndian.Uint16(dataSize)

	data, firehoseData, _ := utils.Take(data, int(firehoseResult.DataSize))

	// Activity type
	const activity uint8 = 0x2
	const nonactivity uint8 = 0x4
	const signpost uint8 = 0x6
	const loss uint8 = 0x7
	const trace uint8 = 0x3

	if unknownLogActivityType[0] == activity {
		activity, activityData := ParseFirehoseActivity(firehoseData, firehoseResult.Flags, unknownLogType[0])
		firehoseData = activityData
		firehoseResult.FirehoseActivity = activity

	} else if unknownLogActivityType[0] == nonactivity {
		nonActivity, nonActivityData := ParseFirehoseNonActivity(firehoseData, firehoseResult.Flags)
		firehoseData = nonActivityData
		firehoseResult.FirehoseNonActivity = nonActivity
	} else if unknownLogActivityType[0] == signpost {
		signpost, signpostData := ParseFirehoseSignpost(firehoseData, firehoseResult.Flags)
		firehoseData = signpostData
		firehoseResult.FirehoseSignpost = signpost
	} else if unknownLogActivityType[0] == loss {
		loss, lossData := ParseFirehoseLoss(firehoseData)
		firehoseData = lossData
		firehoseResult.FirehoseLoss = loss
	} else if unknownLogActivityType[0] == trace {
		trace, traceData, err := ParseFirehoseTrace(firehoseData)
		if err != nil {
			return firehoseResult, data, err
		}
		firehoseData = traceData
		firehoseResult.FirehoseTrace = trace
		firehoseResult.Message = trace.MessageData
	} else if unknownLogActivityType[0] == UnknownRemnantData {
		return firehoseResult, data, nil
	} else {
		// TODO: Handle this warning
		// fmt.Sprintf("Unknown log activity type: %d", unknownLogActivityType[0])
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

	firehoseData, unknownItem, _ := utils.Take(firehoseData, 1)
	firehoseData, numberItems, _ := utils.Take(firehoseData, 1)
	firehoseResult.UnknownItem = unknownItem[0]
	firehoseResult.NumberItems = numberItems[0]
	messageData, _, err := ParseFirehoseMessageItems(firehoseData, numberItems[0], firehoseResult.Flags)
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
	paddingData := utils.PaddingSize(uint64(firehoseResult.DataSize), 8)
	if paddingData > uint64(^uint(0)>>1) {
		return firehoseResult, data, fmt.Errorf("u64 is bigger than system int")
	}
	data, _, _ = utils.Take(data, int(paddingData))
	if int(paddingData) > len(remainingData) {
		data = remainingData
	}

	return firehoseResult, data, nil
}

// ParsePrivateData parses private data for log entry
// Returns the remaining data and an error if the private data is not found
func ParsePrivateData(data []byte, firehoseItemData *ItemData) ([]byte, error) {
	privateStringStart := data

	for i := range firehoseItemData.ItemInfo {
		firehoseInfo := &firehoseItemData.ItemInfo[i]
		if slices.Contains(privateStrings[:], firehoseInfo.ItemType) {
			if firehoseInfo.ItemType == privateStrings[3] || firehoseInfo.ItemType == privateStrings[4] {
				if len(privateStringStart) < int(firehoseInfo.ItemSize) {
					privateData, pointerObject, _ := utils.Take(privateStringStart, len(privateStringStart))
					privateStringStart = privateData
					firehoseInfo.MessageStrings = base64.StdEncoding.EncodeToString(pointerObject)
					continue
				}

				privateData, pointerObject, _ := utils.Take(privateStringStart, int(firehoseInfo.ItemSize))
				privateStringStart = privateData
				firehoseInfo.MessageStrings = base64.StdEncoding.EncodeToString(pointerObject)
				continue
			}
			nullPrivate := uint16(0)
			if firehoseInfo.ItemSize == nullPrivate {
				firehoseInfo.MessageStrings = "<private>"
			} else {
				privateData, messageString, _ := utils.ExtractStringSize(privateStringStart, uint64(firehoseInfo.ItemSize))
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
	// Skip 3 unknown bytes
	input, _, _ := utils.Take(data, 3)

	// Read counts
	input, uuidCountBytes, _ := utils.Take(input, 1)
	input, offsetCountBytes, _ := utils.Take(input, 2)
	uuidCount := int(uuidCountBytes[0])
	offsetCount := int(binary.LittleEndian.Uint16(offsetCountBytes))

	// Read UUID vector (128-bit big-endian UUIDs)
	var uuidVec []string
	for i := 0; i < uuidCount; i++ {
		remaining, uuidBytes, _ := utils.Take(input, 16) // 128 bits = 16 bytes
		input = remaining
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
		remaining, offsetBytes, _ := utils.Take(input, 4)
		input = remaining
		offset := binary.LittleEndian.Uint32(offsetBytes)
		offsetsVec = append(offsetsVec, offset)
	}

	// Read indexes (8-bit)
	var indexes []uint8
	for i := 0; i < offsetCount; i++ {
		remaining, indexBytes, _ := utils.Take(input, 1)
		input = remaining
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
	paddingSize := utils.PaddingSizeFour(uint64(offsetCount))
	if paddingSize > uint64(^uint(0)>>1) { // Check if larger than max int
		return input, backtraceData, fmt.Errorf("u64 is bigger than system int")
	}

	// Skip padding bytes
	backtraceInput, _, _ := utils.Take(input, int(paddingSize))
	input = backtraceInput

	return input, backtraceData, nil
}

// ParseFirehoseMessageItems parses message items from firehose entry data
func ParseFirehoseMessageItems(data []byte, numItems uint8, flags uint16) (ItemData, []byte, error) {
	itemCount := 0
	itemsData := []ItemType{}

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
		backtraceInput, backtraceData, _ := GetBacktraceData(firehoseInput)
		firehoseInput = backtraceInput
		firehoseItemData.BacktraceStrings = backtraceData
	} else if len(firehoseInput) > backtraceSignatureSize {
		backtraceSignature := []uint8{1, 0, 18}
		_, parsedBacktraceSig, _ := utils.Take(firehoseInput, backtraceSignatureSize)
		if len(parsedBacktraceSig) == len(backtraceSignature) &&
			parsedBacktraceSig[0] == backtraceSignature[0] &&
			parsedBacktraceSig[1] == backtraceSignature[1] &&
			parsedBacktraceSig[2] == backtraceSignature[2] {
			backtraceInput, backtraceData, _ := GetBacktraceData(firehoseInput)
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
			ItemSize:       uint16(item.ItemSize),
		})
	}
	return firehoseItemData, firehoseInput, nil
}

// GetFirehoseItems gets the firehose item type and size
func GetFirehoseItems(data []byte) (ItemType, []byte) {
	remainingData, itemTypeBytes, _ := utils.Take(data, 1)
	remainingData, itemSizeBytes, _ := utils.Take(remainingData, 1)
	item := ItemType{
		ItemType: itemTypeBytes[0],
		ItemSize: itemSizeBytes[0],
	}

	if slices.Contains(stringItem[:], uint8(item.ItemType)) || uint8(item.ItemType) == PrivateNumber {
		var offsetBytes, sizeBytes []byte
		remainingData, offsetBytes, _ = utils.Take(remainingData, 2)
		remainingData, sizeBytes, _ = utils.Take(remainingData, 2)
		item.Offset = binary.LittleEndian.Uint16(offsetBytes)
		item.MessageStringSize = binary.LittleEndian.Uint16(sizeBytes)
	}

	// Precision items just contain the length for the actual item. Ex: %*s
	if slices.Contains(precisionItems, item.ItemType) {
		remainingData, _, _ = utils.Take(remainingData, int(item.ItemSize))
	}

	if slices.Contains(sensitiveItems, item.ItemType) {
		var sensitiveOffsetBytes, sensitiveSizeBytes []byte
		remainingData, sensitiveOffsetBytes, _ = utils.Take(remainingData, 2)
		remainingData, sensitiveSizeBytes, _ = utils.Take(remainingData, 2)
		item.Offset = binary.LittleEndian.Uint16(sensitiveOffsetBytes)
		item.MessageStringSize = binary.LittleEndian.Uint16(sensitiveSizeBytes)
	}

	return item, remainingData
}

// ParseItemString parses a string from the firehose data based on the item type and message size
func ParseItemString(data []byte, itemType uint8, messageSize uint16) ([]byte, string, error) {
	if messageSize > uint16(len(data)) {
		return utils.ExtractStringSize(data, uint64(messageSize))
	}
	data, messageData, _ := utils.Take(data, int(messageSize))

	// 0x30, 0x31, and 0x32 represent arbitrary data, need to be decoded again
	if slices.Contains(arbitrary, itemType) {
		return data, base64.StdEncoding.EncodeToString(messageData), nil
	}

	if itemType == Base64Raw {
		return data, base64.StdEncoding.EncodeToString(messageData), nil
	}

	_, messageString, _ := utils.ExtractStringSize(messageData, uint64(messageSize))
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
