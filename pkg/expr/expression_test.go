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
			expr, err := CreateExpression(tc.expr)
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

func TestExtractFloat(t *testing.T) {
	var testCases = []struct {
		name        string
		expr        string
		env         map[string]any
		expected    float64
		expectedErr error
	}{
		{
			name:     "simple float",
			expr:     "1.0",
			env:      map[string]any{},
			expected: 1.0,
		},
		{
			name:     "float with env",
			expr:     `foo + 1.0`,
			env:      map[string]any{"foo": 1.0},
			expected: 2.0,
		},
		{
			name:     "int in env",
			expr:     "foo",
			env:      map[string]any{"foo": 1},
			expected: 1.0,
		},
		{
			name:     "int32 in env",
			expr:     "foo",
			env:      map[string]any{"foo": int32(1)},
			expected: 1.0,
		},
		{
			name:     "int64 in env",
			expr:     "foo",
			env:      map[string]any{"foo": int64(1)},
			expected: 1.0,
		},
		{
			name:     "string conversion",
			expr:     "foo",
			env:      map[string]any{"foo": "1"},
			expected: 1.0,
		},
		{
			name:        "failed string conversion",
			expr:        `foo`,
			env:         map[string]any{"foo": "bar"},
			expectedErr: errors.New("failed to convert string to float"),
		},
		{
			name:        "invalid operation",
			expr:        `foo + "bar"`,
			env:         map[string]any{"foo": 1},
			expectedErr: errors.New("invalid operation: int + string"),
		},
		{
			name:        "invalid value type",
			expr:        `foo`,
			env:         map[string]any{"foo": true},
			expectedErr: errors.New("invalid value type: bool"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := CreateExpression(tc.expr)
			require.NoError(t, err)

			number, err := expr.ExtractFloat(tc.env)
			if tc.expectedErr != nil {
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expected, number)
		})
	}
}

func TestExtractInt(t *testing.T) {
	var testCases = []struct {
		name        string
		expr        string
		env         map[string]any
		expected    int64
		expectedErr error
	}{
		{
			name:     "simple int",
			expr:     "1",
			env:      map[string]any{},
			expected: 1,
		},
		{
			name:     "int with env",
			expr:     `foo + 1`,
			env:      map[string]any{"foo": 1},
			expected: 2,
		},
		{
			name:     "int32 in env",
			expr:     "foo",
			env:      map[string]any{"foo": int32(1)},
			expected: 1,
		},
		{
			name:     "int64 in env",
			expr:     "foo",
			env:      map[string]any{"foo": int64(1)},
			expected: 1,
		},
		{
			name:     "float conversion",
			expr:     "foo",
			env:      map[string]any{"foo": 1.0},
			expected: 1,
		},
		{
			name:     "string conversion",
			expr:     "foo",
			env:      map[string]any{"foo": "1"},
			expected: 1,
		},
		{
			name:        "failed string conversion",
			expr:        `foo`,
			env:         map[string]any{"foo": "bar"},
			expectedErr: errors.New("failed to convert string to int"),
		},
		{
			name:        "invalid operation",
			expr:        `foo + "bar"`,
			env:         map[string]any{"foo": 1},
			expectedErr: errors.New("invalid operation: int + string"),
		},
		{
			name:        "invalid value type",
			expr:        `foo`,
			env:         map[string]any{"foo": true},
			expectedErr: errors.New("invalid value type: bool"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := CreateExpression(tc.expr)
			require.NoError(t, err)

			number, err := expr.ExtractInt(tc.env)
			if tc.expectedErr != nil {
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expected, number)
		})
	}
}

func TestCreateExpression(t *testing.T) {
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
			_, err := CreateExpression(tc.expr)
			if tc.expectedErr != nil {
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMatchRecord(t *testing.T) {
	var testCases = []struct {
		name     string
		expr     string
		record   map[string]interface{}
		expected bool
	}{
		{
			name:     "simple true",
			expr:     "true",
			record:   map[string]interface{}{},
			expected: true,
		},
		{
			name:     "simple false",
			expr:     "false",
			record:   map[string]interface{}{},
			expected: false,
		},
		{
			name:     "true with record",
			expr:     `foo == "bar"`,
			record:   map[string]interface{}{"foo": "bar"},
			expected: true,
		},
		{
			name:     "false with record",
			expr:     `foo == "bar"`,
			record:   map[string]interface{}{"foo": "baz"},
			expected: false,
		},
		{
			name:     "invalid expression",
			expr:     `foo`,
			record:   map[string]interface{}{"foo": "bar"},
			expected: false,
		},
		{
			name:     "invalid record",
			expr:     `foo + "bar"`,
			record:   map[string]interface{}{"foo": 1},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := CreateExpression(tc.expr)
			require.NoError(t, err)

			matches := expr.MatchRecord(tc.record)
			require.Equal(t, tc.expected, matches)
		})
	}
}

func TestCreateBoolExpression(t *testing.T) {
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
			expr:        "1",
			expectedErr: errors.New("expected bool, but got int"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CreateBoolExpression(tc.expr)
			if tc.expectedErr != nil {
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
