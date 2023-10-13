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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_rehydrationCheckpoint(t *testing.T) {
	checkpoint := newCheckpoint()

	time1 := time.Now()
	time2 := time1.Add(time.Hour)
	time3 := time1.Add(-time.Hour)

	blobOne, blobTwo := "one", "two"

	// Ensure returns true from a new check point
	require.True(t, checkpoint.ShouldParse(time1, blobOne))
	require.True(t, checkpoint.ShouldParse(time1, blobTwo))

	// Update checkpoint
	checkpoint.UpdateCheckpoint(time1, blobOne)

	// Validate state was updated correctly
	require.True(t, checkpoint.LastTs.Equal(time1))
	require.Contains(t, checkpoint.ParsedBlobs, blobOne)

	// Redo checks with updated checkpoint

	// Should pass as blob has not been seen
	require.True(t, checkpoint.ShouldParse(time1, blobTwo))

	// Should fail due to blob being seen
	require.False(t, checkpoint.ShouldParse(time1, blobOne))

	// Should fail due to time being before last
	require.False(t, checkpoint.ShouldParse(time3, blobTwo))

	// Should pass as time is after and blob is not seen
	require.True(t, checkpoint.ShouldParse(time2, blobTwo))

}
