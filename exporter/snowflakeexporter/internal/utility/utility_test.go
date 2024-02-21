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

package utility

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func TestConvertAttributesToString(t *testing.T) {
	testCases := []struct {
		desc     string
		m        map[string]any
		expected string
	}{
		{
			desc: "simple",
			m: map[string]any{
				"k1": "v1",
				"k2": "v2",
			},
			expected: `{"k1":"v1","k2":"v2"}`,
		},
		{
			desc: "nested map",
			m: map[string]any{
				"k1": "v1",
				"k2": map[string]any{
					"k2a": "v2a",
					"k2b": "v2b",
				},
			},
			expected: `{"k1":"v1","k2":{"k2a":"v2a","k2b":"v2b"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			m := pcommon.NewMap()
			require.NoError(t, m.FromRaw(tc.m))
			result := ConvertAttributesToString(m, zap.NewNop())
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestTraceIDToHexOrEmptyString(t *testing.T) {
	testCases := []struct {
		desc     string
		id       pcommon.TraceID
		expected string
	}{
		{
			desc:     "simple",
			id:       pcommon.TraceID([16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}),
			expected: "00000000000000000000000000000001",
		},
		{
			desc:     "max",
			id:       pcommon.TraceID([16]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}),
			expected: "ffffffffffffffffffffffffffffffff",
		},
		{
			desc:     "empty",
			id:       pcommon.TraceID([16]byte{}),
			expected: "",
		},
		{
			desc:     "something unique",
			id:       pcommon.TraceID([16]byte{21, 3, 64, 221, 101, 39, 92, 168, 81, 131, 248, 12, 43, 199, 124, 211}),
			expected: "150340dd65275ca85183f80c2bc77cd3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := TraceIDToHexOrEmptyString(tc.id)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestSpanIdToHexOrEmptyString(t *testing.T) {
	testCases := []struct {
		desc     string
		id       pcommon.SpanID
		expected string
	}{
		{
			desc:     "simple",
			id:       pcommon.SpanID([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
			expected: "0000000000000001",
		},
		{
			desc:     "max",
			id:       pcommon.SpanID([8]byte{255, 255, 255, 255, 255, 255, 255, 255}),
			expected: "ffffffffffffffff",
		},
		{
			desc:     "empty",
			id:       pcommon.SpanID([8]byte{}),
			expected: "",
		},
		{
			desc:     "something unique",
			id:       pcommon.SpanID([8]byte{64, 39, 92, 112, 43, 199, 124, 211}),
			expected: "40275c702bc77cd3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := SpanIDToHexOrEmptyString(tc.id)
			require.Equal(t, tc.expected, result)
		})
	}
}
