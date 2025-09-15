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

func ExtractSharedStrings(provider *uuidtext.CacheProvider, stringOffset uint64, firstProcID uint64, secondProcID uint32, catalogs *types.CatalogChunk, originalOffset uint64) (types.MessageData, error) {
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

				processString := getUUIDImagePath(messageData.ProcessUUID, provider)
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

				processString := getUUIDImagePath(messageData.ProcessUUID, provider)
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

			processString := getUUIDImagePath(messageData.ProcessUUID, provider)
			messageData.Process = processString
			return messageData, nil
		}
	}

	// logger.Warn("Failed to get message string from Shared Strings DSC file")
	messageData.FormatString = "Unknown shared string message"
	return messageData, nil
}

func getCatalogDSC(catalogs *types.CatalogChunk, firstProcID uint64, secondProcID uint32) (string, string) {
	return "", ""
}

func getUUIDImagePath(uuid string, provider *uuidtext.CacheProvider) string {
	return ""
}
