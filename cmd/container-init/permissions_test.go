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

package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

const (
	testUID uint32 = 1000000000
	testGID uint32 = 1000000000
)

// fakeFS keeps xattrs and ownership in memory so the walk can run on any
// platform against a real directory tree.
type fakeFS struct {
	xattrs   map[string]map[string][]byte
	owners   map[string][2]uint32
	devices  map[string]uint64
	getErr   map[string]error
	setErr   map[string]error
	chownErr map[string]error
	chowns   []string
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		xattrs:   map[string]map[string][]byte{},
		owners:   map[string][2]uint32{},
		devices:  map[string]uint64{},
		getErr:   map[string]error{},
		setErr:   map[string]error{},
		chownErr: map[string]error{},
	}
}

func (f *fakeFS) Lgetxattr(path, name string) ([]byte, bool, error) {
	if err := f.getErr[path]; err != nil {
		return nil, false, err
	}
	v, ok := f.xattrs[path][name]
	return v, ok, nil
}

func (f *fakeFS) Lsetxattr(path, name string, value []byte) error {
	if err := f.setErr[path]; err != nil {
		return err
	}
	if f.xattrs[path] == nil {
		f.xattrs[path] = map[string][]byte{}
	}
	f.xattrs[path][name] = append([]byte(nil), value...)
	return nil
}

func (f *fakeFS) Lchown(path string, uid, gid uint32) error {
	if err := f.chownErr[path]; err != nil {
		return err
	}
	f.chowns = append(f.chowns, path)
	f.owners[path] = [2]uint32{uid, gid}
	return nil
}

func (f *fakeFS) Owner(path string, _ fs.FileInfo) (uint32, uint32, bool) {
	o := f.owners[path]
	return o[0], o[1], true
}

func (f *fakeFS) Device(path string, _ fs.FileInfo) (uint64, bool) {
	if dev, ok := f.devices[path]; ok {
		return dev, true
	}
	return 1, true
}

func (f *fakeFS) acl(t *testing.T, path, name string) acl {
	t.Helper()
	raw, ok := f.xattrs[path][name]
	if !ok {
		t.Fatalf("%s has no %s", path, name)
	}
	a, err := decodeACL(raw)
	if err != nil {
		t.Fatalf("decode %s of %s: %v", name, path, err)
	}
	return a
}

// logTree is root/{a.log, sub/b.log, link -> a.log}.
type logTree struct {
	root, a, sub, b, link string
}

