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
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
)

func ExtractSharedStrings(provider *uuidtext.CacheProvider, stringOffset uint64, firstProcID uint64, secondProcID uint32, catalogs *types.CatalogChunk) (types.MessageData, error) {
	messageData := types.MessageData{}
	dscUUID, mainUUID := getCatalogDSC(catalogs, firstProcID, secondProcID)

	if provider.CachedDSC(dscUUID) == nil {
		provider.UpdateDSC(dscUUID, mainUUID)
	}
	if provider.CachedUUIDText(mainUUID) == nil {
		provider.UpdateUUID(mainUUID, mainUUID)
	}

	return messageData, nil
}

func getCatalogDSC(catalogs *types.CatalogChunk, firstProcID uint64, secondProcID uint32) (string, string) {
	key := fmt.Sprintf("%d_%d", firstProcID, secondProcID)
	if entry, exists := catalogs.ProcessInfoMap[key]; exists {
		return entry.DSCUUID, entry.MainUUID
	}
	return "", ""
}
