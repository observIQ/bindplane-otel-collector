// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package uuidtext

import (
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
)

// MessageData represents parsed message data
type MessageData struct {
	FormatString string
	Process      string
	Library      string
	LibraryUUID  string
	ProcessUUID  string
}

func ExtractSharedStrings(
	provider types.FileProvider,
	stringOffset uint32,
	firstProcID uint64,
	secondProcID uint32,
	catalogs types.CatalogChunk,
	originalOffset uint64,
) (types.MessageData, error) {
	messageData := MessageData{}

	dscUUID, mainUUID := getCatalogDSC(catalogs, firstProcID, secondProcID)
	// ensure our cache is up to date

}

func getCatalogDSC(catalogs types.CatalogChunk, firstProcID uint64, secondProcID uint32) (string, string) {
	// Look for the process entry that matches the proc IDs
	key := fmt.Sprintf("%d_%d", firstProcID, secondProcID)
	if entry, exists := catalogs.ProcessInfoMap[key]; exists {
		return entry.DSCUUID, entry.MainUUID
	}

	return "", ""
}