func newLogTree(t *testing.T) logTree {
	t.Helper()
	root := filepath.Join(t.TempDir(), "pods")
	tree := logTree{
		root: root,
		a:    filepath.Join(root, "a.log"),
		sub:  filepath.Join(root, "sub"),
		b:    filepath.Join(root, "sub", "b.log"),
		link: filepath.Join(root, "link"),
	}
	if err := os.MkdirAll(tree.sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tree.a, []byte("a"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tree.b, []byte("b"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(tree.a, tree.link); err != nil {
		t.Fatal(err)
	}
	return tree
}

func requireGroupGrant(t *testing.T, a acl, gid uint32, perm uint16) {
	t.Helper()
	if i := a.index(tagGroup, gid); i < 0 || a[i].perm != perm {
		t.Fatalf("group %d entry: want perm %#o in %+v", gid, perm, a)
	}
	if i := a.index(tagMask, aclUndefinedID); i < 0 || a[i].perm&perm != perm {
		t.Fatalf("mask does not admit %#o: %+v", perm, a)
	}
	if err := a.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestParsePermissionsArgs(t *testing.T) {
	cases := map[string]struct {
		args    []string
		wantErr bool
	}{
		"valid":               {[]string{"-uid=1000000000", "-gid=1000000000", "-chown=/etc/otel/storage", "-grant-read=/var/log/pods", "-grant-read=/run/log/journal"}, false},
		"grant only":          {[]string{"-uid=1", "-gid=1", "-grant-read=/var/log/pods"}, false},
		"missing uid":         {[]string{"-gid=1", "-chown=/a"}, true},
		"uid zero":            {[]string{"-uid=0", "-gid=1", "-chown=/a"}, true},
		"gid undefined":       {[]string{"-uid=1", "-gid=4294967295", "-chown=/a"}, true},
		"no paths":            {[]string{"-uid=1", "-gid=1"}, true},
		"relative path":       {[]string{"-uid=1", "-gid=1", "-chown=etc/otel"}, true},
		"unclean path":        {[]string{"-uid=1", "-gid=1", "-chown=/etc/otel/"}, true},
		"root path":           {[]string{"-uid=1", "-gid=1", "-grant-read=/"}, true},
		"duplicate path":      {[]string{"-uid=1", "-gid=1", "-chown=/a", "-grant-read=/a"}, true},
		"positional argument": {[]string{"-uid=1", "-gid=1", "-chown=/a", "extra"}, true},
		"unknown flag":        {[]string{"-uid=1", "-gid=1", "-chown=/a", "-bogus"}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			opts, err := parsePermissionsArgs(tc.args, io.Discard)
			if tc.wantErr {
				var usage *usageError
				if !errors.As(err, &usage) {
					t.Fatalf("got %v, want usageError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if opts.uid == 0 || opts.gid == 0 {
				t.Fatalf("uid/gid not set: %+v", opts)
			}
		})
	}
}

func TestGrantReadTreeAppliesACLs(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()

	st, err := grantReadTree(tree.root, testGID, ops, false)
	if err != nil {
		t.Fatalf("grantReadTree: %v", err)
	}
	if st.dirs != 2 || st.files != 2 || st.changed != 4 || st.unchanged != 0 || st.skipped != 1 || st.failed != 0 {
		t.Fatalf("unexpected stats: %s", st)
	}

	for _, dir := range []string{tree.root, tree.sub} {
		requireGroupGrant(t, ops.acl(t, dir, aclXattrAccess), testGID, permRead|permExec)
		requireGroupGrant(t, ops.acl(t, dir, aclXattrDefault), testGID, permRead|permExec)
	}
	for _, file := range []string{tree.a, tree.b} {
		requireGroupGrant(t, ops.acl(t, file, aclXattrAccess), testGID, permRead)
		if _, ok := ops.xattrs[file][aclXattrDefault]; ok {
			t.Fatalf("%s: files must not receive a default acl", file)
		}
	}
	if _, ok := ops.xattrs[tree.link]; ok {
		t.Fatal("symlink was modified")
	}

	// The default ACL seeds from the directory's own base entries.
	def := ops.acl(t, tree.sub, aclXattrDefault)
	if i := def.index(tagUserObj, aclUndefinedID); i < 0 || def[i].perm != 7 {
		t.Fatalf("default acl owner entry: %+v", def)
	}

	// A second run finds everything in place and writes nothing.
	before := len(ops.xattrs[tree.a][aclXattrAccess])
	st, err = grantReadTree(tree.root, testGID, ops, false)
	if err != nil {
		t.Fatalf("second grantReadTree: %v", err)
	}
	if st.changed != 0 || st.unchanged != 4 {
		t.Fatalf("second run not idempotent: %s", st)
	}
	if len(ops.xattrs[tree.a][aclXattrAccess]) != before {
		t.Fatal("second run rewrote an acl")
	}
}

func TestGrantReadTreeDryRunWritesNothing(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()

	st, err := grantReadTree(tree.root, testGID, ops, true)
	if err != nil {
		t.Fatalf("grantReadTree: %v", err)
	}
	if st.changed != 4 {
		t.Fatalf("dry run should count pending changes: %s", st)
	}
	if len(ops.xattrs) != 0 {
		t.Fatalf("dry run wrote xattrs: %v", ops.xattrs)
	}
}

func TestGrantReadTreeMissingRoot(t *testing.T) {
	_, err := grantReadTree(filepath.Join(t.TempDir(), "missing"), testGID, newFakeFS(), false)
	if !isNotExist(err) {
		t.Fatalf("got %v, want not-exist error", err)
	}
}

func TestGrantReadTreeRejectsSymlinkRoot(t *testing.T) {
	tree := newLogTree(t)
	linkRoot := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(tree.root, linkRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := grantReadTree(linkRoot, testGID, newFakeFS(), false); err == nil {
		t.Fatal("expected error for symlink root")
	}
}

func TestGrantReadTreeUnsupportedFilesystem(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()
	ops.getErr[tree.root] = syscall.ENOTSUP

	_, err := grantReadTree(tree.root, testGID, ops, false)
	if !errors.Is(err, errACLUnsupported) {
		t.Fatalf("got %v, want errACLUnsupported", err)
	}
	if len(ops.xattrs) != 0 {
		t.Fatal("walk continued after unsupported root")
	}
}

func TestGrantReadTreeRootPermissionDeniedIsFatal(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()
	ops.setErr[tree.root] = syscall.EPERM

	_, err := grantReadTree(tree.root, testGID, ops, false)
	if err == nil || errors.Is(err, errACLUnsupported) || !errors.Is(err, syscall.EPERM) {
		t.Fatalf("got %v, want EPERM", err)
	}
}

func TestGrantReadTreeToleratesEntryErrors(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()
	ops.setErr[tree.b] = syscall.EPERM

	st, err := grantReadTree(tree.root, testGID, ops, false)
	if err != nil {
		t.Fatalf("entry errors must not fail the walk: %v", err)
	}
	if st.failed != 1 || st.changed != 3 {
		t.Fatalf("unexpected stats: %s", st)
	}
	requireGroupGrant(t, ops.acl(t, tree.a, aclXattrAccess), testGID, permRead)
}

func TestGrantReadTreeStaysOnFilesystem(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()
	ops.devices[tree.sub] = 2

	st, err := grantReadTree(tree.root, testGID, ops, false)
	if err != nil {
		t.Fatalf("grantReadTree: %v", err)
	}
	if _, ok := ops.xattrs[tree.sub]; ok {
		t.Fatal("directory on another filesystem was modified")
	}
	if _, ok := ops.xattrs[tree.b]; ok {
		t.Fatal("walk descended into another filesystem")
	}
	if st.skipped != 2 || st.dirs != 1 || st.files != 1 {
		t.Fatalf("unexpected stats: %s", st)
	}
}

func TestChownTree(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()

	st, err := chownTree(tree.root, testUID, testGID, ops, false)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	if st.changed != 4 || st.unchanged != 0 || st.skipped != 1 {
		t.Fatalf("unexpected stats: %s", st)
	}
	want := map[string]bool{tree.root: true, tree.a: true, tree.sub: true, tree.b: true}
	for _, p := range ops.chowns {
		if !want[p] {
			t.Fatalf("unexpected chown of %s", p)
		}
		delete(want, p)
	}
	if len(want) != 0 {
		t.Fatalf("entries not chowned: %v", want)
	}

	st, err = chownTree(tree.root, testUID, testGID, ops, false)
	if err != nil {
		t.Fatalf("second chownTree: %v", err)
	}
	if st.changed != 0 || st.unchanged != 4 {
		t.Fatalf("second run not idempotent: %s", st)
	}
}

func TestChownTreeCreatesMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "container", "storage")
	ops := newFakeFS()

	st, err := chownTree(root, testUID, testGID, ops, false)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	fi, err := os.Lstat(root)
	if err != nil || !fi.IsDir() {
		t.Fatalf("root was not created: %v", err)
	}
	if st.changed != 1 || len(ops.chowns) != 1 {
		t.Fatalf("root was not chowned: %s %v", st, ops.chowns)
	}
}

func TestChownTreeDryRun(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()

	st, err := chownTree(tree.root, testUID, testGID, ops, true)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	if st.changed != 4 || len(ops.chowns) != 0 {
		t.Fatalf("dry run applied changes: %s %v", st, ops.chowns)
	}

	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := chownTree(missing, testUID, testGID, ops, true); err != nil {
		t.Fatalf("dry run on missing root: %v", err)
	}
	if _, err := os.Lstat(missing); !isNotExist(err) {
		t.Fatal("dry run created the root")
	}
}

func TestChownTreeFailsOnEntryError(t *testing.T) {
	tree := newLogTree(t)
	ops := newFakeFS()
	ops.chownErr[tree.b] = syscall.EPERM

	st, err := chownTree(tree.root, testUID, testGID, ops, false)
	if err == nil {
		t.Fatal("expected error when an entry cannot be chowned")
	}
	if st.failed != 1 || st.changed != 3 {
		t.Fatalf("unexpected stats: %s", st)
	}
}

func TestRunPermissions(t *testing.T) {
	tree := newLogTree(t)
	storage := filepath.Join(t.TempDir(), "storage")
	if err := os.WriteFile(filepath.Join(t.TempDir(), "unused"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	ops := newFakeFS()

	err := runPermissions([]string{
		"-uid=1000000000", "-gid=1000000000",
		"-chown=" + storage,
		"-grant-read=" + tree.root,
		"-grant-read=" + filepath.Join(t.TempDir(), "absent"),
	}, ops)
	if err != nil {
		t.Fatalf("runPermissions: %v", err)
	}
	if ops.owners[storage] != [2]uint32{testUID, testGID} {
		t.Fatalf("storage not chowned: %v", ops.owners)
	}
	requireGroupGrant(t, ops.acl(t, tree.a, aclXattrAccess), testGID, permRead)

	var usage *usageError
	if err := runPermissions([]string{"-gid=1", "-chown=" + storage}, ops); !errors.As(err, &usage) {
		t.Fatalf("got %v, want usageError", err)
	}
}
