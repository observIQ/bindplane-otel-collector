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

package opamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineContentType(t *testing.T) {
	testCases := []struct {
		desc     string
		filePath string
		expected string
	}{
		{
			desc:     "json file",
			filePath: "path/to/my.json",
			expected: JSONContentType,
		},
		{
			desc:     "YAML file yml extension",
			filePath: "path/to/my.yml",
			expected: YAMLContentType,
		},
		{
			desc:     "YAML file yaml extension",
			filePath: "path/to/my.yaml",
			expected: YAMLContentType,
		},
		{
			desc:     "Unknown",
			filePath: "path/to/my.unicorn",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := DetermineContentType(tc.filePath)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
