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

//go:build aix || linux

package path

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"
)

// srcInstallDir resolves the AIX install directory: the collector home from the
// SRC environment, then the directory of the program the SRC subsystem runs,
// then defaultDir. lssrcOut is the output of lssrc -S for the subsystem.
func srcInstallDir(home string, lssrcOut []byte, defaultDir string) string {
	if home != "" {
		return filepath.Clean(home)
	}
	if program := srcProgramPath(lssrcOut); filepath.IsAbs(program) {
		return filepath.Dir(program)
	}
	return defaultDir
}

// srcProgramPath returns the path field from lssrc -S output, or "" when it has
// none. A colon inside a field shifts the columns, so a row that doesn't match
// the header's column count is ignored rather than guessed at.
func srcProgramPath(lssrcOut []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(lssrcOut))
	if !sc.Scan() {
		return ""
	}
	header := strings.Split(strings.TrimPrefix(sc.Text(), "#"), ":")
	pathCol := -1
	for i, name := range header {
		if name == "path" {
			pathCol = i
		}
	}
	if pathCol < 0 || !sc.Scan() {
		return ""
	}
	row := strings.Split(sc.Text(), ":")
	if len(row) != len(header) {
		return ""
	}
	return row[pathCol]
}
