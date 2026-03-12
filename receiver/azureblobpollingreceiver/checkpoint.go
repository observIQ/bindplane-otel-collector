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

package azureblobpollingreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver"

import (
	"encoding/json"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/blobconsume"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
)

// PollingCheckPoint extends the basic checkpoint to include lastPollTime for continuous polling
type PollingCheckPoint struct {
	blobconsume.CheckPoint // Embed the base checkpoint

	// LastPollTime is the timestamp of the last successful poll
	// This is used to determine the starting time for the next poll window
	LastPollTime time.Time `json:"last_poll_time"`
}

// PollingCheckPoint implements the StorageData interface
var _ storageclient.StorageData = &PollingCheckPoint{}

// NewPollingCheckpoint creates a new PollingCheckPoint
func NewPollingCheckpoint() *PollingCheckPoint {
	return &PollingCheckPoint{
		CheckPoint:   *blobconsume.NewCheckpoint(),
		LastPollTime: time.Time{},
	}
}

// UpdatePollTime updates the last poll time to the given timestamp
func (c *PollingCheckPoint) UpdatePollTime(pollTime time.Time) {
	c.LastPollTime = pollTime
}

// Marshal implements the StorageData interface
func (c *PollingCheckPoint) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Unmarshal implements the StorageData interface
// If the data is empty, it returns nil
func (c *PollingCheckPoint) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, c)
}
