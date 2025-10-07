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
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/models"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
)

// FormatAndMessage represents a formatted message with its formatter
type FormatAndMessage struct {
	Formatter string
	Message   string
}

// ExtractSharedStrings extracts the message data for a shared string firehose entry
func ExtractSharedStrings(
	provider *uuidtext.CacheProvider,
	stringOffset uint64,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *models.CatalogChunk,
	originalOffset uint64,
) (models.MessageData, error) {
	var err error

	messageData := models.MessageData{}
	// Get shared string file (DSC) associated with log entry from Catalog
	dscUUID, mainUUID := GetCatalogDSC(catalogs, firstProcID, secondProcID)

	// DSC and UUID data should already be loaded by the receiver and cached by the extension
	// If data is missing, it means the files weren't found during initialization

	if originalOffset&0x80000000 != 0 {
		if sharedString, exists := provider.CachedDSC(dscUUID); exists {
			if len(sharedString.Ranges) > 0 {
				ranges := sharedString.Ranges[0]
				messageData.FormatString = "%s"
				messageData.Library = sharedString.UUIDs[ranges.UnknownUUIDIndex].PathString
				messageData.LibraryUUID = sharedString.UUIDs[ranges.UnknownUUIDIndex].UUID
				messageData.ProcessUUID = mainUUID

				processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
				if err != nil {
					return messageData, err
				}
				messageData.Process = processString

				return messageData, nil
			}
		}
	}

	if sharedString, exists := provider.CachedDSC(dscUUID); exists {
		for _, r := range sharedString.Ranges {
			var messageStart []byte

			if stringOffset >= r.RangeOffset && stringOffset < (r.RangeOffset+uint64(r.RangeSize)) {
				offset := stringOffset - r.RangeOffset

				if offset > uint64(^uint(0)>>1) { // Check if larger than max int
					return messageData, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
				}

				messageStart, _, err = helpers.Take(r.Strings, int(offset))
				if err != nil {
					return messageData, err
				}
				messageData.FormatString, _ = helpers.ExtractString(messageStart)

				messageData.Library = sharedString.UUIDs[r.UnknownUUIDIndex].PathString
				messageData.LibraryUUID = sharedString.UUIDs[r.UnknownUUIDIndex].UUID

				messageData.ProcessUUID = mainUUID

				processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
				if err != nil {
					return messageData, err
				}
				messageData.Process = processString

				return messageData, nil
			}
		}
	}

	// There is a chance the log entry does not have a valid offset
	// Apple reports as "~~> <Invalid shared cache code pointer offset>" or <Invalid shared cache format string offset>
	if sharedString, exists := provider.CachedDSC(dscUUID); exists {
		if len(sharedString.Ranges) > 0 {
			ranges := sharedString.Ranges[0]
			messageData.Library = sharedString.UUIDs[ranges.UnknownUUIDIndex].PathString
			messageData.LibraryUUID = sharedString.UUIDs[ranges.UnknownUUIDIndex].UUID
			messageData.FormatString = "Error: Invalid shared string offset"
			messageData.ProcessUUID = mainUUID

			processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
			if err != nil {
				return messageData, err
			}
			messageData.Process = processString
			return messageData, nil
		}
	}

	// logger.Warn("Failed to get message string from Shared Strings DSC file")
	messageData.FormatString = "Unknown shared string message"
	return messageData, nil
}

// ExtractFormatStrings extracts the message data for a format UUID firehose entry
func ExtractFormatStrings(
	provider *uuidtext.CacheProvider,
	stringOffset uint64,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *models.CatalogChunk,
	originalOffset uint64,
) (models.MessageData, error) {
	var err error

	_, mainUUID := GetCatalogDSC(catalogs, firstProcID, secondProcID)

	messageData := models.MessageData{
		LibraryUUID: mainUUID,
		ProcessUUID: mainUUID,
	}

	// UUID data should already be loaded by the receiver and cached by the extension

	if originalOffset&0x80000000 != 0 {
		if data, exists := provider.CachedUUIDText(mainUUID); exists {
			// Footer data is a collection of strings that ends with the image path/library associated with strings
			processString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			messageData.Process = processString
			messageData.Library = processString
			messageData.FormatString = "%s"

			return messageData, nil
		}
	}

	if data, exists := provider.CachedUUIDText(mainUUID); exists {
		stringStart := uint32(0)
		for _, entry := range data.EntryDescriptors {
			var messageStart []byte
			if entry.RangeStartOffset > uint32(stringOffset) {
				stringStart += entry.EntrySize
				continue
			}

			offset := uint32(stringOffset) - entry.RangeStartOffset
			if len(data.FooterData) < int(offset+stringStart) || offset > entry.EntrySize {
				stringStart += entry.EntrySize
				continue
			}

			messageStart, _, err = helpers.Take(data.FooterData, int(offset+stringStart))
			if err != nil {
				return messageData, err
			}
			messageFormatString, err := helpers.ExtractString(messageStart)
			if err != nil {
				return messageData, err
			}

			processString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}

			messageData.FormatString = messageFormatString
			messageData.Process = processString
			messageData.Library = processString

			return messageData, nil
		}
	}

	// There is a chance the log entry does not have a valid offset
	// Apple reports as "~~> <Invalid shared cache code pointer offset>" or <Invalid shared cache format string offset>
	if data, exists := provider.CachedUUIDText(mainUUID); exists {
		processString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
		if err != nil {
			return messageData, err
		}
		messageData.Process = processString
		messageData.Library = processString
		messageData.FormatString = fmt.Sprintf("Error: Invalid offset %d for UUID %s", stringOffset, mainUUID)
		return messageData, nil
	}

	// logger.Warn("Failed to get message string from UUIDText file")
	messageData.FormatString = fmt.Sprintf("Failed to get message string from UUIDText file: %s", mainUUID)
	return messageData, nil
}

