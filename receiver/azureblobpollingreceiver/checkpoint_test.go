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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPollingCheckpoint(t *testing.T) {
	t.Run("NewPollingCheckpoint", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		require.NotNil(t, cp)
		require.True(t, cp.LastPollTime.IsZero())
		require.True(t, cp.LastTs.IsZero())
		require.Empty(t, cp.ParsedEntities)
	})

	t.Run("UpdatePollTime", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC()
		cp.UpdatePollTime(now)
		require.Equal(t, now, cp.LastPollTime)
	})

	t.Run("ShouldParse", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC()

		// Should parse new entity
		require.True(t, cp.ShouldParse(now, "blob1"))

		// Update checkpoint with entity
		cp.UpdateCheckpoint(now, "blob1")

		// Should not parse same entity again
		require.False(t, cp.ShouldParse(now, "blob1"))

		// Should parse different entity at same time
		require.True(t, cp.ShouldParse(now, "blob2"))

		// Should not parse entity before LastTs
		past := now.Add(-1 * time.Hour)
		require.False(t, cp.ShouldParse(past, "blob3"))

		// Should parse entity after LastTs
		future := now.Add(1 * time.Hour)
		require.True(t, cp.ShouldParse(future, "blob4"))
	})

	t.Run("UpdateCheckpoint clears old entities", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC()

		// Add some entities at current time
		cp.UpdateCheckpoint(now, "blob1")
		cp.UpdateCheckpoint(now, "blob2")
		require.Len(t, cp.ParsedEntities, 2)

		// Update with newer time should clear old entities
		future := now.Add(1 * time.Hour)
		cp.UpdateCheckpoint(future, "blob3")
		require.Len(t, cp.ParsedEntities, 1)
		require.Contains(t, cp.ParsedEntities, "blob3")
		require.NotContains(t, cp.ParsedEntities, "blob1")
		require.NotContains(t, cp.ParsedEntities, "blob2")
		require.Equal(t, future, cp.LastTs)
	})

	t.Run("Marshal and Unmarshal", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC().Truncate(time.Second) // Truncate for JSON precision

		cp.UpdatePollTime(now)
		cp.UpdateCheckpoint(now, "blob1")
		cp.UpdateCheckpoint(now, "blob2")

		// Marshal
		data, err := cp.Marshal()
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal into new checkpoint
		cp2 := NewPollingCheckpoint()
		err = cp2.Unmarshal(data)
		require.NoError(t, err)

		// Verify data matches
		require.Equal(t, cp.LastPollTime.Unix(), cp2.LastPollTime.Unix())
		require.Equal(t, cp.LastTs.Unix(), cp2.LastTs.Unix())
		require.Equal(t, cp.ParsedEntities, cp2.ParsedEntities)
	})

	t.Run("Unmarshal empty data", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		err := cp.Unmarshal([]byte{})
		require.NoError(t, err)
	})
}
