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

package macosunifiedloggingencodingextension

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTimesyncRecord(t *testing.T) {
	testData := []byte{
		84, 115, 32, 0, 0, 0, 0, 0, 165, 196, 104, 252, 1, 0, 0, 0, 216, 189, 100, 108, 116,
		158, 131, 22, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	record, err := ParseTimesyncRecord(testData)
	require.NoError(t, err)
	require.Equal(t, uint32(0x207354), record.Signature)
	require.Equal(t, uint32(0), record.UnknownFlags)
	require.Equal(t, uint64(8529691813), record.KernelTime)
	require.Equal(t, int64(1622314513655447000), record.WallTime)
	require.Equal(t, uint32(0), record.Timezone)
	require.Equal(t, uint32(0), record.DaylightSavings)
}

func TestParseTimesyncBoot(t *testing.T) {
	testData := []byte{
		176, 187, 48, 0, 0, 0, 0, 0, 132, 91, 13, 213, 1, 96, 69, 62, 172, 224, 56, 118, 12,
		123, 92, 29, 1, 0, 0, 0, 1, 0, 0, 0, 168, 167, 19, 176, 114, 158, 131, 22, 0, 0, 0, 0,
		0, 0, 0, 0,
	}
	boot, err := ParseTimesyncBoot(testData)
	require.NoError(t, err)
	require.Equal(t, uint16(0xbbb0), boot.Signature)
	require.Equal(t, uint16(48), boot.HeaderSize)
	require.Equal(t, uint32(0), boot.Unknown)
	require.Equal(t, "845B0DD50160453EACE038760C7B5C1D", boot.BootUUID)
	require.Equal(t, uint32(1), boot.TimebaseNumerator)
	require.Equal(t, uint32(1), boot.TimebaseDenominator)
	require.Equal(t, int64(1622314506201049000), boot.BootTime)
	require.Equal(t, uint32(0), boot.TimezoneOffsetMins)
	require.Equal(t, uint32(0), boot.DaylightSavings)
}