// ExtractAbsoluteStrings extracts the message data for an absolute UUID firehose entry
func ExtractAbsoluteStrings(
	provider *uuidtext.CacheProvider,
	absoluteOffset uint64,
	stringOffset uint64,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *models.CatalogChunk,
	originalOffset uint64,
) (models.MessageData, error) {
	var err error

	key := fmt.Sprintf("%d_%d", firstProcID, secondProcID)
	uuid := ""

	if entry, exists := catalogs.ProcessInfoEntries[key]; exists {
		// In addition to firstProcID and secondProcID, we need to go through UUID entries in the catalog
		// Entries with the Absolute flag have the UUID stored in a slice of UUIDs and offsets/load_address
		// The correct UUID entry is the one where the absoluteOffset value falls in between loadAddress and loadAddress + size
		for _, uuidEntry := range entry.UUIDInfoEntries {
			if absoluteOffset >= uuidEntry.LoadAddress &&
				absoluteOffset <= (uuidEntry.LoadAddress+uint64(uuidEntry.Size)) {
				uuid = uuidEntry.UUID
				break
			}
		}
	}

	_, mainUUID := GetCatalogDSC(catalogs, firstProcID, secondProcID)
	messageData := models.MessageData{
		LibraryUUID: uuid,
		ProcessUUID: mainUUID,
	}

	// UUID data should already be loaded by the receiver and cached by the extension
	if originalOffset&0x80000000 != 0 {
		if data, exists := provider.CachedUUIDText(messageData.LibraryUUID); exists {
			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			messageData.Library = libraryString
			processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
			if err != nil {
				return messageData, err
			}
			messageData.Process = processString
			messageData.FormatString = "%s"

			return messageData, nil
		}
	}

	if data, exists := provider.CachedUUIDText(messageData.LibraryUUID); exists {
		stringStart := uint32(0)
		var messageStart []byte

		for _, entry := range data.EntryDescriptors {
			if entry.RangeStartOffset > uint32(stringOffset) {
				stringStart += entry.EntrySize
				continue
			}

			offset := stringOffset - uint64(entry.RangeStartOffset)
			if len(data.FooterData) < int(offset+uint64(stringStart)) || offset > uint64(entry.EntrySize) {
				stringStart += entry.EntrySize
				continue
			}

			if offset > uint64(^uint(0)>>1) {
				return messageData, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
			}

			messageStart, _, err = helpers.Take(data.FooterData, int(offset+uint64(stringStart)))
			if err != nil {
				return messageData, err
			}
			messageFormatString, err := helpers.ExtractString(messageStart)
			if err != nil {
				return messageData, err
			}

			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			messageData.FormatString = messageFormatString
			messageData.Library = libraryString

			processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
			if err != nil {
				return messageData, err
			}
			messageData.Process = processString
		}
	}

	// There is a chance the log entry does not have a valid offset
	// Apple labels as "error: ~~> Invalid bounds 4334340 for E502E11E-518F-38A7-9F0B-E129168338E7"
	if data, exists := provider.CachedUUIDText(messageData.LibraryUUID); exists {
		libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
		if err != nil {
			return messageData, err
		}
		messageData.Library = libraryString
		messageData.FormatString = fmt.Sprintf("Error: Invalid offset %d for absolute UUID %s", stringOffset, messageData.LibraryUUID)
		processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
		if err != nil {
			return messageData, err
		}
		messageData.Process = processString
		return messageData, nil
	}

	// logger.Warn("Failed to get message string from UUIDText file")
	messageData.FormatString = fmt.Sprintf("Failed to get string message from absolute UUIDText file: %s", messageData.LibraryUUID)
	return messageData, nil
}

