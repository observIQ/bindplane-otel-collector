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
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestNewChownTarget(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		uid, gid uint
		want     chownTarget
		wantErr  bool
	}{
		{name: "disabled"},
		{
			name: "valid", path: "/etc/otel/storage/", uid: 1000000000, gid: 1000000000,
			want: chownTarget{path: "/etc/otel/storage", uid: 1000000000, gid: 1000000000},
		},
		{name: "ids without path", uid: 1, wantErr: true},
		{name: "relative path", path: "etc/otel", uid: 1, gid: 1, wantErr: true},
		{name: "root path", path: "/etc/..", uid: 1, gid: 1, wantErr: true},
		{name: "zero uid", path: "/etc/otel", gid: 1, wantErr: true},
		{name: "gid too large", path: "/etc/otel", uid: 1, gid: maxID, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newChownTarget(tt.path, tt.uid, tt.gid)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// storageTree is root/{config.yaml, storage/receiver.db, link -> outside},
// the shape the collector leaves behind in its storage volume plus a symlink
// pointing out of the tree. It returns root and the entries a chown must
// visit; nothing under outside is one of them.
func storageTree(t *testing.T) (root string, entries []string) {
	t.Helper()
	tmp := t.TempDir()
	root = filepath.Join(tmp, "storage")
	sub := filepath.Join(root, "storage")
	config := filepath.Join(root, "config.yaml")
	db := filepath.Join(sub, "receiver.db")
	link := filepath.Join(root, "link")
	outside := filepath.Join(tmp, "outside")
	for _, dir := range []string{sub, outside} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{config, db, filepath.Join(outside, "secret")} {
		if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	return root, []string{root, config, link, sub, db}
}

func TestChownTree(t *testing.T) {
	root, want := storageTree(t)
	var got []string
	lchown := func(path string, uid, gid int) error {
		if uid != 10005 || gid != 10005 {
			t.Errorf("%s: chown to %d:%d, want 10005:10005", path, uid, gid)
		}
		got = append(got, path)
		return nil
	}
	if err := chownTree(root, 10005, 10005, lchown); err != nil {
		t.Fatalf("chownTree: %v", err)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("chowned %v, want %v", got, want)
	}
}

func TestChownTreeStopsAtFirstError(t *testing.T) {
	root, _ := storageTree(t)
	fail := errors.New("boom")
	calls := 0
	err := chownTree(root, 10005, 10005, func(string, int, int) error {
		calls++
		return fail
	})
	if !errors.Is(err, fail) {
		t.Fatalf("err = %v, want %v", err, fail)
	}
	if calls != 1 {
		t.Fatalf("lchown called %d times, want 1", calls)
	}
}

func TestChownTreeMissingRoot(t *testing.T) {
	err := chownTree(filepath.Join(t.TempDir(), "missing"), 10005, 10005, func(path string, _, _ int) error {
		t.Fatalf("lchown %s called for a missing root", path)
		return nil
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want not exist", err)
	}
}
