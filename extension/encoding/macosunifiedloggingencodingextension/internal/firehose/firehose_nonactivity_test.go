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

func TestParseFirehoseNonActivity(t *testing.T) {
	testFlags := uint16(556)
	parsedFirehoseNonActivity, _, err := ParseFirehoseNonActivity(firehoseNonActivityTestData, testFlags)
	require.NoError(t, err)
	require.Equal(t, uint32(0), parsedFirehoseNonActivity.UnknownActivityID)
	require.Equal(t, uint32(0), parsedFirehoseNonActivity.UnknownSentinel)
	require.Equal(t, uint16(0), parsedFirehoseNonActivity.PrivateStringsOffset)
	require.Equal(t, uint16(0), parsedFirehoseNonActivity.PrivateStringsSize)
	require.Equal(t, uint32(0), parsedFirehoseNonActivity.UnknownMessageStringRef)
	require.Equal(t, uint16(0), parsedFirehoseNonActivity.FirehoseFormatters.MainExeAltIndex)
	require.Equal(t, "", parsedFirehoseNonActivity.FirehoseFormatters.UUIDRelative)
	require.False(t, parsedFirehoseNonActivity.FirehoseFormatters.MainExe)
	require.False(t, parsedFirehoseNonActivity.FirehoseFormatters.Absolute)
	require.Equal(t, uint16(41), parsedFirehoseNonActivity.SubsystemValue)
	require.Equal(t, uint8(0), parsedFirehoseNonActivity.TTLValue)
	require.Equal(t, uint32(0), parsedFirehoseNonActivity.DataRefValue)
	require.Equal(t, uint16(4), parsedFirehoseNonActivity.FirehoseFormatters.LargeSharedCache)
	require.Equal(t, uint16(2), parsedFirehoseNonActivity.FirehoseFormatters.HasLargeOffset)
	require.Equal(t, uint32(218936186), parsedFirehoseNonActivity.UnknownPCID)
}

var firehoseNonActivityTestData = []byte{
	122, 179, 12, 13, 2, 0, 4, 0, 41, 0, 34, 9, 32, 4, 0, 0, 1, 0, 32, 4, 1, 0, 1, 0, 32,
	4, 2, 0, 14, 0, 0, 8, 2, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 2, 0,
	0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 4, 1, 0, 0, 0, 0, 4, 1, 0, 0, 0, 0, 0, 100, 105,
	115, 112, 97, 116, 99, 104, 69, 118, 101, 110, 116, 0,
}
