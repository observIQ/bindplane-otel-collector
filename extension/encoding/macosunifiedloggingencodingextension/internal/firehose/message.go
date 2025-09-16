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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
)

func ExtractSharedStrings(
	provider *uuidtext.CacheProvider,
	stringOffset uint64,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *types.CatalogChunk,
	originalOffset uint64,
) (types.MessageData, error) {
	messageData := types.MessageData{}
	// Get shared string file (DSC) associated with log entry from Catalog
	dscUUID, mainUUID := getCatalogDSC(catalogs, firstProcID, secondProcID)

	// Check if we have the DSC data cached
	if _, exists := provider.CachedDSC(dscUUID); !exists {
		provider.UpdateDSC(dscUUID, mainUUID)
	}

	// Check if we have the UUID text data cached
	if _, exists := provider.CachedUUIDText(mainUUID); !exists {
		provider.UpdateUUID(mainUUID, mainUUID)
	}

	if originalOffset&0x80000000 != 0 {
		if sharedString, exists := provider.CachedDSC(dscUUID); exists {
			if ranges, exists := sharedString.Ranges[0]; exists {
				messageData.FormatString = "%s"
				messageData.Library = sharedString.UUIDs[ranges.UnknownUUIDIndex].PathString
				messageData.LibraryUUID = sharedString.UUIDs[ranges.UnknownUUIDIndex].UUID
				messageData.ProcessUUID = mainUUID

				processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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
			if stringOffset >= r.RangeOffset && stringOffset < (r.RangeOffset+uint64(r.RangeSize)) {
				offset := stringOffset - r.RangeOffset

				if offset > uint64(^uint(0)>>1) { // Check if larger than max int
					return messageData, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
				}

				messageStart, _, _ := utils.Take(r.Strings, int(offset))
				messageData.FormatString, _ = utils.ExtractString(messageStart)

				messageData.Library = sharedString.UUIDs[r.UnknownUUIDIndex].PathString
				messageData.LibraryUUID = sharedString.UUIDs[r.UnknownUUIDIndex].UUID

				messageData.ProcessUUID = mainUUID

				processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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
		if ranges, exists := sharedString.Ranges[0]; exists {
			messageData.Library = sharedString.UUIDs[ranges.UnknownUUIDIndex].PathString
			messageData.LibraryUUID = sharedString.UUIDs[ranges.UnknownUUIDIndex].UUID
			messageData.FormatString = "Error: Invalid shared string offset"
			messageData.ProcessUUID = mainUUID

			processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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

func ExtractFormatStrings(
	provider *uuidtext.CacheProvider,
	stringOffset uint64,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *types.CatalogChunk,
	originalOffset uint64,
) (types.MessageData, error) {
	_, mainUUID := getCatalogDSC(catalogs, firstProcID, secondProcID)

	messageData := types.MessageData{
		LibraryUUID: mainUUID,
		ProcessUUID: mainUUID,
	}

	if _, exists := provider.CachedUUIDText(mainUUID); !exists {
		provider.UpdateUUID(mainUUID, mainUUID)
	}

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
			if entry.RangeStartOffset > uint32(stringOffset) {
				stringStart += entry.EntrySize
				continue
			}

			offset := uint32(stringOffset) - entry.RangeStartOffset
			if len(data.FooterData) < int(offset+stringStart) || offset > entry.EntrySize {
				stringStart += entry.EntrySize
				continue
			}

			messageStart, _, _ := utils.Take(data.FooterData, int(offset+stringStart))
			messageFormatString, err := utils.ExtractString(messageStart)
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

func ExtractAbsoluteStrings(
	provider *uuidtext.CacheProvider,
	absoluteOffset uint64,
	stringOffset uint64,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *types.CatalogChunk,
	originalOffset uint64,
) (types.MessageData, error) {
	key := fmt.Sprintf("%d_%d", firstProcID, secondProcID)
	uuid := ""

	if entry, exists := catalogs.ProcessInfoMap[key]; exists {
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

	_, mainUUID := getCatalogDSC(catalogs, firstProcID, secondProcID)
	messageData := types.MessageData{
		LibraryUUID: uuid,
		ProcessUUID: mainUUID,
	}

	if _, exists := provider.CachedUUIDText(messageData.ProcessUUID); !exists {
		provider.UpdateUUID(messageData.ProcessUUID, messageData.LibraryUUID)
	}
	if _, exists := provider.CachedUUIDText(messageData.LibraryUUID); !exists {
		provider.UpdateUUID(messageData.LibraryUUID, messageData.ProcessUUID)
	}
	if originalOffset&0x80000000 != 0 {
		if data, exists := provider.CachedUUIDText(messageData.LibraryUUID); exists {
			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			messageData.Library = libraryString
			processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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

			messageStart, _, _ := utils.Take(data.FooterData, int(offset+uint64(stringStart)))
			messageFormatString, err := utils.ExtractString(messageStart)
			if err != nil {
				return messageData, err
			}

			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			messageData.FormatString = messageFormatString
			messageData.Library = libraryString

			processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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
		messageData.FormatString = fmt.Sprintf("Error: Invalid offset %d for UUID %s", stringOffset, messageData.LibraryUUID)
		processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
		if err != nil {
			return messageData, err
		}
		messageData.Process = processString
		return messageData, nil
	}

	// logger.Warn("Failed to get message string from UUIDText file")
	messageData.FormatString = fmt.Sprintf("Failed to get message string from UUIDText file: %s", messageData.LibraryUUID)
	return messageData, nil
}

func ExtractAltUUIDStrings(
	provider *uuidtext.CacheProvider,
	stringOffset uint64,
	uuid string,
	firstProcID uint64,
	secondProcID uint32,
	catalogs *types.CatalogChunk,
	originalOffset uint64,
) (types.MessageData, error) {
	_, mainUUID := getCatalogDSC(catalogs, firstProcID, secondProcID)
	messageData := types.MessageData{
		LibraryUUID: uuid,
		ProcessUUID: mainUUID,
	}

	if _, exists := provider.CachedUUIDText(messageData.LibraryUUID); !exists {
		provider.UpdateUUID(messageData.LibraryUUID, messageData.ProcessUUID)
	}
	if _, exists := provider.CachedUUIDText(messageData.ProcessUUID); !exists {
		provider.UpdateUUID(messageData.ProcessUUID, messageData.LibraryUUID)
	}

	if originalOffset&0x80000000 != 0 {
		if data, exists := provider.CachedUUIDText(uuid); exists {
			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			messageData.Library = libraryString
			processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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
			if entry.RangeStartOffset > uint32(stringOffset) {
				stringStart += entry.EntrySize
				continue
			}

			offset := uint32(stringOffset) - entry.RangeStartOffset
			if len(data.FooterData) < int(offset) || offset > entry.EntrySize {
				stringStart += entry.EntrySize
				continue
			}

			messageStart, _, _ := utils.Take(data.FooterData, int(offset+stringStart))
			messageFormatString, err := utils.ExtractString(messageStart)
			if err != nil {
				return messageData, err
			}
			libraryString, err := uuidTextImagePath(data.FooterData, data.EntryDescriptors)
			if err != nil {
				return messageData, err
			}
			processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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
		processString, err := getUUIDImagePath(messageData.ProcessUUID, provider)
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

func getCatalogDSC(catalogs *types.CatalogChunk, firstProcID uint64, secondProcID uint32) (string, string) {
	return "", ""
}

func getUUIDImagePath(uuid string, provider *uuidtext.CacheProvider) (string, error) {
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

func uuidTextImagePath(data []byte, entries []uuidtext.UUIDTextEntry) (string, error) {
	// Add up all entry range offset sizes to get image library offset
	imageLibraryOffset := uint32(0)

	for _, entry := range entries {
		imageLibraryOffset += entry.EntrySize
	}

	libraryStart, _, _ := utils.Take(data, int(imageLibraryOffset))
	return utils.ExtractString(libraryStart)
}
