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
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// aclTempDir returns a temp directory on a filesystem with POSIX ACL support
// or skips the test.
func aclTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	probe := filepath.Join(dir, "probe")
	if err := os.WriteFile(probe, nil, 0600); err != nil {
		t.Fatal(err)
	}
	err := unix.Lsetxattr(probe, aclXattrAccess, aclFromMode(0600).grantGroup(4242, permRead).encode(), 0)
	if isNotSupported(err) {
		t.Skipf("filesystem at %s has no posix acl support", dir)
	}
	if err != nil {
		t.Fatalf("acl probe: %v", err)
	}
	return dir
}

func TestLinuxGrantReadRoundTrip(t *testing.T) {
	root := filepath.Join(aclTempDir(t), "pods")
	sub := filepath.Join(root, "sub")
	f640 := filepath.Join(root, "f640.log")
	f600 := filepath.Join(sub, "f600.log")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f640, []byte("x"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f600, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	const gid uint32 = 4242
	ops := linuxFS{}

	st, err := grantReadTree(root, gid, ops, false)
	if err != nil {
		t.Fatalf("grantReadTree: %v", err)
	}
	if st.changed != 4 || st.failed != 0 {
		t.Fatalf("unexpected stats: %s", st)
	}

	readACL := func(path, name string) acl {
		t.Helper()
		raw, ok, err := ops.Lgetxattr(path, name)
		if err != nil || !ok {
			t.Fatalf("%s of %s: ok=%v err=%v", name, path, ok, err)
		}
		a, err := decodeACL(raw)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}

	requireGroupGrant(t, readACL(f640, aclXattrAccess), gid, permRead)
	requireGroupGrant(t, readACL(f600, aclXattrAccess), gid, permRead)
	requireGroupGrant(t, readACL(root, aclXattrAccess), gid, permRead|permExec)
	requireGroupGrant(t, readACL(sub, aclXattrDefault), gid, permRead|permExec)

	// The kernel mirrors the mask in the group mode bits: 0640 is unchanged
	// and the 0600 file now displays r for the group class.
	for path, want := range map[string]os.FileMode{f640: 0640, f600: 0640} {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != want {
			t.Fatalf("%s mode: got %o, want %o", path, fi.Mode().Perm(), want)
		}
	}

	// A file created afterwards inherits the grant from the default ACL.
	inherited := filepath.Join(sub, "later.log")
	f, err := os.OpenFile(inherited, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	// The named entry is copied from the default ACL as r-x while the mask is
	// derived from the create mode's group bits, so the effective permission
	// is read only.
	inheritedACL := readACL(inherited, aclXattrAccess)
	entry := inheritedACL.index(tagGroup, gid)
	mask := inheritedACL.index(tagMask, aclUndefinedID)
	if entry < 0 || mask < 0 {
		t.Fatalf("inherited acl lacks group or mask entry: %+v", inheritedACL)
	}
	if effective := inheritedACL[entry].perm & inheritedACL[mask].perm; effective != permRead {
		t.Fatalf("inherited effective perm: got %#o, want r-- (%+v)", effective, inheritedACL)
	}

	// Rerunning is a no-op.
	st, err = grantReadTree(root, gid, ops, false)
	if err != nil {
		t.Fatalf("second grantReadTree: %v", err)
	}
	if st.changed != 0 {
		t.Fatalf("second run changed entries: %s", st)
	}
}

func TestLinuxDefaultACLRejectedOnFiles(t *testing.T) {
	file := filepath.Join(aclTempDir(t), "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	err := linuxFS{}.Lsetxattr(file, aclXattrDefault, aclFromMode(0600).grantGroup(4242, permRead).encode())
	if !errors.Is(err, unix.EACCES) {
		t.Fatalf("got %v, want EACCES", err)
	}
}

func TestLinuxChownTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "storage")
	if err := os.MkdirAll(filepath.Join(root, "storage"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	ops := linuxFS{}

	if os.Geteuid() != 0 {
		st, err := chownTree(root, uint32(os.Getuid()), uint32(os.Getgid()), ops, false)
		if err != nil || st.unchanged != 3 || st.changed != 0 {
			t.Fatalf("chown to current owner: %s %v", st, err)
		}
		if _, err := chownTree(root, testUID, testGID, ops, false); err == nil {
			t.Fatal("expected EPERM when chowning without CAP_CHOWN")
		}
		return
	}

	st, err := chownTree(root, testUID, testGID, ops, false)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	if st.changed != 3 {
		t.Fatalf("unexpected stats: %s", st)
	}
	fi, err := os.Lstat(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if sys := fi.Sys().(*syscall.Stat_t); sys.Uid != testUID || sys.Gid != testGID {
		t.Fatalf("owner: got %d:%d", sys.Uid, sys.Gid)
	}
}

func TestDescribeProcess(t *testing.T) {
	desc := describeProcess()
	for _, key := range []string{"Uid=", "Gid=", "CapEff="} {
		if !contains(desc, key) {
			t.Fatalf("%q lacks %s", desc, key)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
