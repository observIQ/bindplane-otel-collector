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
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const (
	testUID uint32 = 1000000000
	testGID uint32 = 1000000000
)

// fakeFS keeps ownership in memory so the walk can run on any platform
// against a real directory tree.
type fakeFS struct {
	owners   map[string][2]uint32
	devices  map[string]uint64
	chownErr map[string]error
	chowns   []string
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		owners:   map[string][2]uint32{},
		devices:  map[string]uint64{},
		chownErr: map[string]error{},
	}
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

// storageTree is root/{config.yaml, storage/receiver.db, link -> config.yaml},
// the shape the collector leaves behind in its storage volume.
type storageTree struct {
	root, config, sub, db, link string
}

func newStorageTree(t *testing.T) storageTree {
	t.Helper()
	root := filepath.Join(t.TempDir(), "storage")
	tree := storageTree{
		root:   root,
		config: filepath.Join(root, "config.yaml"),
		sub:    filepath.Join(root, "storage"),
		db:     filepath.Join(root, "storage", "receiver.db"),
		link:   filepath.Join(root, "link"),
	}
	if err := os.MkdirAll(tree.sub, 0750); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{tree.config, tree.db} {
		if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(tree.config, tree.link); err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestChownTree(t *testing.T) {
	tree := newStorageTree(t)
	ops := newFakeFS()

	st, err := chownTree(tree.root, testUID, testGID, ops)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	if st.dirs != 2 || st.files != 2 || st.changed != 4 || st.skipped != 1 {
		t.Fatalf("unexpected stats: %s", st)
	}
	for _, path := range []string{tree.root, tree.config, tree.sub, tree.db} {
		if ops.owners[path] != [2]uint32{testUID, testGID} {
			t.Fatalf("%s owner: got %v", path, ops.owners[path])
		}
	}
	// The symlink is skipped rather than followed, so a link planted by the
	// collector cannot redirect the chown onto a host path.
	if slices.Contains(ops.chowns, tree.link) {
		t.Fatalf("symlink was chowned: %v", ops.chowns)
	}

	// Rerunning is a no-op, which is what every pod restart does.
	ops.chowns = nil
	st, err = chownTree(tree.root, testUID, testGID, ops)
	if err != nil {
		t.Fatalf("second chownTree: %v", err)
	}
	if st.changed != 0 || st.unchanged != 4 || len(ops.chowns) != 0 {
		t.Fatalf("second run was not a no-op: %s %v", st, ops.chowns)
	}
}

func TestChownTreeCreatesMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "storage")
	ops := newFakeFS()

	st, err := chownTree(root, testUID, testGID, ops)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	if st.dirs != 1 || st.changed != 1 {
		t.Fatalf("unexpected stats: %s", st)
	}
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		t.Fatalf("root was not created: %v", err)
	}
}

func TestChownTreeRejectsFileRoot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := chownTree(file, testUID, testGID, newFakeFS()); err == nil {
		t.Fatal("expected an error for a non-directory root")
	}
}

// TestChownTreeFailsOnEntryError covers the upgrade case where one file
// cannot be handed over: the collector would not be able to use its storage,
// so the init container must fail loudly instead of continuing.
func TestChownTreeFailsOnEntryError(t *testing.T) {
	tree := newStorageTree(t)
	ops := newFakeFS()
	ops.chownErr[tree.db] = errors.New("boom")

	st, err := chownTree(tree.root, testUID, testGID, ops)
	if err == nil {
		t.Fatal("expected an error")
	}
	if st.failed != 1 {
		t.Fatalf("unexpected stats: %s", st)
	}
}

func TestChownTreeStaysOnFilesystem(t *testing.T) {
	tree := newStorageTree(t)
	ops := newFakeFS()
	ops.devices[tree.sub] = 99

	st, err := chownTree(tree.root, testUID, testGID, ops)
	if err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	if slices.Contains(ops.chowns, tree.db) {
		t.Fatalf("descended into another filesystem: %v", ops.chowns)
	}
	if st.skipped != 2 {
		t.Fatalf("unexpected stats: %s", st)
	}
}
