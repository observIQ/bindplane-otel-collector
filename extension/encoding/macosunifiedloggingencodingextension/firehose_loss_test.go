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

package macosunifiedloggingencodingextension

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: consolidate this test with the firehose test file after merge
func TestParseFirehoseLoss(t *testing.T) {
	parsedFirehoseLoss, _ := ParseFirehoseLoss(firehoseLossTestData)
	require.Equal(t, uint64(707475528), parsedFirehoseLoss.StartTime)
	require.Equal(t, uint64(3144863719), parsedFirehoseLoss.EndTime)
	require.Equal(t, uint64(63), parsedFirehoseLoss.Count)
}

var firehoseLossTestData = []byte{72, 56, 43, 42, 0, 0, 0, 0, 231, 207, 114, 187, 0, 0, 0, 0, 63, 0, 0, 0, 0, 0, 0, 0}