// ExtractAltUUIDStrings extracts the message data for an alternative UUID firehose entry
func ExtractAltUUIDStrings(
	provider *uuidtext.CacheProvider,
	stringOffset uint64,
	uuid string,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *models.CatalogChunk,
	originalOffset uint64,
) (models.MessageData, error) {
	var err error

	_, mainUUID := GetCatalogDSC(catalogs, firstProcID, secondProcID)
	messageData := models.MessageData{
		LibraryUUID: uuid,
		ProcessUUID: mainUUID,
	}

	// UUID data should already be loaded by the receiver and cached by the extension

	if originalOffset&0x80000000 != 0 {
		if data, exists := provider.CachedUUIDText(uuid); exists {
			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			messageData.Library = libraryString
			processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
			if err != nil {
				return messageData, err
			}
			messageData.Process = processString
			messageData.FormatString = "%s"
			return messageData, nil
		}
	}

	if data, exists := provider.CachedUUIDText(uuid); exists {
		stringStart := uint32(0)
		for _, entry := range data.EntryDescriptors {
			var messageStart []byte

			if entry.RangeStartOffset > uint32(stringOffset) {
				stringStart += entry.EntrySize
				continue
			}

			offset := uint32(stringOffset) - entry.RangeStartOffset
			if len(data.FooterData) < int(offset) || offset > entry.EntrySize {
				stringStart += entry.EntrySize
				continue
			}

			messageStart, _, err = helpers.Take(data.FooterData, int(offset+stringStart))
			if err != nil {
				return messageData, err
			}
			messageFormatString, err := helpers.ExtractString(messageStart)
			if err != nil {
				return messageData, err
			}
			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
			if err != nil {
				return messageData, err
			}
			messageData.Library = libraryString
			messageData.Process = processString
			messageData.FormatString = messageFormatString
			return messageData, nil
		}
	}

	// There is a chance the log entry does not have a valid offset
	// Apple labels as "error: ~~> Invalid bounds 4334340 for E502E11E-518F-38A7-9F0B-E129168338E7"
	if data, exists := provider.CachedUUIDText(uuid); exists {
		libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
		if err != nil {
			return messageData, err
		}
		messageData.Library = libraryString
		messageData.FormatString = fmt.Sprintf("Error: Invalid offset %d for UUID %s", stringOffset, uuid)
		processString, err := GetUUIDImagePath(messageData.ProcessUUID, provider)
		if err != nil {
			return messageData, err
		}
		messageData.Process = processString
		return messageData, nil
	}

	// logger.Warn("Failed to get message string from UUIDText file")
	messageData.FormatString = fmt.Sprintf("Failed to get message string from alternative UUIDText file: %s", uuid)
	return messageData, nil

}

func GetCatalogDSC(catalogs *models.CatalogChunk, firstProcID uint64, secondProcID uint32) (string, string) {
	key := fmt.Sprintf("%d_%d", firstProcID, secondProcID)
	if entry, exists := catalogs.ProcessInfoEntries[key]; exists {
		return entry.DSCUUID, entry.MainUUID
	}
	// TODO: log warning
	return "", ""
}

func GetUUIDImagePath(uuid string, provider *uuidtext.CacheProvider) (string, error) {
	// An UUID of all zeros is possible in the Catalog, if this happens there is no process path
	if uuid == "00000000000000000000000000000000" {
		// logger.Info("Got UUID of all zeros fom Catalog")
		return "", nil
	}

	if data, exists := provider.CachedUUIDText(uuid); exists {
		return uuidTextImagePath(data.FooterData, data.EntryDescriptors)
	}

	// logger.Warn("Failed to get path string from UUIDText file for entry: %s", uuid)
	return fmt.Sprintf("Failed to get path string from UUIDText file for entry: %s", uuid), nil
}

func uuidTextImagePath(data []byte, entries []uuidtext.Entry) (string, error) {
	// Add up all entry range offset sizes to get image library offset
	imageLibraryOffset := uint32(0)
	var libraryStart []byte
	var err error

	for _, entry := range entries {
		imageLibraryOffset += entry.EntrySize
	}

	libraryStart, _, err = helpers.Take(data, int(imageLibraryOffset))
	if err != nil {
		return "", err
	}
	return helpers.ExtractString(libraryStart)
}

