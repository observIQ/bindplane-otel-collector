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

package expr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateExpressionMap(t *testing.T) {
	var testCases = []struct {
		name        string
		expressions map[string]string
		expectedErr error
	}{
		{
			name: "simple",
			expressions: map[string]string{
				"foo": "true",
			},
		},
		{
			name: "invalid expression",
			expressions: map[string]string{
				"foo": "....",
			},
			expectedErr: errors.New("failed to create expression for foo: unexpected token"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expressionMap, err := CreateExpressionMap(tc.expressions)
			if tc.expectedErr != nil {
				require.Contains(t, err.Error(), tc.expectedErr.Error())
				return
			}
			require.NoError(t, err)

			for key := range tc.expressions {
				require.NotNil(t, expressionMap.expressions[key])
				require.Equal(t, tc.expressions[key], expressionMap.expressions[key].Source.Content())
			}
		})
	}
}

func TestMapExtract(t *testing.T) {
	var testCases = []struct {
		name        string
		expressions map[string]string
		record      Record
		expected    map[string]any
	}{
		{
			name: "simple",
			expressions: map[string]string{
				"foo": "true",
			},
			record:   Record{},
			expected: map[string]any{"foo": true},
		},
		{
			name: "multiple",
			expressions: map[string]string{
				"foo": "true",
				"bar": "false",
			},
			record:   Record{},
			expected: map[string]any{"foo": true, "bar": false},
		},
		{
			name: "simple record",
			expressions: map[string]string{
				"expr1": BodyField,
			},
			record: Record{
				BodyField: "value1",
			},
			expected: map[string]any{"expr1": "value1"},
		},
		{
			name: "missing field",
			expressions: map[string]string{
				"expr1": BodyField,
				"expr2": AttributesField,
			},
			record: Record{
				BodyField: "value1",
			},
			expected: map[string]any{"expr1": "value1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expressionMap, err := CreateExpressionMap(tc.expressions)
			require.NoError(t, err)

			result := expressionMap.Extract(tc.record)
			require.Equal(t, tc.expected, result)
		})
	}
}
