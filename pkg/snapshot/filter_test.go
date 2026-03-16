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

package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestQueryMatchesValue(t *testing.T) {
	testCases := []struct {
		name           string
		valFunc        func(t *testing.T) pcommon.Value
		query          string
		expectedOutput bool
	}{
		{
			name:           "Matches string value",
			valFunc:        func(_ *testing.T) pcommon.Value { return pcommon.NewValueStr("String") },
			query:          "String",
			expectedOutput: true,
		},
		{
			name:           "Does not match string value (Case sensitive)",
			valFunc:        func(_ *testing.T) pcommon.Value { return pcommon.NewValueStr("String") },
			query:          "string",
			expectedOutput: false,
		},
		{
			name:           "Match string value (substring)",
			valFunc:        func(_ *testing.T) pcommon.Value { return pcommon.NewValueStr("String") },
			query:          "rin",
			expectedOutput: true,
		},
		{
			name: "Value is map (matches)",
			valFunc: func(t *testing.T) pcommon.Value {
				m := pcommon.NewValueMap()
				err := m.Map().FromRaw(map[string]any{
					"Key":        "Value",
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "AnotherVal",
			expectedOutput: true,
		},
		{
			name: "Value is map (does not match)",
			valFunc: func(t *testing.T) pcommon.Value {
				m := pcommon.NewValueMap()
				err := m.Map().FromRaw(map[string]any{
					"Key":        "Value",
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "aaaaa",
			expectedOutput: false,
		},
		{
			name: "Value is slice (matches)",
			valFunc: func(t *testing.T) pcommon.Value {
				s := pcommon.NewValueSlice()
				err := s.Slice().FromRaw([]any{"Thing1", "Thing2", 34})
				require.NoError(t, err)
				return s
			},
			query:          "Th",
			expectedOutput: true,
		},
		{
			name: "Value is slice (does not match)",
			valFunc: func(t *testing.T) pcommon.Value {
				s := pcommon.NewValueSlice()
				err := s.Slice().FromRaw([]any{"Thing1", "Thing2", 34})
				require.NoError(t, err)
				return s
			},
			query:          "DNE",
			expectedOutput: false,
		},
		{
			name: "Value is empty",
			valFunc: func(_ *testing.T) pcommon.Value {
				return pcommon.NewValueEmpty()
			},
			query:          "",
			expectedOutput: false,
		},
		{
			name: "Value is int",
			valFunc: func(_ *testing.T) pcommon.Value {
				return pcommon.NewValueInt(1345)
			},
			query:          "134",
			expectedOutput: true,
		},
		{
			name: "Value is double",
			valFunc: func(_ *testing.T) pcommon.Value {
				return pcommon.NewValueDouble(1452.25)
			},
			query:          "452.25",
			expectedOutput: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := queryMatchesValue(tc.valFunc(t), tc.query)
			require.Equal(t, tc.expectedOutput, o)
		})
	}
}

func TestQueryMatchesMap(t *testing.T) {
	testCases := []struct {
		name           string
		mapFunc        func(t *testing.T) pcommon.Map
		query          string
		expectedOutput bool
	}{
		{
			name: "Matches key",
			mapFunc: func(t *testing.T) pcommon.Map {
				m := pcommon.NewMap()
				err := m.FromRaw(map[string]any{
					"Key":        "Value",
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "AnotherKey",
			expectedOutput: true,
		},
		{
			name: "Matches subset of key",
			mapFunc: func(t *testing.T) pcommon.Map {
				m := pcommon.NewMap()
				err := m.FromRaw(map[string]any{
					"Key":        "Value",
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "herK",
			expectedOutput: true,
		},
		{
			name: "Key is substring of query",
			mapFunc: func(t *testing.T) pcommon.Map {
				m := pcommon.NewMap()
				err := m.FromRaw(map[string]any{
					"Key":        "Value",
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "ThisIsAnotherKeyVeryLong",
			expectedOutput: false,
		},
		{
			name: "Matches string value",
			mapFunc: func(t *testing.T) pcommon.Map {
				m := pcommon.NewMap()
				err := m.FromRaw(map[string]any{
					"Key":        "Value",
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "Val",
			expectedOutput: true,
		},
		{
			name: "Matches int value",
			mapFunc: func(t *testing.T) pcommon.Map {
				m := pcommon.NewMap()
				err := m.FromRaw(map[string]any{
					"Key":        123,
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "123",
			expectedOutput: true,
		},
		{
			name: "Matches value in nested map",
			mapFunc: func(t *testing.T) pcommon.Map {
				m := pcommon.NewMap()
				err := m.FromRaw(map[string]any{
					"Key": map[string]any{
						"Nested": "FindMeIfYouCan",
					},
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "FindMeIfYouCan",
			expectedOutput: true,
		},
		{
			name: "Matches value in nested slice",
			mapFunc: func(t *testing.T) pcommon.Map {
				m := pcommon.NewMap()
				err := m.FromRaw(map[string]any{
					"Key":        []any{"FindMeIfYouCan"},
					"AnotherKey": "AnotherValue",
				})
				require.NoError(t, err)

				return m
			},
			query:          "FindMeIfYouCan",
			expectedOutput: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := queryMatchesMap(tc.mapFunc(t), tc.query)
			require.Equal(t, tc.expectedOutput, o)
		})
	}
}

func TestQueryMatchesSlice(t *testing.T) {
	testCases := []struct {
		name           string
		sliceFunc      func(t *testing.T) pcommon.Slice
		query          string
		expectedOutput bool
	}{
		{
			name: "Empty Slice",
			sliceFunc: func(t *testing.T) pcommon.Slice {
				s := pcommon.NewSlice()
				err := s.FromRaw([]any{})
				require.NoError(t, err)
				return s
			},
			query:          "",
			expectedOutput: false,
		},
		{
			name: "Matches element in slice",
			sliceFunc: func(t *testing.T) pcommon.Slice {
				s := pcommon.NewSlice()
				err := s.FromRaw([]any{"Elem1", "Elem2", "Elem3"})
				require.NoError(t, err)
				return s
			},
			query:          "Elem2",
			expectedOutput: true,
		},
		{
			name: "Matches element in nested slice",
			sliceFunc: func(t *testing.T) pcommon.Slice {
				s := pcommon.NewSlice()
				err := s.FromRaw([]any{"Elem1", []any{"Elem2"}, "Elem3"})
				require.NoError(t, err)
				return s
			},
			query:          "Elem2",
			expectedOutput: true,
		},
		{
			name: "Matches element in nested map",
			sliceFunc: func(t *testing.T) pcommon.Slice {
				s := pcommon.NewSlice()
				err := s.FromRaw([]any{"Elem1", map[string]any{
					"SomeKey": "Elem2",
				}, "Elem3"})
				require.NoError(t, err)
				return s
			},
			query:          "SomeKey",
			expectedOutput: true,
		},
		{
			name: "Does not match any element in slice",
			sliceFunc: func(t *testing.T) pcommon.Slice {
				s := pcommon.NewSlice()
				err := s.FromRaw([]any{"Elem1", "Elem2", "Elem3"})
				require.NoError(t, err)
				return s
			},
			query:          "Does Not Match",
			expectedOutput: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := queryMatchesSlice(tc.sliceFunc(t), tc.query)
			require.Equal(t, tc.expectedOutput, o)
		})
	}
}