// FormatFirehoseLogMessage formats a complete log message from a firehose entry
func FormatFirehoseLogMessage(formatString string, itemMessage []ItemInfo, messageRe *regexp.Regexp) (string, error) {
	var err error
	formatAndMessageVec := []FormatAndMessage{}
	if len(itemMessage) == 0 && len(formatString) == 0 {
		return "", nil
	}
	if len(formatString) == 0 {
		return itemMessage[0].MessageStrings, nil
	}
	regexResults := messageRe.FindAllString(formatString, -1)
	for i, formatter := range regexResults {
		if strings.HasPrefix(formatter, "% ") {
			continue
		}

		formatAndMessage := FormatAndMessage{
			Formatter: "",
			Message:   "",
		}

		if formatter == "%%" {
			formatAndMessage.Formatter = formatter
			formatAndMessage.Message = "%"
			formatAndMessageVec = append(formatAndMessageVec, formatAndMessage)
			continue
		}

		if i >= len(itemMessage) {
			formatAndMessage.Formatter = formatter
			formatAndMessage.Message = "<Missing message data>"
			formatAndMessageVec = append(formatAndMessageVec, formatAndMessage)
			continue
		}

		formattedLogMessage := itemMessage[i].MessageStrings
		formatterString := formatter

		if strings.HasPrefix(formatterString, "%{") && strings.HasSuffix(formatterString, "}") {
			formatAndMessage.Formatter = formatterString
			formatterString = formatterString[1:]
			formatAndMessage.Message = formatterString
			formatAndMessageVec = append(formatAndMessageVec, formatAndMessage)
			continue
		}

		precisionItems := []uint8{0x10, 0x12}
		if slices.Contains(precisionItems, itemMessage[i].ItemType) {
			i++
		}

		dynamicPrecisionValue := uint8(0x0)
		if itemMessage[i].ItemType == dynamicPrecisionValue && itemMessage[i].ItemSize == 0 && strings.Contains(formatterString, "%*") {
			i++
		}

		if i >= len(itemMessage) {
			formatAndMessage.Formatter = formatter
			formatAndMessage.Message = "<Missing message data>"
			formatAndMessageVec = append(formatAndMessageVec, formatAndMessage)
			continue
		}

		privateStrings := []uint8{0x1, 0x21, 0x31, 0x41}
		privateNumber := uint8(0x1)
		privateMessage := uint16(0x8000)
		var results string
		if strings.HasPrefix(formatterString, "%{") {
			if slices.Contains(privateStrings, itemMessage[i].ItemType) && itemMessage[i].ItemSize == 0 || (itemMessage[i].ItemType == privateNumber && itemMessage[i].ItemSize == privateMessage) {
				formattedLogMessage = "<private>"
			} else {
				results, err = parseTypeFormatter(formatterString, itemMessage, itemMessage[i].ItemType, i)
				if err != nil {
					return "", err
				}
				formattedLogMessage = results
			}
		} else {
			if slices.Contains(privateStrings, itemMessage[i].ItemType) && itemMessage[i].ItemSize == 0 || (itemMessage[i].ItemType == privateNumber && itemMessage[i].ItemSize == privateMessage) {
				formattedLogMessage = "<private>"
			} else {
				results, err = parseFormatter(formatterString, itemMessage, itemMessage[i].ItemType, i)
				if err != nil {
					return "", err
				}
				formattedLogMessage = results
			}
		}

		i++
		formatAndMessage.Formatter = formatter
		formatAndMessage.Message = formattedLogMessage
		formatAndMessageVec = append(formatAndMessageVec, formatAndMessage)
	}

	logMessageVec := []string{}
	for _, values := range formatAndMessageVec {
		// Split the message by the formatter and replace it with the actual value
		// We need to do this instead of simple replace because the replacement string may also contain a formatter
		parts := strings.SplitN(formatString, values.Formatter, 2)
		if len(parts) == 2 {
			logMessageVec = append(logMessageVec, parts[0])
			logMessageVec = append(logMessageVec, values.Message)
			formatString = parts[1]
		} else {
			// If we can't split by the formatter, log an error and continue
			// This shouldn't happen in normal cases
			logMessageVec = append(logMessageVec, values.Formatter)
		}
	}

	// Add any remaining message content
	logMessageVec = append(logMessageVec, formatString)
	return strings.Join(logMessageVec, ""), nil
}

func parseFormatter(formatterString string, messageValue []ItemInfo, itemType uint8, itemIndex int) (string, error) {
	return "", nil
}

func parseTypeFormatter(formatterString string, messageValue []ItemInfo, itemType uint8, itemIndex int) (string, error) {
	return "", nil
}
