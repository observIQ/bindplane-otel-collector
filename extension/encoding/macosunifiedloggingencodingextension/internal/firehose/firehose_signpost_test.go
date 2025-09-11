// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

func TestParseFirehoseSignpost(t *testing.T) {
	testData := []byte{225, 244, 2, 0, 1, 0, 238, 238, 178, 178, 181, 176, 238, 238, 176, 63, 27, 0, 0, 0}
	testFlags := uint16(33282)
	signpost, _ := ParseFirehoseSignpost(testData, testFlags)
	require.Equal(t, uint32(193761), signpost.UnknownPCID)
	require.Equal(t, uint32(0), signpost.UnknownActivityID)
	require.Equal(t, uint32(0), signpost.UnknownSentinel)
	require.Equal(t, uint16(1), signpost.Subsystem)
	require.Equal(t, uint64(17216892719917625070), signpost.SignpostID)
	require.Equal(t, uint32(1785776), signpost.SignpostName)
	require.Equal(t, uint8(0), signpost.TTLValue)
	require.Equal(t, uint32(0), signpost.DataRefValue)

	require.True(t, signpost.FirehoseFormatters.MainExe)
	require.False(t, signpost.FirehoseFormatters.SharedCache)
	require.Equal(t, uint16(0), signpost.FirehoseFormatters.HasLargeOffset)
	require.Equal(t, uint16(0), signpost.FirehoseFormatters.LargeSharedCache)
	require.False(t, signpost.FirehoseFormatters.Absolute)
	require.Equal(t, "", signpost.FirehoseFormatters.UUIDRelative)
	require.Equal(t, uint16(0), signpost.FirehoseFormatters.MainExeAltIndex)
}
