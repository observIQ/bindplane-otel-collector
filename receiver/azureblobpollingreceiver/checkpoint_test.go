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
	"fmt"
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

	t.Run("ShouldParse with entity at same time as LastTs", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC()

		// Update checkpoint with first entity
		cp.UpdateCheckpoint(now, "blob1")

		// Should not parse same entity at same time
		require.False(t, cp.ShouldParse(now, "blob1"))

		// Should parse different entity at same time
		require.True(t, cp.ShouldParse(now, "blob2"))
	})

	t.Run("ShouldParse with exactly LastTs time", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC()

		// Set LastTs
		cp.UpdateCheckpoint(now, "blob1")

		// Entity at exactly LastTs time should be processed if not already parsed
		require.True(t, cp.ShouldParse(now, "blob2"))
		require.False(t, cp.ShouldParse(now, "blob1"))
	})

	t.Run("UpdateCheckpoint preserves entities at same time", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC()

		// Add multiple entities at same time
		cp.UpdateCheckpoint(now, "blob1")
		cp.UpdateCheckpoint(now, "blob2")
		cp.UpdateCheckpoint(now, "blob3")

		// All entities should be tracked
		require.Len(t, cp.ParsedEntities, 3)
		require.Contains(t, cp.ParsedEntities, "blob1")
		require.Contains(t, cp.ParsedEntities, "blob2")
		require.Contains(t, cp.ParsedEntities, "blob3")
		require.Equal(t, now, cp.LastTs)
	})

	t.Run("Marshal and Unmarshal with empty ParsedEntities", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC().Truncate(time.Second)

		cp.UpdatePollTime(now)
		// Don't add any entities

		// Marshal
		data, err := cp.Marshal()
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal
		cp2 := NewPollingCheckpoint()
		err = cp2.Unmarshal(data)
		require.NoError(t, err)

		// Verify
		require.Equal(t, cp.LastPollTime.Unix(), cp2.LastPollTime.Unix())
		require.Empty(t, cp2.ParsedEntities)
	})

	t.Run("Marshal and Unmarshal with many entities", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC().Truncate(time.Second)

		cp.UpdatePollTime(now)
		// Add many entities
		for i := 0; i < 100; i++ {
			blob := fmt.Sprintf("blob%d", i)
			cp.UpdateCheckpoint(now, blob)
		}

		// Marshal
		data, err := cp.Marshal()
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal
		cp2 := NewPollingCheckpoint()
		err = cp2.Unmarshal(data)
		require.NoError(t, err)

		// Verify all entities are preserved
		require.Equal(t, cp.LastPollTime.Unix(), cp2.LastPollTime.Unix())
		require.Equal(t, cp.LastTs.Unix(), cp2.LastTs.Unix())
		require.Len(t, cp2.ParsedEntities, 100)
		require.Equal(t, cp.ParsedEntities, cp2.ParsedEntities)
	})

	t.Run("Unmarshal invalid JSON", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		err := cp.Unmarshal([]byte("invalid json"))
		require.Error(t, err)
	})

	t.Run("ShouldParse with zero LastTs", func(t *testing.T) {
		cp := NewPollingCheckpoint()
		now := time.Now().UTC()

		// With zero LastTs, any entity with non-zero time should be processed
		require.True(t, cp.ShouldParse(now, "blob1"))
	})
}
