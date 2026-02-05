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

package blobconsume //import "github.com/observiq/bindplane-otel-collector/internal/blobconsume"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Checkpoint(t *testing.T) {
	checkpoint := NewCheckpoint()

	time1 := time.Now()
	time2 := time1.Add(time.Hour)
	time3 := time1.Add(-time.Hour)

	entityOne, entityTwo := "one", "two"

	// Ensure returns true from a new check point
	require.True(t, checkpoint.ShouldParse(time1, entityOne))
	require.True(t, checkpoint.ShouldParse(time1, entityTwo))

	// Update checkpoint
	checkpoint.UpdateCheckpoint(time1, entityOne)

	// Validate state was updated correctly
	require.True(t, checkpoint.LastTs.Equal(time1))
	require.Contains(t, checkpoint.ParsedEntities, entityOne)

	// Redo checks with updated checkpoint

	// Should pass as entity has not been seen
	require.True(t, checkpoint.ShouldParse(time1, entityTwo))

	// Should fail due to entity being seen
	require.False(t, checkpoint.ShouldParse(time1, entityOne))

	// Should fail due to time being before last
	require.False(t, checkpoint.ShouldParse(time3, entityTwo))

	// Should pass as time is after and entity is not seen
	require.True(t, checkpoint.ShouldParse(time2, entityTwo))

}
