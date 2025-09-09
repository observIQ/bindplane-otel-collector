// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

type FirehosePreamble struct {
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
	publicData               []FirehoseEntry
}

const (
	UnknownRemnantData uint8 = 0x0
	PrivateNumber      uint8 = 0x1
	Base64Raw          uint8 = 0xf2
)

var (
	LogTypes       = [5]uint8{0x2, 0x6, 0x4, 0x7, 0x3}
	StringItem     = [8]uint8{0x20, 0x22, 0x40, 0x42, 0x30, 0x31, 0x32, 0xf2}
	PrivateStrings = [7]uint8{0x21, 0x25, 0x35, 0x31, 0x41, 0x81, 0xf1}
	PrecisionItems = []uint8{0x10, 0x12}
	SensitiveItems = []uint8{0x5, 0x45, 0x85}
	Arbitrary      = []uint8{0x30, 0x31, 0x32}
)

// FirehoseEntry represents a parsed firehose log entry
type FirehoseEntry struct {
	ActivityType            uint8
	LogType                 uint8
	Flags                   uint16
	FormatStringLocation    uint32
	ThreadID                uint64
	ContinousTimeDelta      uint32
	ContinousTimeDeltaUpper uint16
	DataSize                uint16
	FirehoseActivity        FirehoseActivity
	FirehoseNonActivity     FirehoseNonActivity
	FirehoseLoss            FirehoseLoss
	FirehoseSignpost        FirehoseSignpost
	FirehoseTrace           FirehoseTrace
	UnknownItem             uint8
	NumberItems             uint8
	Message                 FirehoseItemData
}

type FirehoseItemType struct {
	ItemType          uint8
	ItemSize          uint16
	Offset            uint16
	MessageStringSize uint16
	MessageStrings    string
}

type FirehoseItemData struct {
	ItemInfo         []FirehoseItemInfo
	BacktraceStrings []string
}

// Dummy types for testing
type FirehoseNonActivity struct {
	PrivateStringsSize   uint16
	PrivateStringsOffset uint16
}

type FirehoseLoss struct{}

type FirehoseSignpost struct{}

type FirehoseTrace struct{}

// FirehoseItemInfo represents a parsed message item from firehose entry data
// Based on the Rust implementation's FirehoseItemInfo struct
type FirehoseItemInfo struct {
	MessageStrings string
	ItemType       uint8
	ItemSize       uint16
}

