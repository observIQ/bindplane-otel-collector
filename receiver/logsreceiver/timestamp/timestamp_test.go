// Copyright  observIQ, Inc.
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

package timestamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestUnixTimestampToOtelTimestamp(t *testing.T) {
	// Test seconds (<= 100 billion)
	out := UnixTimestampToOtelTimestamp(1000)
	require.Equal(t, pdata.Timestamp(1_000_000_000_000), out)

	// Test milliseconds (> 100 billion)
	out = UnixTimestampToOtelTimestamp(1_000_000_000_000)
	require.Equal(t, pdata.Timestamp(1_000_000_000_000_000_000), out)
}

func TestCoerceValToTimestamp(t *testing.T) {
	testDate := time.Date(2021, 6, 16, 13, 32, 0, 0, time.UTC)
	testDateTs := pdata.NewTimestampFromTime(testDate)
	origLocal := time.Local

	testCases := []struct {
		name       string
		val        pdata.AttributeValue
		ts         pdata.Timestamp
		beforeTest func()
		afterTest  func()
		ok         bool
	}{
		{
			name: "Unix seconds",
			val:  pdata.NewAttributeValueInt(1000),
			ts:   pdata.Timestamp(1_000_000_000_000),
			ok:   true,
		},
		{
			name: "Unix ms",
			val:  pdata.NewAttributeValueInt(1_000_000_000_000),
			ts:   pdata.Timestamp(1_000_000_000_000_000_000),
			ok:   true,
		},
		{
			name: "ISO8601 timestamp",
			val:  pdata.NewAttributeValueString(testDate.Format(iso8601TimestampLayout)),
			ts:   testDateTs,
			ok:   true,
		},
		{
			name: "RFC3339 timestamp",
			val:  pdata.NewAttributeValueString(testDate.Format(time.RFC3339)),
			ts:   testDateTs,
			ok:   true,
		},
		{
			name: "RFC3339 Nano timestamp",
			val:  pdata.NewAttributeValueString(testDate.Format(time.RFC3339Nano)),
			ts:   testDateTs,
			ok:   true,
		},
		{
			name: "That one format that is ambiguous",
			val:  pdata.NewAttributeValueString(testDate.Format("2006-01-02 15:04:05.999 MST")),
			beforeTest: func() {
				estContainingTz, err := time.LoadLocation("America/New_York")
				require.NoError(t, err)
				time.Local = estContainingTz
			},
			afterTest: func() {
				time.Local = origLocal
			},
			ts: testDateTs,
			ok: true,
		}, {
			name: "That one format that is ambiguous (ambiguous timezone)",
			val:  pdata.NewAttributeValueString("2021-02-05 14:41:56.21 PST"),
			beforeTest: func() {
				estContainingTz, err := time.LoadLocation("America/New_York")
				require.NoError(t, err)
				time.Local = estContainingTz
			},
			afterTest: func() {
				time.Local = origLocal
			},
			ok: false,
		},
		{
			name: "That one format that is ambiguous (ambiguous timezone, valid for location)",
			val:  pdata.NewAttributeValueString("2021-02-05 14:41:56.21 EST"),
			ts:   pdata.Timestamp(1612554116210000000),
			beforeTest: func() {
				estContainingTz, err := time.LoadLocation("America/New_York")
				require.NoError(t, err)
				time.Local = estContainingTz
			},
			afterTest: func() {
				time.Local = origLocal
			},
			ok: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.beforeTest != nil {
				testCase.beforeTest()
			}

			out, ok := CoerceValToTimestamp(testCase.val)

			if testCase.ok {
				require.True(t, ok)
				require.Equal(t, testCase.ts, out)
			} else {
				require.False(t, ok)
			}

			if testCase.afterTest != nil {
				testCase.afterTest()
			}
		})
	}
}
