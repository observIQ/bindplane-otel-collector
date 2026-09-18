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
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestUnixChownTree exercises the real syscalls. Only root can hand a tree to
// another user, so an unprivileged run asserts the no-op path and that the
// chown fails without CAP_CHOWN instead.
func TestUnixChownTree(t *testing.T) {
	tree := newStorageTree(t)
	ops := newFS()

	if os.Geteuid() != 0 {
		st, err := chownTree(tree.root, uint32(os.Getuid()), uint32(os.Getgid()), ops)
		if err != nil || st.changed != 0 || st.unchanged != 4 {
			t.Fatalf("chown to current owner: %s %v", st, err)
		}
		if _, err := chownTree(tree.root, testUID, testGID, ops); err == nil {
			t.Fatal("expected an error when chowning without CAP_CHOWN")
		}
		return
	}

	st, err := chownTree(tree.root, testUID, testGID, ops)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	if st.changed != 4 {
		t.Fatalf("unexpected stats: %s", st)
	}
	for _, path := range []string{tree.root, tree.config, tree.sub, tree.db} {
		fi, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if sys := fi.Sys().(*syscall.Stat_t); sys.Uid != testUID || sys.Gid != testGID {
			t.Fatalf("%s owner: got %d:%d", path, sys.Uid, sys.Gid)
		}
	}
	// The symlink target must keep its owner.
	fi, err := os.Lstat(filepath.Join(tree.root, "link"))
	if err != nil {
		t.Fatal(err)
	}
	if sys := fi.Sys().(*syscall.Stat_t); sys.Uid != uint32(os.Geteuid()) {
		t.Fatalf("symlink owner changed: %d", sys.Uid)
	}
}

func TestDescribeProcess(t *testing.T) {
	// /proc is linux only; elsewhere the helper returns an empty string.
	if desc := describeProcess(); desc != "" {
		for _, key := range []string{"Uid=", "Gid=", "CapEff="} {
			if !contains(desc, key) {
				t.Fatalf("%q lacks %s", desc, key)
			}
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
