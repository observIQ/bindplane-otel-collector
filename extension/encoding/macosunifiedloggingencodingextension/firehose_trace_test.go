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
package macosunifiedloggingencodingextension

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFirehoseTrace(t *testing.T) {
	testData := []byte{106, 139, 3, 0, 0}
	firehoseTrace, _, err := ParseFirehoseTrace(testData)
	require.NoError(t, err)
	require.Equal(t, uint32(232298), firehoseTrace.UnknownPCID)

	testData = []byte{248, 145, 3, 0, 200, 0, 0, 0, 0, 0, 0, 0, 8, 1}
	firehoseTrace, _, err = ParseFirehoseTrace(testData)
	require.NoError(t, err)
	require.Equal(t, uint32(233976), firehoseTrace.UnknownPCID)
	require.Equal(t, 1, len(firehoseTrace.MessageData.ItemInfo))
}

func TestGetMessage(t *testing.T) {
	testData := []byte{200, 0, 0, 0, 0, 0, 0, 0, 8, 1}
	slices.Reverse(testData)
	_, message, err := GetMessage(testData)
	require.NoError(t, err)
	require.Equal(t, "200", message.ItemInfo[0].MessageStrings)
}

func TestGetMessageMultiple(t *testing.T) {
	testData := []byte{2, 8, 8, 0, 0, 0, 0, 0, 0, 0, 200, 0, 0, 127, 251, 75, 225, 96, 176}
	_, message, err := GetMessage(testData)
	require.NoError(t, err)
	require.Equal(t, "140717286580400", message.ItemInfo[0].MessageStrings)
	require.Equal(t, "200", message.ItemInfo[1].MessageStrings)
}
