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

//go:build linux

package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectorGroupFromServiceFile(t *testing.T) {
	testCases := []struct {
		name     string
		contents string
		expected string
		errSubst string
	}{
		{
			name:     "default group",
			contents: "[Service]\nUser=root\nGroup=bdot\nExecStart=sleep 1000\n",
			expected: "bdot",
		},
		{
			name:     "custom group",
			contents: "[Service]\nUser=root\nGroup=collector\nExecStart=sleep 1000\n",
			expected: "collector",
		},
		{
			name:     "trims whitespace",
			contents: "[Service]\nUser=root\n  Group=bdot  \nExecStart=sleep 1000\n",
			expected: "bdot",
		},
		{
			name:     "group absent",
			contents: "[Service]\nUser=root\nExecStart=sleep 1000\n",
			errSubst: "Group= not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "unit.service")
			require.NoError(t, os.WriteFile(p, []byte(tc.contents), 0600))

			group, err := collectorGroupFromServiceFile(p)
			if tc.errSubst != "" {
				require.ErrorContains(t, err, tc.errSubst)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, group)
		})
	}

	t.Run("file does not exist", func(t *testing.T) {
		_, err := collectorGroupFromServiceFile(filepath.Join(t.TempDir(), "does-not-exist.service"))
		require.ErrorContains(t, err, "read service file")
	})
}
