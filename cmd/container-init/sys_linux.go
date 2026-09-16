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

package main

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// linuxFS implements fsOps with the real syscalls.
type linuxFS struct{}

func newFS() (fsOps, error) { return linuxFS{}, nil }

func (linuxFS) Lgetxattr(path, name string) ([]byte, bool, error) {
	buf := make([]byte, 256)
	for {
		n, err := unix.Lgetxattr(path, name, buf)
		switch {
		case err == nil:
			return buf[:n], true, nil
		case errors.Is(err, unix.ENODATA):
			return nil, false, nil
		case errors.Is(err, unix.ERANGE):
			size, err := unix.Lgetxattr(path, name, nil)
			if err != nil {
				return nil, false, err
			}
			buf = make([]byte, size)
		default:
			return nil, false, err
		}
	}
}

func (linuxFS) Lsetxattr(path, name string, value []byte) error {
	return unix.Lsetxattr(path, name, value, 0)
}

func (linuxFS) Lchown(path string, uid, gid uint32) error {
	return unix.Lchown(path, int(uid), int(gid))
}

func (linuxFS) Owner(_ string, fi fs.FileInfo) (uint32, uint32, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return st.Uid, st.Gid, true
}

func (linuxFS) Device(_ string, fi fs.FileInfo) (uint64, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Dev), true // #nosec G115 -- Dev is unsigned on every linux architecture
}

// describeProcess reports the identity and effective capabilities of this
// process so init container logs show what the command was allowed to do.
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
		for _, key := range []string{"Uid:", "Gid:", "CapEff:"} {
			if strings.HasPrefix(line, key) {
				fields = append(fields, strings.TrimSuffix(key, ":")+"="+strings.Join(strings.Fields(line[len(key):]), ","))
			}
		}
	}
	return strings.Join(fields, " ")
}
