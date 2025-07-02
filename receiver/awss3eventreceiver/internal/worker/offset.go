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

const OffsetStorageKey = "aws_s3_event_offset"

type Offset struct {
	Offset int64 `json:"offset"`
}

var _ storageclient.StorageData = &Offset{}

func NewOffset(o int64) *Offset {
	return &Offset{
		Offset: o,
	}
}

func (o *Offset) Marshal() ([]byte, error) {
	return json.Marshal(o)
}

func (o *Offset) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, o)
}