func ParseFirehosePreamble(data []byte) (FirehosePreamble, []byte) {
	var preamble FirehosePreamble

	chunkTag, data, _ := Take(data, 4)
	chunkSubTag, data, _ := Take(data, 4)
	chunkDataSize, data, _ := Take(data, 8)
	firstProcID, data, _ := Take(data, 8)
	secondProcID, data, _ := Take(data, 4)
	ttl, data, _ := Take(data, 1)
	collapsed, data, _ := Take(data, 1)
	unknown, data, _ := Take(data, 2)
	publicDataSize, data, _ := Take(data, 2)
	privateDataVirtualOffset, data, _ := Take(data, 2)
	unknown2, data, _ := Take(data, 2)
	unknown3, data, _ := Take(data, 2)
	baseContinuousTime, logData, _ := Take(data, 8)

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
	publicData, data, _ := Take(logData, int(preamble.publicDataSize)-publicDataSizeOffset)

	// Go through all the public data associated with log Firehose entry
	for len(publicData) > 0 {
		firehosePublicData, firehoseInput := ParseFirehoseEntry(publicData)
		publicData = firehoseInput

		if contains(LogTypes[:], firehosePublicData.ActivityType) || len(publicData) < 24 {
			// If the activity type is unknown remnant data, break
			if firehosePublicData.ActivityType == UnknownRemnantData {
				break
			}
			if preamble.privateDataVirtualOffset != 0x1000 {
				privateDataOffset := 0x1000 - preamble.privateDataVirtualOffset

				if len(data) > int(privateDataOffset) && len(publicData) == 0 {
					leftoverDataSize := len(data) - int(privateDataOffset)
					privateData, _, _ := Take(data, leftoverDataSize)
					data = privateData
				} else {
					// If log data and public data are the same size, use private data offset to calculate the private data
					if len(logData) == int(preamble.publicDataSize)-publicDataSizeOffset {
						_, privateInputData, _ := Take(logData, int(preamble.privateDataVirtualOffset)-publicDataSizeOffset-len(publicData))
						data = privateInputData
					} else {
						// If we have private data, then any leftover public data is actually prepended to the private data
						_, privateInputData, _ := Take(logData, int(preamble.publicDataSize)-publicDataSizeOffset-len(publicData))
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
			// Rust implementation claims that only non-activity firehose entries have private strings
			if data.FirehoseNonActivity.PrivateStringsSize == 0 {
				continue
			}
			stringOffset := data.FirehoseNonActivity.PrivateStringsOffset - preamble.privateDataVirtualOffset
			_, privateStringStart, _ := Take(privateInput, int(stringOffset))
			ParsePrivateData(privateStringStart, &data.Message)
		}
		data = privateInput
	}

	return preamble, data
}

// contains checks if a slice contains a specific value
func contains(slice []uint8, value uint8) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func ParseFirehoseEntry(data []byte) (FirehoseEntry, []byte) {
	firehoseResult := FirehoseEntry{}

	unknownLogActivityType, data, _ := Take(data, 1)
	unknownLogType, data, _ := Take(data, 1)
	flags, data, _ := Take(data, 2)
	formatStringLocation, data, _ := Take(data, 4)
	threadID, data, _ := Take(data, 8)
	continousTimeDelta, data, _ := Take(data, 4)
	continousTimeDeltaUpper, data, _ := Take(data, 2)
	dataSize, data, _ := Take(data, 2)

	firehoseResult.ActivityType = unknownLogActivityType[0]
	firehoseResult.LogType = unknownLogType[0]
	firehoseResult.Flags = binary.LittleEndian.Uint16(flags)
	firehoseResult.FormatStringLocation = binary.LittleEndian.Uint32(formatStringLocation)
	firehoseResult.ThreadID = binary.LittleEndian.Uint64(threadID)
	firehoseResult.ContinousTimeDelta = binary.LittleEndian.Uint32(continousTimeDelta)
	firehoseResult.ContinousTimeDeltaUpper = binary.LittleEndian.Uint16(continousTimeDeltaUpper)
	firehoseResult.DataSize = binary.LittleEndian.Uint16(dataSize)

	firehoseData, data, _ := Take(data, int(firehoseResult.DataSize))

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
		nonActivity, nonActivityData := ParseFirehoseNonActivity(firehoseData, firehoseResult.Flags, unknownLogType[0])
		firehoseData = nonActivityData
		firehoseResult.FirehoseNonActivity = nonActivity
	} else if unknownLogActivityType[0] == signpost {
		signpost, signpostData := ParseFirehoseSignpost(firehoseData, firehoseResult.Flags, unknownLogType[0])
		firehoseData = signpostData
		firehoseResult.FirehoseSignpost = signpost
	} else if unknownLogActivityType[0] == loss {
		loss, lossData := ParseFirehoseLoss(firehoseData, unknownLogType[0])
		firehoseData = lossData
		firehoseResult.FirehoseLoss = loss
	} else if unknownLogActivityType[0] == trace {
		trace, traceData := ParseFirehoseTrace(firehoseData, unknownLogType[0])
		firehoseData = traceData
		firehoseResult.FirehoseTrace = trace
	}

	return firehoseResult, data
}

func ParsePrivateData(data []byte, firehoseItemData *FirehoseItemData) ([]byte, error) {
	privateStringStart := data

	for _, firehoseInfo := range firehoseItemData.ItemInfo {
		if contains(PrivateStrings[:], firehoseInfo.ItemType) {
			if firehoseInfo.ItemType == PrivateStrings[3] || firehoseInfo.ItemType == PrivateStrings[4] {
				if len(privateStringStart) < int(firehoseInfo.ItemSize) {
					pointerObject, privateData, _ := Take(privateStringStart, len(privateStringStart))
					privateStringStart = privateData
					firehoseInfo.MessageStrings = base64.StdEncoding.EncodeToString(pointerObject)
					continue
				}

				pointerObject, privateData, _ := Take(privateStringStart, int(firehoseInfo.ItemSize))
				privateStringStart = privateData
				firehoseInfo.MessageStrings = base64.StdEncoding.EncodeToString(pointerObject)
				continue
			}
			nullPrivate := uint16(0)
			if firehoseInfo.ItemSize == nullPrivate {
				firehoseInfo.MessageStrings = "<private>"
			} else {
				privateData, messageString, _ := ExtractStringSize(privateStringStart, uint64(firehoseInfo.ItemSize))
				privateStringStart = privateData
				firehoseInfo.MessageStrings = messageString
			}
		} else if firehoseInfo.ItemType == PrivateNumber {
			nullPrivate := uint16(0x8000)
			if firehoseInfo.ItemSize == nullPrivate {
				firehoseInfo.MessageStrings = "<private>"
			} else {
				privateData, privateNumber := ParseItemNumber(privateStringStart, firehoseInfo.ItemSize)
				privateStringStart = privateData
				firehoseInfo.MessageStrings = fmt.Sprintf("%d", privateNumber)
			}
		}
	}
	return privateStringStart, nil
}

// TODO: Add test for this function
// Parse Backtrace data for log entry (chunk). This only exists if `has_context_data` flag is set
func GetBacktraceData(data []byte) ([]byte, []string, error) {
	// Skip 3 unknown bytes
	_, input, _ := Take(data, 3)

	// Read counts
	uuidCountBytes, input, _ := Take(input, 1)
	offsetCountBytes, input, _ := Take(input, 2)
	uuidCount := int(uuidCountBytes[0])
	offsetCount := int(binary.LittleEndian.Uint16(offsetCountBytes))

	// Read UUID vector (128-bit big-endian UUIDs)
	var uuidVec []uint64
	for i := 0; i < uuidCount; i++ {
		uuidBytes, remaining, _ := Take(input, 16) // 128 bits = 16 bytes
		input = remaining
		// Convert 128-bit UUID to two 64-bit values (big-endian)
		uuidHigh := binary.BigEndian.Uint64(uuidBytes[:8])
		_ = binary.BigEndian.Uint64(uuidBytes[8:])
		// For simplicity, we'll use the high part as the main UUID
		uuidVec = append(uuidVec, uuidHigh)
	}

	// Read offsets vector (32-bit little-endian)
	var offsetsVec []uint32
	for i := 0; i < offsetCount; i++ {
		offsetBytes, remaining, _ := Take(input, 4)
		input = remaining
		offset := binary.LittleEndian.Uint32(offsetBytes)
		offsetsVec = append(offsetsVec, offset)
	}

	// Read indexes (8-bit)
	var indexes []uint8
	for i := 0; i < offsetCount; i++ {
		indexBytes, remaining, _ := Take(input, 1)
		input = remaining
		indexes = append(indexes, indexBytes[0])
	}

	// Build backtrace strings
	var backtraceData []string
	for i, idx := range indexes {
		var uuid uint64
		if int(idx) < len(uuidVec) {
			uuid = uuidVec[idx]
		}
		var offset uint32
		if i < len(offsetsVec) {
			offset = offsetsVec[i]
		}

		backtraceStr := fmt.Sprintf("\"%X\" +0x%x", uuid, offset)
		backtraceData = append(backtraceData, backtraceStr)
	}

	// Calculate padding size (align to 4-byte boundary)
	paddingSize := paddingSizeFour(uint64(offsetCount))
	if paddingSize > uint64(^uint(0)>>1) { // Check if larger than max int
		return input, backtraceData, fmt.Errorf("u64 is bigger than system int")
	}

	// Skip padding bytes
	_, backtraceInput, _ := Take(input, int(paddingSize))
	input = backtraceInput

	return input, backtraceData, nil
}

func ParseFirehoseMessageItems(data []byte, numItems uint8, flags uint16) (FirehoseItemData, []byte) {
	itemCount := 0
	itemsData := []FirehoseItemType{}

	firehoseInput := data
	firehoseItemData := FirehoseItemData{}
	numberItemType := []uint8{0x0, 0x2}
	objectItems := []uint8{0x40, 0x42}

	for itemCount < int(numItems) {
		item, itemInput := GetFirehoseItems(firehoseInput)
		firehoseInput = itemInput

		if contains(PrecisionItems, item.ItemType) {
			itemsData = append(itemsData, item)
			itemCount += 1
			continue
		}

		if contains(numberItemType, item.ItemType) {
			itemValueInput, messageNumber := ParseItemNumber(firehoseInput, uint16(item.ItemSize))
			item.MessageStrings = fmt.Sprintf("%d", messageNumber)
			firehoseInput = itemValueInput
			itemCount += 1
			itemsData = append(itemsData, item)
			continue
		}

		if item.MessageStringSize == 0 && contains(objectItems, item.ItemType) {
			item.MessageStrings = "(null)"
		}
		itemsData = append(itemsData, item)
		itemCount += 1
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
		parsedBacktraceSig, _, _ := Take(firehoseInput, backtraceSignatureSize)
		if len(parsedBacktraceSig) == len(backtraceSignature) &&
			parsedBacktraceSig[0] == backtraceSignature[0] &&
			parsedBacktraceSig[1] == backtraceSignature[1] &&
			parsedBacktraceSig[2] == backtraceSignature[2] {
			backtraceInput, backtraceData, _ := GetBacktraceData(firehoseInput)
			firehoseInput = backtraceInput
			firehoseItemData.BacktraceStrings = backtraceData
		}
	}

	for _, item := range itemsData {
		if contains(numberItemType, item.ItemType) {
			continue
		}

		if contains(PrivateStrings[:], item.ItemType) || contains(SensitiveItems, item.ItemType) {
			item.MessageStrings = "<private>"
			continue
		}

		if item.ItemType == PrivateNumber {
			continue
		}

		if contains(PrecisionItems, item.ItemType) {
			continue
		}

		if item.MessageStringSize == 0 && len(item.MessageStrings) != 0 {
			continue
		}

		if len(firehoseInput) == 0 {
			break
		}

		if contains(StringItem[:], item.ItemType) {
			itemValueInput, messageString, _ := ParseItemString(firehoseInput, item.ItemType, item.MessageStringSize)
			firehoseInput = itemValueInput
			item.MessageStrings = messageString
		} else {
			// TODO: error
		}
	}

	for _, item := range itemsData {
		firehoseItemData.ItemInfo = append(firehoseItemData.ItemInfo, FirehoseItemInfo{
			MessageStrings: item.MessageStrings,
			ItemType:       item.ItemType,
			ItemSize:       item.ItemSize,
		})
	}
	return firehoseItemData, firehoseInput
}

func GetFirehoseItems(data []byte) (FirehoseItemType, []byte) {
	itemType, firehoseInput, _ := Take(data, 1)
	itemSize, firehoseInput, _ := Take(firehoseInput, 1)
	item := FirehoseItemType{
		ItemType: itemType[0],
		ItemSize: binary.LittleEndian.Uint16(itemSize),
	}

	if contains(StringItem[:], item.ItemType) || item.ItemType == PrivateNumber {
		messageOffset, firehoseInput, _ := Take(firehoseInput, 2)
		messageSize, firehoseInput, _ := Take(firehoseInput, 2)
		item.Offset = binary.LittleEndian.Uint16(messageOffset)
		item.MessageStringSize = binary.LittleEndian.Uint16(messageSize)
	}

	// Precision items just contain the length for the actual item. Ex: %*s
	if contains(PrecisionItems, item.ItemType) {
		_, firehoseInput, _ = Take(firehoseInput, int(item.ItemSize))
	}

	if contains(SensitiveItems, item.ItemType) {
		messageOffset, firehoseInput, _ := Take(firehoseInput, 2)
		messageSize, firehoseInput, _ := Take(firehoseInput, 2)
		item.Offset = binary.LittleEndian.Uint16(messageOffset)
		item.MessageStringSize = binary.LittleEndian.Uint16(messageSize)
	}

	return item, firehoseInput
}

// TODO: Add test for this function
func ParseItemString(data []byte, itemType uint8, messageSize uint16) ([]byte, string, error) {
	if messageSize > uint16(len(data)) {
		return ExtractStringSize(data, uint64(messageSize))
	}
	messageData, data, _ := Take(data, int(messageSize))

	// 0x30, 0x31, and 0x32 represent arbitrary data, need to be decoded again
	if contains(Arbitrary, itemType) {
		return data, base64.StdEncoding.EncodeToString(messageData), nil
	}

	if itemType == Base64Raw {
		return data, base64.StdEncoding.EncodeToString(messageData), nil
	}

	_, messageString, _ := ExtractStringSize(messageData, uint64(messageSize))
	return data, messageString, nil
}

// TODO: Add test for this function
func ParseItemNumber(data []byte, itemSize uint16) ([]byte, uint64) {
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

// // parseFirehoseMessageItems parses message items from firehose entry data
// // Based on the Rust implementation's collect_items and parse_private_data logic
// func ParseFirehoseMessageItemsv1(data []byte, itemCount uint8) []FirehoseItemInfo {
// 	var items []FirehoseItemInfo
// 	offset := 0

// 	// Item type constants from rust implementation
// 	stringItems := map[uint8]bool{
// 		0x20: true, 0x21: true, 0x22: true, 0x25: true, 0x40: true, 0x41: true, 0x42: true,
// 		0x30: true, 0x31: true, 0x32: true, 0xf2: true, 0x35: true, 0x81: true, 0xf1: true,
// 	}

// 	numberItems := map[uint8]bool{
// 		0x0: true, 0x2: true,
// 	}

// 	precisionItems := map[uint8]bool{
// 		0x10: true, 0x12: true,
// 	}

// 	sensitiveItems := map[uint8]bool{
// 		0x5: true, 0x45: true, 0x85: true,
// 	}

// 	objectItems := map[uint8]bool{
// 		0x40: true, 0x42: true,
// 	}

// 	privateNumber := uint8(0x1)

// 	// First pass: parse item metadata
// 	for i := uint8(0); i < itemCount && offset < len(data); i++ {
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
// 			item.MessageOffset = binary.LittleEndian.Uint16(data[offset : offset+2])
// 			item.MessageStringSize = binary.LittleEndian.Uint16(data[offset+2 : offset+4])
// 			offset += 4
// 		}

// 		// Precision items just contain the length for the actual item
// 		if precisionItems[item.ItemType] {
// 			if offset+int(item.ItemSize) <= len(data) {
// 				offset += int(item.ItemSize)
// 			}
// 		}

// 		// Sensitive items have 4 bytes of metadata
// 		if sensitiveItems[item.ItemType] {
// 			if offset+4 > len(data) {
// 				break
// 			}
// 			item.MessageOffset = binary.LittleEndian.Uint16(data[offset : offset+2])
// 			item.MessageStringSize = binary.LittleEndian.Uint16(data[offset+2 : offset+4])
// 			offset += 4
// 		}

// 		items = append(items, item)
// 	}

// 	// Second pass: extract number values immediately (they follow the item metadata)
// 	for i := range items {
// 		item := &items[i]

// 		if numberItems[item.ItemType] {
// 			// Parse number based on item size
// 			if offset+int(item.ItemSize) <= len(data) {
// 				if item.ItemSize >= 8 {
// 					value := binary.LittleEndian.Uint64(data[offset : offset+8])
// 					item.MessageStrings = fmt.Sprintf("%d", value)
// 					offset += 8
// 				} else if item.ItemSize >= 4 {
// 					value := binary.LittleEndian.Uint32(data[offset : offset+4])
// 					item.MessageStrings = fmt.Sprintf("%d", value)
// 					offset += 4
// 				} else if item.ItemSize >= 2 {
// 					value := binary.LittleEndian.Uint16(data[offset : offset+2])
// 					item.MessageStrings = fmt.Sprintf("%d", value)
// 					offset += 2
// 				} else if item.ItemSize >= 1 {
// 					item.MessageStrings = fmt.Sprintf("%d", data[offset])
// 					offset++
// 				}
// 			}
// 		}
// 	}

// 	// Third pass: extract string data from the end of the buffer
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
// 		} else if sensitiveItems[item.ItemType] {
// 			// Sensitive items are private
// 			item.MessageStrings = "<private>"
// 		} else if objectItems[item.ItemType] && item.MessageStringSize == 0 {
// 			// Object items with size 0 are "(null)"
// 			item.MessageStrings = "(null)"
// 		} else if precisionItems[item.ItemType] {
// 			// Precision items just contain the length
// 			item.MessageStrings = fmt.Sprintf("%d", item.ItemSize)
// 		}
// 	}

// 	return items
// }

// // ParseFirehoseChunk parses a Firehose chunk (0x6001 and variants) containing multiple individual log entries
// // Returns a slice of TraceV3Entry representing each individual log event within the chunk
// func ParseFirehoseChunk(data []byte, entry *TraceV3Entry, header *TraceV3Header, timesyncData map[string]*TimesyncBoot) []*TraceV3Entry {
// 	var entries []*TraceV3Entry

// 	// Parse firehose chunk with enhanced subsystem/category mapping

// 	if len(data) < 48 { // Minimum size for firehose preamble
// 		entry.Message = fmt.Sprintf("Firehose chunk too small: %d bytes", len(data))
// 		return []*TraceV3Entry{entry}
// 	}

// 	// Parse firehose preamble fields (same structure as before)
// 	firstProcID := binary.LittleEndian.Uint64(data[16:24])
// 	secondProcID := binary.LittleEndian.Uint32(data[24:28])
// 	ttl := data[28]
// 	collapsed := data[29]
// 	baseContinuousTime := binary.LittleEndian.Uint64(data[40:48])

// 	// Look for public data section
// 	if len(data) < 52 {
// 		entry.Message = fmt.Sprintf("Firehose chunk missing public data section: %d bytes", len(data))
// 		return []*TraceV3Entry{entry}
// 	}

// 	publicDataSize := binary.LittleEndian.Uint16(data[48:50])
// 	privateDataOffset := binary.LittleEndian.Uint16(data[50:52])

// 	// publicDataSize includes 16 bytes for the firehose preamble itself

// 	// Check for private data (0x1000 = 4096 means NO private data)
// 	hasPrivateData := privateDataOffset != 0x1000
// 	var privateData []byte

// 	if hasPrivateData {
// 		// Calculate private data location
// 		// private_data_offset = 0x1000 - private_data_virtual_offset
// 		actualPrivateDataOffset := 0x1000 - int(privateDataOffset)

// 		// Private data comes after public data
// 		privateDataStart := 52 + int(publicDataSize)
// 		if len(data) > privateDataStart+actualPrivateDataOffset {
// 			privateData = data[privateDataStart+actualPrivateDataOffset:]
// 		}
// 	}

// 	// Focus on private data parsing since that's where the real log content is
// 	if hasPrivateData && len(privateData) > 0 {
// 		// Parse private data to extract log messages
// 		privateEntries := parsePrivateData(privateData, privateDataOffset, firstProcID, secondProcID, baseContinuousTime, header, timesyncData)
// 		entries = append(entries, privateEntries...)
// 	}

// 	// Try to parse public data - enhanced to handle multiple firehose entries
// 	if publicDataSize > 4 && len(data) >= int(52+publicDataSize) {
// 		var actualPublicDataSize uint16
// 		var publicDataStart int

// 		if publicDataSize > 16 {
// 			actualPublicDataSize = publicDataSize - 16
// 			publicDataStart = 52
// 		} else {
// 			actualPublicDataSize = publicDataSize
// 			publicDataStart = 52
// 		}

// 		if len(data) >= int(publicDataStart)+int(actualPublicDataSize) {
// 			publicData := data[publicDataStart : publicDataStart+int(actualPublicDataSize)]
// 			// Enhanced parsing to handle multiple firehose entries in the public data section
// 			// This matches the rust implementation's approach of processing all entries in a loop
// 			publicEntries := parseMultipleFirehoseEntries(publicData, header, firstProcID, secondProcID, baseContinuousTime, ttl, collapsed, timesyncData)
// 			entries = append(entries, publicEntries...)
// 		}
// 	}

// 	// If no individual entries found, create a summary entry with private data info
// 	if len(entries) == 0 {
// 		entry.ThreadID = firstProcID
// 		entry.ProcessID = secondProcID
// 		// For summary entry, use baseContinuousTime as both the delta time and preamble time since there's no individual entry delta
// 		entry.Timestamp = convertMachTimeToUnixNanosWithTimesync(baseContinuousTime, header.BootUUID, baseContinuousTime, timesyncData)

// 		entry.Message = fmt.Sprintf("Firehose chunk parsed: size=%d entries=%d", len(data), len(entries))
// 		entry.Level = "Info"
// 		entry.Category = "parsing_summary"
// 		entry.Subsystem = "com.apple.firehose.summary"
// 		entries = []*TraceV3Entry{entry}
// 	} else {
// 		// Return only the individual entries, no summary entry
// 		// This replaces summary entries with actual log entries as requested
// 	}

// 	return entries
// }

// // parseMultipleFirehoseEntries parses multiple individual firehose entries from the public data section
// // This implements the rust approach of processing all entries in a loop until the data is exhausted
// func parseMultipleFirehoseEntries(publicData []byte, header *TraceV3Header, firstProcID uint64, secondProcID uint32, baseContinuousTime uint64, ttl, collapsed uint8, timesyncData map[string]*TimesyncBoot) []*TraceV3Entry {
// 	var entries []*TraceV3Entry
// 	offset := 0

// 	// Process all firehose entries in the public data section
// 	// This matches the rust implementation's while loop approach
// 	for offset < len(publicData) {
// 		// Need at least 24 bytes for a complete firehose entry header
// 		if offset+24 > len(publicData) {
// 			break
// 		}

// 		// Parse individual firehose entry header (24 bytes)
// 		logActivityType := publicData[offset]
// 		logType := publicData[offset+1]
// 		flags := binary.LittleEndian.Uint16(publicData[offset+2:])
// 		formatStringLocation := binary.LittleEndian.Uint32(publicData[offset+4:])
// 		threadID := binary.LittleEndian.Uint64(publicData[offset+8:])
// 		continuousTimeDelta := binary.LittleEndian.Uint32(publicData[offset+16:])
// 		continuousTimeDeltaUpper := binary.LittleEndian.Uint16(publicData[offset+20:])
// 		dataSize := binary.LittleEndian.Uint16(publicData[offset+22:])

// 		// Check for remnant data (rust implementation stops here)
// 		const remnantData = 0x0
// 		if logActivityType == remnantData {
// 			break
// 		}

// 		// Validate data size
// 		if dataSize > uint16(len(publicData)-offset-24) {
// 			// Data size exceeds available data, this entry is probably malformed
// 			offset += 1
// 			continue
// 		}

// 		// Verify we have enough data for this entry
// 		if offset+24+int(dataSize) > len(publicData) {
// 			// Not enough data for this entry, break out
// 			break
// 		}

// 		// Create FirehoseEntry structure
// 		firehoseEntry := &FirehoseEntry{
// 			ActivityType:         logActivityType,
// 			LogType:              logType,
// 			Flags:                flags,
// 			FormatStringLocation: formatStringLocation,
// 			ThreadID:             threadID,
// 			TimeDelta:            continuousTimeDelta,
// 			TimeDeltaUpper:       continuousTimeDeltaUpper,
// 			DataSize:             dataSize,
// 		}

// 		// Extract message data if present
// 		if dataSize > 0 {
// 			firehoseEntry.MessageData = publicData[offset+24 : offset+24+int(dataSize)]
// 		}

// 		// Parse the individual firehose entry
// 		entry := parseSingleFirehoseEntry(firehoseEntry, header, firstProcID, secondProcID, baseContinuousTime, timesyncData)
// 		if entry != nil {
// 			entries = append(entries, entry)
// 		}

// 		// Move to next entry (24-byte header + data_size bytes)
// 		offset += 24 + int(dataSize)

// 		// Safety check to prevent infinite loops
// 		if len(entries) >= 10000 {
// 			break
// 		}
// 	}

// 	return entries
// }

// // parseSingleFirehoseEntry parses a single firehose entry and returns a TraceV3Entry
// func parseSingleFirehoseEntry(firehoseEntry *FirehoseEntry, header *TraceV3Header, firstProcID uint64, secondProcID uint32, baseContinuousTime uint64, timesyncData map[string]*TimesyncBoot) *TraceV3Entry {
// 	// Extract number of items from firehose data
// 	var itemCount uint8 = 0
// 	if len(firehoseEntry.MessageData) >= 2 {
// 		// The number of items is the second byte in the message data
// 		itemCount = firehoseEntry.MessageData[1]
// 	}

// 	// Calculate the combined continuous time (6 bytes total: 4 + 2)
// 	combinedTimeDelta := uint64(firehoseEntry.TimeDelta) | (uint64(firehoseEntry.TimeDeltaUpper) << 32)

// 	// Parse subsystem ID using enhanced parsing
// 	subsystemID := parseSubsystemFromEntry(firehoseEntry)
// 	firehoseEntry.SubsystemID = subsystemID

// 	// Use the original parsing as fallback
// 	_, actualSubsystemName, actualCategoryName, useSharedCache := parseFirehoseEntrySubsystem(firehoseEntry.MessageData, firehoseEntry.Flags, firstProcID, secondProcID)

// 	// Use catalog to resolve process information if available
// 	actualPID := secondProcID
// 	subsystemName := actualSubsystemName // Use parsed subsystem name as default
// 	categoryName := actualCategoryName   // Use parsed category name as default

// 	if GlobalCatalog != nil {
// 		// Try to resolve using the chunk's process IDs
// 		if resolvedPID := GlobalCatalog.GetPID(firstProcID, secondProcID); resolvedPID != 0 {
// 			actualPID = resolvedPID
// 		}

// 		// Try to resolve subsystem information using the actual subsystem ID from the log entry
// 		if subsystemID != 0 { // Only lookup if we have a valid subsystem ID
// 			if subsysInfo := GlobalCatalog.GetSubsystem(subsystemID, firstProcID, secondProcID); subsysInfo.Subsystem != "Unknown subsystem" && subsysInfo.Subsystem != "" {
// 				subsystemName = subsysInfo.Subsystem
// 				if subsysInfo.Category != "" {
// 					categoryName = subsysInfo.Category
// 				}
// 			}
// 		}
// 	}

// 	// Create individual log entry
// 	// According to rust implementation:
// 	// firehose_log_delta_time = firehose_preamble_time + firehose_log_entry_continous_time
// 	// where firehose_preamble_time = baseContinuousTime and firehose_log_entry_continous_time = combinedTimeDelta
// 	firehoseLogDeltaTime := baseContinuousTime + combinedTimeDelta
// 	logEntry := &TraceV3Entry{
// 		Type:         0x6001,                                                                                                          // Firehose chunk type
// 		Size:         uint32(24 + firehoseEntry.DataSize),                                                                             // Header + data size (24-byte header)
// 		Timestamp:    convertMachTimeToUnixNanosWithTimesync(firehoseLogDeltaTime, header.BootUUID, baseContinuousTime, timesyncData), // Use proper timesync-converted timestamp with correct parameters
// 		ThreadID:     firehoseEntry.ThreadID,
// 		ProcessID:    actualPID, // Use catalog-resolved PID when available
// 		ChunkType:    "firehose",
// 		Subsystem:    subsystemName, // Use catalog-resolved subsystem when available
// 		Category:     categoryName,  // Use catalog-resolved category when available
// 		TimezoneName: extractTimezoneName(header.TimezonePath),
// 	}

// 	// Determine log level and message type based on log type and activity type
// 	logEntry.MessageType = getLogType(firehoseEntry.LogType, firehoseEntry.ActivityType)
// 	logEntry.Level = logEntry.MessageType // Keep Level for backward compatibility

// 	// Determine event type based on activity type
// 	logEntry.EventType = getEventType(firehoseEntry.ActivityType)

// 	// Category should come from subsystem data, not be hardcoded based on activity type
// 	// For now, set to empty string like the rust implementation does initially
// 	// The category will be populated from subsystem data if available
// 	logEntry.Category = ""

// 	// Determine shared cache usage with enhanced method
// 	useSharedCacheEnhanced := shouldUseSharedCache(firehoseEntry)
// 	if !useSharedCacheEnhanced {
// 		useSharedCacheEnhanced = useSharedCache // Fallback to original logic
// 	}

// 	// Resolve format string using UUID references (with enhanced shared cache detection)
// 	formatData := GetFormatString(firehoseEntry.FormatStringLocation, firstProcID, secondProcID, useSharedCacheEnhanced)

// 	// Extract message content using enhanced parsing
// 	messageContent := extractMessageContent(firehoseEntry)

// 	// Create descriptive message with resolved format string
// 	catalogInfo := ""
// 	if GlobalCatalog != nil {
// 		if euid := GlobalCatalog.GetEUID(firstProcID, secondProcID); euid != 0 {
// 			catalogInfo = fmt.Sprintf(" uid=%d", euid)
// 		}
// 	}

// 	// Parse message items for format string substitution
// 	messageItems := parseFirehoseMessageItems(firehoseEntry.MessageData, itemCount)

// 	// Use resolved format string and process information
// 	if formatData.FormatString != "" && formatData.FormatString != fmt.Sprintf("raw_format_0x%x", firehoseEntry.FormatStringLocation) {
// 		// We have a resolved format string - try to format it with message items
// 		formattedMessage := formatFirehoseLogMessage(formatData.FormatString, messageItems)
// 		if formattedMessage != "" {
// 			// Successfully formatted the message
// 			logEntry.Message = formattedMessage
// 		} else {
// 			// Formatting failed, use format string with message content
// 			logEntry.Message = fmt.Sprintf("Format: %s | Process: %s | Thread: %d | %s%s",
// 				formatData.FormatString, formatData.Process, firehoseEntry.ThreadID, messageContent, catalogInfo)
// 		}
// 	} else {
// 		// Enhanced fallback with activity type details and debug info
// 		catalogDebug := ""
// 		if GlobalCatalog != nil {
// 			catalogDebug = fmt.Sprintf(" [catalog_procs=%d]", len(GlobalCatalog.ProcessInfoEntries))
// 		}
// 		// If we have message items but no format string, try to extract meaningful content
// 		if len(messageItems) > 0 {
// 			// Try to build a meaningful message from the items
// 			var messageParts []string
// 			for _, item := range messageItems {
// 				if item.MessageStrings != "" && item.MessageStrings != "<private>" && isPrintableString(item.MessageStrings) {
// 					messageParts = append(messageParts, item.MessageStrings)
// 				}
// 			}
// 			if len(messageParts) > 0 {
// 				logEntry.Message = strings.Join(messageParts, " ")
// 			} else if messageContent != "" && !strings.Contains(messageContent, "data_size=") {
// 				logEntry.Message = messageContent
// 			} else {
// 				logEntry.Message = fmt.Sprintf("Firehose %s: level=%s flags=0x%x format=0x%x thread=%d delta=%d subsys_id=%d proc_ids=%d/%d resolved_subsys=%s%s%s | %s",
// 					mapActivityTypeToString(firehoseEntry.ActivityType), logEntry.Level, firehoseEntry.Flags, firehoseEntry.FormatStringLocation,
// 					firehoseEntry.ThreadID, combinedTimeDelta, subsystemID, firstProcID, secondProcID, subsystemName, catalogInfo, catalogDebug, messageContent)
// 			}
// 		} else {
// 			logEntry.Message = fmt.Sprintf("Firehose %s: level=%s flags=0x%x format=0x%x thread=%d delta=%d subsys_id=%d proc_ids=%d/%d resolved_subsys=%s%s%s | %s",
// 				mapActivityTypeToString(firehoseEntry.ActivityType), logEntry.Level, firehoseEntry.Flags, firehoseEntry.FormatStringLocation,
// 				firehoseEntry.ThreadID, combinedTimeDelta, subsystemID, firstProcID, secondProcID, subsystemName, catalogInfo, catalogDebug, messageContent)
// 		}
// 	}

// 	return logEntry
// }

// // parseIndividualFirehoseEntries parses multiple individual log entries from the firehose public data section
// // This is the legacy function - kept for backward compatibility
// func parseIndividualFirehoseEntries(publicData []byte, header *TraceV3Header, firstProcID uint64, secondProcID uint32, baseContinuousTime uint64, ttl, collapsed uint8, timesyncData map[string]*TimesyncBoot) []*TraceV3Entry {
// 	var entries []*TraceV3Entry
// 	offset := 0

// 	// Debug entries removed - return only actual log entries

// 	// Remove restrictive log type validation - let the rust implementation handle this
// 	// The rust implementation processes all log types and maps them appropriately

// 	const remnantData = 0x0 // When we encounter this, stop parsing

// 	// Parse individual entries from public data
// 	for offset < len(publicData) {
// 		// Each individual firehose entry starts with a 24-byte header (matching rust implementation)
// 		// 1+1+2+4+8+4+2+2 = 24 bytes total
// 		// But try to parse even if we don't have full 24 bytes for debugging
// 		minHeaderSize := 8 // Minimum to get activity type, log type, flags, format string location
// 		if offset+minHeaderSize > len(publicData) {
// 			break
// 		}

// 		// Only require full header if we have enough data
// 		needFullHeader := offset+24 <= len(publicData)

// 		// Parse individual firehose entry header (matches rust parse_firehose structure exactly)
// 		logActivityType := publicData[offset]                                     // 1 byte
// 		logType := publicData[offset+1]                                           // 1 byte
// 		flags := binary.LittleEndian.Uint16(publicData[offset+2:])                // 2 bytes
// 		formatStringLocation := binary.LittleEndian.Uint32(publicData[offset+4:]) // 4 bytes

// 		// Only parse remaining fields if we have enough data
// 		var threadID uint64
// 		var continuousTimeDelta uint32
// 		var continuousTimeDeltaUpper uint16
// 		var dataSize uint16

// 		if needFullHeader {
// 			threadID = binary.LittleEndian.Uint64(publicData[offset+8:])                  // 8 bytes
// 			continuousTimeDelta = binary.LittleEndian.Uint32(publicData[offset+16:])      // 4 bytes
// 			continuousTimeDeltaUpper = binary.LittleEndian.Uint16(publicData[offset+20:]) // 2 bytes
// 			dataSize = binary.LittleEndian.Uint16(publicData[offset+22:])                 // 2 bytes
// 		} else {
// 			// Use defaults for partial header
// 			threadID = firstProcID
// 			continuousTimeDelta = 0
// 			continuousTimeDeltaUpper = 0
// 			dataSize = 0
// 		}

// 		// Check for remnant data (rust implementation stops here)
// 		if logActivityType == remnantData {
// 			break
// 		}

// 		// Debug entries removed - return only actual log entries

// 		// Process all log types - let the rust implementation handle the mapping
// 		// No need to skip entries based on log type

// 		// Check if remaining data is sufficient (rust implementation check)
// 		if needFullHeader && len(publicData)-offset < 24 {
// 			break
// 		}

// 		// Additional validation: check if data size is reasonable (only for full headers)
// 		if needFullHeader && dataSize > uint16(len(publicData)-offset-24) {
// 			// Data size exceeds available data, this entry is probably malformed
// 			offset += 1
// 			continue
// 		}

// 		// Verify we have enough data for this entry (only for full headers)
// 		if needFullHeader && offset+24+int(dataSize) > len(publicData) {
// 			// Not enough data for this entry, break out
// 			break
// 		}

// 		// Create FirehoseEntry structure for enhanced parsing
// 		firehoseEntry := &FirehoseEntry{
// 			ActivityType:         logActivityType,
// 			LogType:              logType,
// 			Flags:                flags,
// 			FormatStringLocation: formatStringLocation,
// 			ThreadID:             threadID,
// 			TimeDelta:            continuousTimeDelta,
// 			TimeDeltaUpper:       continuousTimeDeltaUpper,
// 			DataSize:             dataSize,
// 		}

// 		// Extract message data if present (only for full headers)
// 		if needFullHeader && dataSize > 0 {
// 			firehoseEntry.MessageData = publicData[offset+24 : offset+24+int(dataSize)]
// 		}

// 		// Extract number of items from firehose data (rust implementation approach)
// 		var itemCount uint8 = 0
// 		if needFullHeader && len(firehoseEntry.MessageData) >= 2 {
// 			// The number of items is the second byte in the message data
// 			// (first byte is unknown_item, second byte is number_items)
// 			itemCount = firehoseEntry.MessageData[1]
// 		}

// 		// Calculate the combined continuous time (6 bytes total: 4 + 2)
// 		if int(dataSize) >= 0 { // Accept any valid data size
// 			combinedTimeDelta := uint64(continuousTimeDelta) | (uint64(continuousTimeDeltaUpper) << 32)

// 			// Parse subsystem ID using enhanced parsing
// 			subsystemID := parseSubsystemFromEntry(firehoseEntry)
// 			firehoseEntry.SubsystemID = subsystemID

// 			// Use the original parsing as fallback
// 			_, actualSubsystemName, actualCategoryName, useSharedCache := parseFirehoseEntrySubsystem(firehoseEntry.MessageData, flags, firstProcID, secondProcID)

// 			// Use catalog to resolve process information if available
// 			actualPID := secondProcID
// 			subsystemName := actualSubsystemName // Use parsed subsystem name as default
// 			categoryName := actualCategoryName   // Use parsed category name as default

// 			if GlobalCatalog != nil {
// 				// Try to resolve using the chunk's process IDs
// 				if resolvedPID := GlobalCatalog.GetPID(firstProcID, secondProcID); resolvedPID != 0 {
// 					actualPID = resolvedPID
// 				}

// 				// Try to resolve subsystem information using the actual subsystem ID from the log entry
// 				if subsystemID != 0 { // Only lookup if we have a valid subsystem ID
// 					if subsysInfo := GlobalCatalog.GetSubsystem(subsystemID, firstProcID, secondProcID); subsysInfo.Subsystem != "Unknown subsystem" && subsysInfo.Subsystem != "" {
// 						subsystemName = subsysInfo.Subsystem
// 						if subsysInfo.Category != "" {
// 							categoryName = subsysInfo.Category
// 						}
// 					}
// 				}
// 			}

// 			// Create individual log entry
// 			// According to rust implementation:
// 			// firehose_log_delta_time = firehose_preamble_time + firehose_log_entry_continous_time
// 			// where firehose_preamble_time = baseContinuousTime and firehose_log_entry_continous_time = combinedTimeDelta
// 			firehoseLogDeltaTime := baseContinuousTime + combinedTimeDelta
// 			logEntry := &TraceV3Entry{
// 				Type:         0x6001,                                                                                                          // Firehose chunk type
// 				Size:         uint32(24 + dataSize),                                                                                           // Header + data size (24-byte header)
// 				Timestamp:    convertMachTimeToUnixNanosWithTimesync(firehoseLogDeltaTime, header.BootUUID, baseContinuousTime, timesyncData), // Use proper timesync-converted timestamp with correct parameters
// 				ThreadID:     threadID,
// 				ProcessID:    actualPID, // Use catalog-resolved PID when available
// 				ChunkType:    "firehose",
// 				Subsystem:    subsystemName, // Use catalog-resolved subsystem when available
// 				Category:     categoryName,  // Use catalog-resolved category when available
// 				TimezoneName: extractTimezoneName(header.TimezonePath),
// 			}

// 			// Determine log level and message type based on log type and activity type
// 			logEntry.MessageType = getLogType(logType, logActivityType)
// 			logEntry.Level = logEntry.MessageType // Keep Level for backward compatibility

// 			// Determine event type based on activity type
// 			logEntry.EventType = getEventType(logActivityType)

// 			// Category should come from subsystem data, not be hardcoded based on activity type
// 			// For now, set to empty string like the rust implementation does initially
// 			// The category will be populated from subsystem data if available
// 			logEntry.Category = ""

// 			// Determine shared cache usage with enhanced method
// 			useSharedCacheEnhanced := shouldUseSharedCache(firehoseEntry)
// 			if !useSharedCacheEnhanced {
// 				useSharedCacheEnhanced = useSharedCache // Fallback to original logic
// 			}

// 			// Resolve format string using UUID references (with enhanced shared cache detection)
// 			formatData := GetFormatString(formatStringLocation, firstProcID, secondProcID, useSharedCacheEnhanced)

// 			// Extract message content using enhanced parsing
// 			messageContent := extractMessageContent(firehoseEntry)

// 			// Create descriptive message with resolved format string
// 			catalogInfo := ""
// 			if GlobalCatalog != nil {
// 				if euid := GlobalCatalog.GetEUID(firstProcID, secondProcID); euid != 0 {
// 					catalogInfo = fmt.Sprintf(" uid=%d", euid)
// 				}
// 			}

// 			// Parse message items for format string substitution
// 			messageItems := parseFirehoseMessageItems(firehoseEntry.MessageData, itemCount)

// 			// Use resolved format string and process information
// 			if formatData.FormatString != "" && formatData.FormatString != fmt.Sprintf("raw_format_0x%x", formatStringLocation) {
// 				// We have a resolved format string - try to format it with message items
// 				formattedMessage := formatFirehoseLogMessage(formatData.FormatString, messageItems)
// 				if formattedMessage != "" {
// 					// Successfully formatted the message
// 					logEntry.Message = formattedMessage
// 				} else {
// 					// Formatting failed, use format string with message content
// 					logEntry.Message = fmt.Sprintf("Format: %s | Process: %s | Thread: %d | %s%s",
// 						formatData.FormatString, formatData.Process, threadID, messageContent, catalogInfo)
// 				}
// 			} else {
// 				// Enhanced fallback with activity type details and debug info
// 				catalogDebug := ""
// 				if GlobalCatalog != nil {
// 					catalogDebug = fmt.Sprintf(" [catalog_procs=%d]", len(GlobalCatalog.ProcessInfoEntries))
// 				}
// 				// If we have message items but no format string, try to extract meaningful content
// 				if len(messageItems) > 0 {
// 					// Try to build a meaningful message from the items
// 					var messageParts []string
// 					for _, item := range messageItems {
// 						if item.MessageStrings != "" && item.MessageStrings != "<private>" && isPrintableString(item.MessageStrings) {
// 							messageParts = append(messageParts, item.MessageStrings)
// 						}
// 					}
// 					if len(messageParts) > 0 {
// 						logEntry.Message = strings.Join(messageParts, " ")
// 					} else if messageContent != "" && !strings.Contains(messageContent, "data_size=") {
// 						logEntry.Message = messageContent
// 					} else {
// 						logEntry.Message = fmt.Sprintf("Firehose %s: level=%s flags=0x%x format=0x%x thread=%d delta=%d subsys_id=%d proc_ids=%d/%d resolved_subsys=%s%s%s | %s",
// 							mapActivityTypeToString(logActivityType), logEntry.Level, flags, formatStringLocation,
// 							threadID, combinedTimeDelta, subsystemID, firstProcID, secondProcID, subsystemName, catalogInfo, catalogDebug, messageContent)
// 					}
// 				} else {
// 					logEntry.Message = fmt.Sprintf("Firehose %s: level=%s flags=0x%x format=0x%x thread=%d delta=%d subsys_id=%d proc_ids=%d/%d resolved_subsys=%s%s%s | %s",
// 						mapActivityTypeToString(logActivityType), logEntry.Level, flags, formatStringLocation,
// 						threadID, combinedTimeDelta, subsystemID, firstProcID, secondProcID, subsystemName, catalogInfo, catalogDebug, messageContent)
// 				}
// 			}

// 			entries = append(entries, logEntry)
// 		} else {
// 			// Even if structured parsing failed, try to extract any readable content
// 			if len(publicData) > offset+24 {
// 				// Try to extract any readable strings from the data
// 				readableContent := extractReadableContentFromBytes(publicData[offset:min(offset+200, len(publicData))])
// 				if readableContent != "" {
// 					fallbackEntry := &TraceV3Entry{
// 						Type:         0x6001,
// 						Size:         uint32(min(200, len(publicData)-offset)),
// 						Timestamp:    header.ContinuousTime + uint64(len(entries))*1000000,
// 						ThreadID:     0,
// 						ProcessID:    header.LogdPID,
// 						Level:        "Info",
// 						MessageType:  "Info",
// 						EventType:    "logEvent",
// 						TimezoneName: extractTimezoneName(header.TimezonePath),
// 						ChunkType:    "firehose_fallback",
// 						Subsystem:    "com.apple.firehose.fallback",
// 						Category:     "extracted_content",
// 						Message:      fmt.Sprintf("Fallback extraction: %s", readableContent),
// 					}
// 					entries = append(entries, fallbackEntry)
// 				}
// 			}
// 		}

// 		// Move to next entry (24-byte header + data_size bytes for full headers, or minimum advance for partial)
// 		if needFullHeader {
// 			offset += 24 + int(dataSize)
// 		} else {
// 			// For partial headers, advance by minimum amount to continue searching
// 			offset += minHeaderSize
// 		}

// 		// Safety check to prevent infinite loops with reasonable limit
// 		if len(entries) >= 10000 {
// 			break
// 		}
// 	}

// 	return entries
// }

// // extractReadableContentFromBytes attempts to extract readable strings from raw bytes
// func extractReadableContentFromBytes(data []byte) string {
// 	if len(data) == 0 {
// 		return ""
// 	}

// 	var readableParts []string

// 	// Look for printable ASCII strings
// 	currentString := ""
// 	for _, b := range data {
// 		if b >= 32 && b <= 126 { // Printable ASCII range
// 			currentString += string(b)
// 		} else {
// 			if len(currentString) >= 4 { // Only keep strings of 4+ characters
// 				readableParts = append(readableParts, currentString)
// 			}
// 			currentString = ""
// 		}
// 	}

// 	// Add the last string if it's long enough
// 	if len(currentString) >= 4 {
// 		readableParts = append(readableParts, currentString)
// 	}

// 	// Return the first few readable parts
// 	if len(readableParts) > 0 {
// 		maxParts := 3
// 		if len(readableParts) < maxParts {
// 			maxParts = len(readableParts)
// 		}
// 		return strings.Join(readableParts[:maxParts], " | ")
// 	}

// 	return ""
// }

// // min returns the minimum of two integers
// func min(a, b int) int {
// 	if a < b {
// 		return a
// 	}
// 	return b
// }

// // parseFirehoseMessageData attempts to extract readable message data from firehose entry data
// func parseFirehoseMessageData(data []byte) string {
// 	if len(data) == 0 {
// 		return ""
// 	}

// 	// For now, return a simple representation - this will be enhanced later
// 	// to parse actual message items (strings, numbers, etc.) based on the format string
// 	if len(data) < 8 {
// 		return fmt.Sprintf("raw:%x", data)
// 	}

// 	// Try to extract basic message structure
// 	// The message format depends on the number of items and their types
// 	numberItems := data[1] // Second byte typically contains number of items

// 	if numberItems == 0 {
// 		return "empty"
// 	}

// 	endIdx := len(data)
// 	if endIdx > 16 {
// 		endIdx = 16
// 	}
// 	return fmt.Sprintf("items:%d raw:%x", numberItems, data[:endIdx])
// }

// // isPrintableString checks if a string contains mostly printable characters
// func isPrintableString(s string) bool {
// 	if len(s) < 3 {
// 		return false
// 	}

// 	printableCount := 0
// 	for _, r := range s {
// 		if r >= 32 && r <= 126 { // Printable ASCII range
// 			printableCount++
// 		}
// 	}

// 	return float64(printableCount)/float64(len(s)) > 0.7 // At least 70% printable
// }

// // mapActivityTypeToString maps activity type to a descriptive string
// func mapActivityTypeToString(activityType uint8) string {
// 	switch activityType {
// 	case 0x2:
// 		return "Activity"
// 	case 0x4:
// 		return "Log"
// 	case 0x6:
// 		return "Signpost"
// 	case 0x7:
// 		return "Loss"
// 	case 0x3:
// 		return "Trace"
// 	default:
// 		return fmt.Sprintf("Unknown(0x%x)", activityType)
// 	}
// }

// // parsePrivateData extracts log messages from firehose private data sections
// // This is where the actual log content is stored in most firehose chunks
// func parsePrivateData(privateData []byte, privateDataOffset uint16, firstProcID uint64, secondProcID uint32, baseContinuousTime uint64, header *TraceV3Header, timesyncData map[string]*TimesyncBoot) []*TraceV3Entry {
// 	var entries []*TraceV3Entry

// 	if len(privateData) == 0 {
// 		return entries
// 	}

// 	// Skip any null padding at the beginning
// 	offset := 0
// 	for offset < len(privateData) && privateData[offset] == 0 {
// 		offset++
// 	}

// 	if offset >= len(privateData) {
// 		return entries
// 	}

// 	// Look for strings and other log content in the private data
// 	extractedStrings := extractStringsFromPrivateData(privateData[offset:])

// 	if len(extractedStrings) > 0 {
// 		// Combine related strings into a single complete log message
// 		// instead of creating separate entries for each string fragment
// 		var messageParts []string
// 		for _, str := range extractedStrings {
// 			if len(str) >= 3 && isPrintableString(str) {
// 				messageParts = append(messageParts, str)
// 			}
// 		}

// 		if len(messageParts) > 0 {
// 			// Create a single log entry with the combined message
// 			combinedMessage := strings.Join(messageParts, " ")

// 			// Use catalog to resolve process information if available
// 			actualPID := secondProcID
// 			subsystemName := "com.apple.firehose.private"
// 			categoryName := "log"

// 			if GlobalCatalog != nil {
// 				if resolvedPID := GlobalCatalog.GetPID(firstProcID, secondProcID); resolvedPID != 0 {
// 					actualPID = resolvedPID
// 				}

// 				// Try to get subsystem info (subsystem ID would come from associated public data)
// 				if subsysInfo := GlobalCatalog.GetSubsystem(0, firstProcID, secondProcID); subsysInfo.Subsystem != "Unknown subsystem" && subsysInfo.Subsystem != "" {
// 					subsystemName = subsysInfo.Subsystem
// 					if subsysInfo.Category != "" {
// 						categoryName = subsysInfo.Category
// 					}
// 				}
// 			}

// 			logEntry := &TraceV3Entry{
// 				Type:         0x6001,
// 				Size:         uint32(len(combinedMessage)),
// 				Timestamp:    convertMachTimeToUnixNanosWithTimesync(baseContinuousTime, header.BootUUID, 0, timesyncData),
// 				ThreadID:     0,
// 				ProcessID:    actualPID,
// 				ChunkType:    "firehose_private",
// 				Subsystem:    subsystemName,
// 				Category:     categoryName,
// 				Level:        "Info",
// 				MessageType:  "Log",
// 				EventType:    "firehose",
// 				TimezoneName: extractTimezoneName(header.TimezonePath),
// 				Message:      combinedMessage,
// 			}

// 			entries = append(entries, logEntry)
// 		}
// 	}

// 	// If no strings found, create a summary entry about the private data
// 	if len(entries) == 0 {
// 		logEntry := &TraceV3Entry{
// 			Type:         0x6001,
// 			Size:         uint32(len(privateData)),
// 			Timestamp:    convertMachTimeToUnixNanosWithTimesync(baseContinuousTime, header.BootUUID, 0, timesyncData),
// 			ThreadID:     0,
// 			ProcessID:    secondProcID,
// 			ChunkType:    "firehose_private",
// 			Subsystem:    "com.apple.firehose.private",
// 			Category:     "data",
// 			Level:        "Info",
// 			MessageType:  "Log",
// 			EventType:    "firehose",
// 			TimezoneName: extractTimezoneName(header.TimezonePath),
// 			Message:      fmt.Sprintf("Private data: %d bytes, privateOffset=0x%x", len(privateData), privateDataOffset),
// 		}

// 		entries = append(entries, logEntry)
// 	}

// 	return entries
// }

// // extractStringsFromPrivateData finds null-terminated strings in private data
// func extractStringsFromPrivateData(data []byte) []string {
// 	var strings []string

// 	if len(data) == 0 {
// 		return strings
// 	}

// 	start := 0
// 	for i := 0; i < len(data); i++ {
// 		if data[i] == 0 {
// 			// Found null terminator
// 			if i > start {
// 				str := string(data[start:i])
// 				// Only include strings that look like log content
// 				if isLogString(str) {
// 					strings = append(strings, str)
// 				}
// 			}
// 			start = i + 1
// 		}
// 	}

// 	// Handle case where there's no null terminator at the end
// 	if start < len(data) {
// 		str := string(data[start:])
// 		if isLogString(str) {
// 			strings = append(strings, str)
// 		}
// 	}

// 	return strings
// }

// // isLogString determines if a string looks like actual log content
// func isLogString(s string) bool {
// 	if len(s) < 3 {
// 		return false
// 	}

// 	// Check for common log content patterns
// 	logIndicators := []string{
// 		"error", "warn", "info", "debug", "trace",
// 		"failed", "success", "start", "stop", "end",
// 		"process", "service", "event", "message",
// 		"config", "init", "load", "save", "open", "close",
// 		"connect", "disconnect", "request", "response",
// 		"exception", "crash", "panic", "abort",
// 	}

// 	lowerStr := strings.ToLower(s)

// 	// Check if it contains common log indicators
// 	for _, indicator := range logIndicators {
// 		if strings.Contains(lowerStr, indicator) {
// 			return true
// 		}
// 	}

// 	// Check if it looks like a path, URL, or identifier
// 	if strings.Contains(s, "/") || strings.Contains(s, ".") || strings.Contains(s, ":") {
// 		return isPrintableString(s)
// 	}

// 	// Check if it's mostly printable and has reasonable content
// 	if isPrintableString(s) && len(s) >= 10 {
// 		// Count alphabetic characters
// 		alphaCount := 0
// 		for _, r := range s {
// 			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
// 				alphaCount++
// 			}
// 		}

// 		// Must have at least 30% alphabetic characters for longer strings
// 		return float64(alphaCount)/float64(len(s)) >= 0.3
// 	}

// 	return false
// }

// // formatFirehoseLogMessage formats a log message by substituting format specifiers with message items
// // This is a simplified version of the Rust implementation's format_firehose_log_message function
// func formatFirehoseLogMessage(formatString string, items []FirehoseItemInfo) string {
// 	if formatString == "" {
// 		return ""
// 	}

// 	// Handle empty format strings or items
// 	if len(items) == 0 {
// 		// If no items but we have a format string, return it as-is (might be a literal message)
// 		return formatString
// 	}

// 	// Simple implementation: substitute common format specifiers
// 	// This is a simplified version - the full implementation would handle all C printf formats
// 	result := formatString
// 	itemIndex := 0

// 	// Look for common format specifiers and replace them with item values
// 	formatSpecifiers := []string{
// 		"%u", "%d", "%i", "%ld", "%lu", "%lld", "%llu", "%llx", "%x", "%X",
// 		"%s", "%@", "%c", "%f", "%g", "%e",
// 		"%{public}s", "%{private}s", "%{public}@", "%{private}@",
// 		"%{public}u", "%{private}u", "%{public}d", "%{private}d",
// 		"%{public}x", "%{private}x", "%{public}llx", "%{private}llx",
// 	}

// 	for _, spec := range formatSpecifiers {
// 		for strings.Contains(result, spec) && itemIndex < len(items) {
// 			item := items[itemIndex]

// 			var replacement string

// 			// Handle different item types and format specifiers
// 			if strings.Contains(spec, "private") {
// 				replacement = "<private>"
// 			} else if strings.Contains(spec, "s") || strings.Contains(spec, "@") || strings.Contains(spec, "c") {
// 				// String types
// 				if item.MessageStrings != "" {
// 					replacement = item.MessageStrings
// 				} else {
// 					replacement = fmt.Sprintf("<string_type_0x%x>", item.ItemType)
// 				}
// 			} else {
// 				// Numeric types - try to extract number from item data
// 				if len(item.MessageData) >= 4 {
// 					switch {
// 					case strings.Contains(spec, "llu") || strings.Contains(spec, "llx"):
// 						// 64-bit unsigned
// 						if len(item.MessageData) >= 8 {
// 							value := binary.LittleEndian.Uint64(item.MessageData[:8])
// 							if strings.Contains(spec, "llx") {
// 								replacement = fmt.Sprintf("0x%x", value)
// 							} else {
// 								replacement = fmt.Sprintf("%d", value)
// 							}
// 						} else {
// 							replacement = "<missing_64bit_data>"
// 						}
// 					case strings.Contains(spec, "ld") || strings.Contains(spec, "lu"):
// 						// 32-bit or 64-bit depending on platform, assume 32-bit for safety
// 						value := binary.LittleEndian.Uint32(item.MessageData[:4])
// 						replacement = fmt.Sprintf("%d", value)
// 					case strings.Contains(spec, "x") || strings.Contains(spec, "X"):
// 						// 32-bit hex
// 						value := binary.LittleEndian.Uint32(item.MessageData[:4])
// 						replacement = fmt.Sprintf("0x%x", value)
// 					default:
// 						// Default 32-bit unsigned
// 						value := binary.LittleEndian.Uint32(item.MessageData[:4])
// 						replacement = fmt.Sprintf("%d", value)
// 					}
// 				} else {
// 					replacement = fmt.Sprintf("<numeric_type_0x%x>", item.ItemType)
// 				}
// 			}

// 			// Replace the first occurrence of this format specifier
// 			result = strings.Replace(result, spec, replacement, 1)
// 			itemIndex++

// 			// Safety check
// 			if itemIndex >= len(items) {
// 				break
// 			}
// 		}

// 		if itemIndex >= len(items) {
// 			break
// 		}
// 	}

// 	// If there are still format specifiers but no more items, replace them with placeholder
// 	for _, spec := range formatSpecifiers {
// 		for strings.Contains(result, spec) {
// 			result = strings.Replace(result, spec, "<missing_data>", 1)
// 		}
// 	}

// 	return result
// }
