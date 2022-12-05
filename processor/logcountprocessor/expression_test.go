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

package logcountprocessor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpressionMatch(t *testing.T) {
	var testCases = []struct {
		name        string
		expr        string
		env         map[string]any
		expected    bool
		expectedErr error
	}{
		{
			name:     "simple true",
			expr:     "true",
			env:      map[string]any{},
			expected: true,
		},
		{
			name:     "simple false",
			expr:     "false",
			env:      map[string]any{},
			expected: false,
		},
		{
			name:     "true with env",
			expr:     `foo == "bar"`,
			env:      map[string]any{"foo": "bar"},
			expected: true,
		},
		{
			name:     "false with env",
			expr:     `foo == "bar"`,
			env:      map[string]any{"foo": "baz"},
			expected: false,
		},
		{
			name:        "invalid expression",
			expr:        `foo`,
			env:         map[string]any{"foo": "bar"},
			expectedErr: errors.New("expression did not return a boolean"),
		},
		{
			name:        "invalid env",
			expr:        `foo + "bar"`,
			env:         map[string]any{"foo": 1},
			expectedErr: errors.New("invalid operation: int + string"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := NewExpression(tc.expr)
			require.NoError(t, err)

			matches, err := expr.Match(tc.env)
			if tc.expectedErr != nil {
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expected, matches)
		})
	}
}

func TestNewExpression(t *testing.T) {
	var testCases = []struct {
		name        string
		expr        string
		expectedErr error
	}{
		{
			name: "simple true",
			expr: "true",
		},
		{
			name: "simple false",
			expr: "false",
		},
		{
			name:        "invalid expression",
			expr:        "",
			expectedErr: errors.New("unexpected token EOF"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewExpression(tc.expr)
			if tc.expectedErr != nil {
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
