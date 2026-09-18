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

//go:build unix

package main

import (
	"bufio"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

// unixFS implements fsOps with the real syscalls.
type unixFS struct{}

func newFS() fsOps { return unixFS{} }

func (unixFS) Lchown(path string, uid, gid uint32) error {
	return os.Lchown(path, int(uid), int(gid))
}

func (unixFS) Owner(_ string, fi fs.FileInfo) (uint32, uint32, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return st.Uid, st.Gid, true
}

func (unixFS) Device(_ string, fi fs.FileInfo) (uint64, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Dev), true // #nosec G115 -- Dev is unsigned on every supported platform
}

// describeProcess reports the identity and effective capabilities of this
// process so init container logs show what the command was allowed to do. It
// returns an empty string where /proc is not available.
func describeProcess() string {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return ""
	}
	defer f.Close()

	var fields []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, key := range []string{"Uid:", "Gid:", "Groups:", "CapEff:"} {
			if strings.HasPrefix(line, key) {
				fields = append(fields, strings.TrimSuffix(key, ":")+"="+strings.Join(strings.Fields(line[len(key):]), ","))
			}
		}
	}
	return strings.Join(fields, " ")
}
