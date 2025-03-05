// Copyright  Bindplane, Inc..
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

package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceInstallDir(t *testing.T) {
	testCases := []struct {
		input      []byte
		installDir string
		output     []byte
	}{
		{
			input:      []byte("[INSTALLDIR]"),
			installDir: "some/install/directory",
			output:     []byte(filepath.Join("some", "install", "directory") + string(os.PathSeparator)),
		},
		{
			input:      []byte("no install dir"),
			installDir: "some/install/directory",
			output:     []byte("no install dir"),
		},
		{
			input:      []byte("[INSTALLDIR]observiq-otel-collector"),
			installDir: "some/install/directory",
			output:     []byte(filepath.Join("some", "install", "directory", "observiq-otel-collector")),
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			out := replaceInstallDir(tc.input, tc.installDir)
			require.Equal(t, tc.output, out)
		})
	}
}
