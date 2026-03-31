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

package ocsfstandardizationprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoerceType(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		typeName string
		expected any
	}{
		{name: "integer from int", value: 42, typeName: "integer", expected: 42},
		{name: "integer from string", value: "42", typeName: "integer", expected: 42},
		{name: "integer from float64", value: 42.9, typeName: "integer", expected: 42},
		{name: "integer from int64", value: int64(42), typeName: "integer", expected: int(42)},
		{name: "integer from bool true", value: true, typeName: "integer", expected: 1},
		{name: "integer from bool false", value: false, typeName: "integer", expected: 0},
		{name: "integer from invalid string", value: "abc", typeName: "integer", expected: "abc"},

		{name: "long from int64", value: int64(1000), typeName: "long", expected: int64(1000)},
		{name: "long from int", value: 1000, typeName: "long", expected: int64(1000)},
		{name: "long from float64", value: 1000.7, typeName: "long", expected: int64(1000)},
		{name: "long from string", value: "1000", typeName: "long", expected: int64(1000)},
		{name: "long from bool true", value: true, typeName: "long", expected: int64(1)},
		{name: "long from invalid string", value: "abc", typeName: "long", expected: "abc"},

		{name: "float from float64", value: 3.14, typeName: "float", expected: 3.14},
		{name: "float from int", value: 3, typeName: "float", expected: float64(3)},
		{name: "float from int64", value: int64(3), typeName: "float", expected: float64(3)},
		{name: "float from string", value: "3.14", typeName: "float", expected: 3.14},
		{name: "float from invalid string", value: "abc", typeName: "float", expected: "abc"},

		{name: "boolean from bool", value: true, typeName: "boolean", expected: true},
		{name: "boolean from string true", value: "true", typeName: "boolean", expected: true},
		{name: "boolean from string false", value: "false", typeName: "boolean", expected: false},
		{name: "boolean from int 1", value: 1, typeName: "boolean", expected: true},
		{name: "boolean from int 0", value: 0, typeName: "boolean", expected: false},
		{name: "boolean from int64 1", value: int64(1), typeName: "boolean", expected: true},
		{name: "boolean from float64 0", value: float64(0), typeName: "boolean", expected: false},
		{name: "boolean from invalid string", value: "abc", typeName: "boolean", expected: "abc"},

		{name: "unknown type returns value as-is", value: "hello", typeName: "string", expected: "hello"},
		{name: "unknown type returns int as-is", value: 42, typeName: "unknown", expected: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coerceType(tt.value, tt.typeName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCoerceToTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{name: "int64 passthrough", value: int64(1618524549901), expected: int64(1618524549901)},
		{name: "int to int64", value: 1618524549, expected: int64(1618524549)},
		{name: "float64 to int64", value: float64(1618524549901), expected: int64(1618524549901)},
		{name: "numeric string", value: "1618524549901", expected: int64(1618524549901)},
		{name: "RFC3339 string", value: "2021-04-16T01:09:09Z", expected: int64(1618535349000)},
		{name: "RFC3339 with offset", value: "2021-04-16T01:09:09+00:00", expected: int64(1618535349000)},
		{name: "RFC3339Nano string", value: "2021-04-16T01:09:09.123456789Z", expected: int64(1618535349123)},
		{name: "invalid string unchanged", value: "not-a-timestamp", expected: "not-a-timestamp"},
		{name: "unsupported type unchanged", value: []string{"a"}, expected: []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coerceToTimestamp(tt.value)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCoerceToDatetime(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{name: "string passthrough", value: "2021-04-16T01:09:09Z", expected: "2021-04-16T01:09:09Z"},
		{name: "int64 to RFC3339", value: int64(1618535349000), expected: "2021-04-16T01:09:09Z"},
		{name: "int to RFC3339", value: 1618535349000, expected: "2021-04-16T01:09:09Z"},
		{name: "float64 to RFC3339", value: float64(1618535349000), expected: "2021-04-16T01:09:09Z"},
		{name: "unsupported type unchanged", value: true, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coerceToDatetime(tt.value)
			require.Equal(t, tt.expected, result)
		})
	}
}
