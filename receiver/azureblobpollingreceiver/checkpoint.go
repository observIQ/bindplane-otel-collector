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

	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
)

// PollingCheckPoint extends the basic checkpoint to include lastPollTime for continuous polling
type PollingCheckPoint struct {
	// LastPollTime is the timestamp of the last successful poll
	// This is used to determine the starting time for the next poll window
	LastPollTime time.Time `json:"last_poll_time"`

	// LastTs is the time created from the folder path of the last consumed entity
	LastTs time.Time `json:"last_ts"`

	// ParsedEntities is a lookup of all entities that were parsed in the LastTs path
	ParsedEntities map[string]struct{} `json:"parsed_entities"`
}

// PollingCheckPoint implements the StorageData interface
var _ storageclient.StorageData = &PollingCheckPoint{}

// NewPollingCheckpoint creates a new PollingCheckPoint
func NewPollingCheckpoint() *PollingCheckPoint {
	return &PollingCheckPoint{
		LastPollTime:   time.Time{},
		LastTs:         time.Time{},
		ParsedEntities: make(map[string]struct{}),
	}
}

// ShouldParse returns true if the entity should be parsed based on its time and name.
// A value of false will be returned for entities that have a time before the LastTs or whose
// name is already tracked as parsed.
func (c *PollingCheckPoint) ShouldParse(entityTime time.Time, entityName string) bool {
	if entityTime.Before(c.LastTs) {
		return false
	}

	_, ok := c.ParsedEntities[entityName]
	return !ok
}

// UpdateCheckpoint updates the checkpoint with the lastEntityName.
// If the newTs is after the LastTs it sets lastTs to the newTs and clears its mapping of ParsedEntities.
// The lastEntityName is tracked in the mapping of ParsedEntities
func (c *PollingCheckPoint) UpdateCheckpoint(newTs time.Time, lastEntityName string) {
	if newTs.After(c.LastTs) {
		c.LastTs = newTs
		c.ParsedEntities = make(map[string]struct{})
	}

	c.ParsedEntities[lastEntityName] = struct{}{}
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
