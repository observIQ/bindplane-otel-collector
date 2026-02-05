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

// Package blobconsume provides a checkpoint for blob-consuming receivers that can be stored using an internal/storageclient.StorageClient
package blobconsume //import "github.com/observiq/bindplane-otel-collector/internal/blobconsume"

import (
	"encoding/json"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
)

// CheckPoint is the checkpoint used with a storage extension to
// keep track of what's been parsed.
type CheckPoint struct {
	// LastTs is the time created from the folder path of the last consumed entity
	LastTs time.Time `json:"last_ts"`

	// ParsedEntities is a lookup of all entities that were parsed in the LastTs path
	ParsedEntities map[string]struct{} `json:"parsed_entities"`
}

// CheckPoint implements the StorageData interface
var _ storageclient.StorageData = &CheckPoint{}

// NewCheckpoint creates a new CheckPoint
func NewCheckpoint() *CheckPoint {
	return &CheckPoint{
		LastTs:         time.Time{},
		ParsedEntities: make(map[string]struct{}),
	}
}

// ShouldParse returns true if the entity should be parsed based on it's time and name.
// A value of false will be returned for entities that have a time before the LastTs or who's
// name is already tracked as parsed.
func (c *CheckPoint) ShouldParse(entityTime time.Time, entityName string) bool {
	if entityTime.Before(c.LastTs) {
		return false
	}

	_, ok := c.ParsedEntities[entityName]
	return !ok
}

// UpdateCheckpoint updates the checkpoint with the lastEntityName.
// If the newTs is after the LastTs it sets lastTs to the newTs and clears it's mapping of ParsedEntities.
// The lastEntityName is tracked in the mapping of ParsedEntities
func (c *CheckPoint) UpdateCheckpoint(newTs time.Time, lastEntityName string) {
	if newTs.After(c.LastTs) {
		c.LastTs = newTs
		c.ParsedEntities = make(map[string]struct{})
	}

	c.ParsedEntities[lastEntityName] = struct{}{}
}

// Marshal implements the StorageData interface
func (c *CheckPoint) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Unmarshal implements the StorageData interface
// If the data is empty, it returns nil
func (c *CheckPoint) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, c)
}
