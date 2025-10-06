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

package firehose

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFirehoseActivity(t *testing.T) {
	data := []byte{
		178, 251, 0, 0, 0, 0, 0, 128, 236, 0, 0, 0, 0, 0, 0, 0, 178, 251, 0, 0, 0, 0, 0, 128,
		179, 251, 0, 0, 0, 0, 0, 128, 64, 63, 24, 18, 1, 0, 2, 0,
	}

	flags := uint16(573)
	logType := uint8(0x1)
	results, _, err := ParseFirehoseActivity(data, flags, logType)
	require.NoError(t, err)

	require.Equal(t, results.ActivityID, uint32(64434))
	require.Equal(t, results.Sentinel, uint32(2147483648))
	require.Equal(t, results.PID, uint64(236))
	require.Equal(t, results.ActivityID2, uint32(64434))
	require.Equal(t, results.Sentinel2, uint32(2147483648))
	require.Equal(t, results.ActivityID3, uint32(64435))
	require.Equal(t, results.Sentinel3, uint32(2147483648))
	require.Equal(t, results.MessageStringRef, uint32(0))
	require.False(t, results.FirehoseFormatters.MainExe)
	require.False(t, results.FirehoseFormatters.Absolute)
	require.False(t, results.FirehoseFormatters.SharedCache)
	require.Equal(t, results.FirehoseFormatters.MainExeAltIndex, uint16(0))
	require.Equal(t, results.FirehoseFormatters.UUIDRelative, "")
	require.Equal(t, results.PCID, uint32(303578944))
	require.Equal(t, results.FirehoseFormatters.HasLargeOffset, uint16(1))
	require.Equal(t, results.FirehoseFormatters.LargeSharedCache, uint16(2))
}
