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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver"

import (
	"time"
)

// rehydrationCheckpoint is the checkpoint used with a storage extension to
// keep track of what's been rehydrated.
type rehydrationCheckpoint struct {
	// LastTs is the time created from the folder path of the last consumed blob
	LastTs time.Time `json:"last_ts"`

	// ParsedBlobs is a lookup of all blobs that were parsed in the LastTs path
	ParsedBlobs map[string]struct{} `json:"parsed_blobs"`
}

// newCheckpoint creates a new rehydrationCheckpoint
func newCheckpoint() *rehydrationCheckpoint {
	return &rehydrationCheckpoint{
		LastTs:      time.Time{},
		ParsedBlobs: make(map[string]struct{}),
	}
}

// ShouldParse returns true if the blob should be parsed based on it's time and name.
// A value of false will be returned for blobs that have a time before the LastTs or who's
// name is already tracked as parsed.
func (c *rehydrationCheckpoint) ShouldParse(blobTime time.Time, blobName string) bool {
	if blobTime.Before(c.LastTs) {
		return false
	}

	_, ok := c.ParsedBlobs[blobName]
	return !ok
}

// UpdateCheckpoint updates the checkpoint with the lastBlobName.
// If the newTs is after the LastTs it sets lastTs to the newTs and clears it's mapping of ParsedBlobs.
// The lastBlobName is tracked in the mapping of ParsedBlobs
func (c *rehydrationCheckpoint) UpdateCheckpoint(newTs time.Time, lastBlobName string) {
	if newTs.After(c.LastTs) {
		c.LastTs = newTs
		c.ParsedBlobs = make(map[string]struct{})
	}

	c.ParsedBlobs[lastBlobName] = struct{}{}
}
