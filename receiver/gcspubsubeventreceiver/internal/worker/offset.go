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

package worker

import (
	"encoding/json"

	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
)

// OffsetStorageKey is the key used to store offsets in the storage client made by this receiver type
const OffsetStorageKey = "_gcs_pub_event_offset"

// Offset is used to keep track of where in a GCS event stream the receiver has read
type Offset struct {
	// Offset is an int64 tracking which byte was last read
	Offset int64 `json:"offset"`
}

// Offset implements the StorageData interface
var _ storageclient.StorageData = &Offset{}

// NewOffset creates a new Offset with the given offset
func NewOffset(o int64) *Offset {
	return &Offset{
		Offset: o,
	}
}

// Marshal implements the StorageData interface
func (o *Offset) Marshal() ([]byte, error) {
	return json.Marshal(o)
}

// Unmarshal implements the StorageData interface
// If the data is empty, it returns nil
func (o *Offset) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, o)
}
