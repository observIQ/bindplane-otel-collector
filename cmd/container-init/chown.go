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
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

// maxReportedErrors caps the per-entry warnings so a large tree cannot flood
// the logs.
const maxReportedErrors = 10

// fsOps abstracts the ownership syscalls so the tree walk can be tested on
// any platform.
type fsOps interface {
	// Lchown changes the owner of path without following symlinks.
	Lchown(path string, uid, gid uint32) error
	// Owner returns the uid and gid recorded in fi.
	Owner(path string, fi fs.FileInfo) (uid, gid uint32, ok bool)
	// Device returns the filesystem device id recorded in fi.
	Device(path string, fi fs.FileInfo) (dev uint64, ok bool)
}

// treeStats summarizes one tree walk.
type treeStats struct {
	dirs, files, changed, unchanged, skipped, failed int
}

func (s treeStats) String() string {
	return fmt.Sprintf("dirs=%d files=%d changed=%d unchanged=%d skipped=%d errors=%d",
		s.dirs, s.files, s.changed, s.unchanged, s.skipped, s.failed)
}

func (s *treeStats) recordError(path string, err error) {
	s.failed++
	if s.failed <= maxReportedErrors {
		log.Printf("WARNING %s: %v", path, err)
	} else if s.failed == maxReportedErrors+1 {
		log.Printf("WARNING further errors are counted but not logged")
	}
}

// chownTree recursively changes the owner of root to uid:gid, creating root
// when it does not exist. Any failure is an error because the collector
// cannot start without write access to its storage.
func chownTree(root string, uid, gid uint32, ops fsOps) (treeStats, error) {
	var st treeStats
	if _, err := os.Lstat(root); isNotExist(err) {
		if err := os.MkdirAll(root, 0750); err != nil {
			return st, err
		}
		log.Printf("created directory %s", root)
	}

	err := walkTree(root, ops, &st, func(path string, fi fs.FileInfo) error {
		curUID, curGID, ok := ops.Owner(path, fi)
		if ok && curUID == uid && curGID == gid {
			st.unchanged++
			return nil
		}
		if err := ops.Lchown(path, uid, gid); err != nil {
			return err
		}
		st.changed++
		return nil
	})
	if err != nil {
		return st, err
	}
	if st.failed > 0 {
		return st, fmt.Errorf("%d entries could not be chowned", st.failed)
	}
	return st, nil
}

// walkTree visits root and every regular file and directory below it that
// lives on the same filesystem. Symlinks, special files and directories on
// other filesystems are skipped and counted, so a symlink planted in the tree
// by the unprivileged collector cannot redirect the caller onto a host path.
// Errors on root abort the walk; errors on other entries are counted and the
// walk continues.
func walkTree(root string, ops fsOps, st *treeStats, visit func(path string, fi fs.FileInfo) error) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}
	rootDev, haveDev := ops.Device(root, rootInfo)

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr
			}
			if isNotExist(walkErr) {
				st.skipped++
			} else {
				st.recordError(path, walkErr)
			}
			return nil
		}
		if !d.IsDir() && !d.Type().IsRegular() {
			st.skipped++
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			if isNotExist(err) {
				st.skipped++
			} else {
				st.recordError(path, err)
			}
			return nil
		}
		if d.IsDir() && path != root && haveDev {
			if dev, ok := ops.Device(path, fi); ok && dev != rootDev {
				st.skipped++
				return fs.SkipDir
			}
		}
		if d.IsDir() {
			st.dirs++
		} else {
			st.files++
		}
		if err := visit(path, fi); err != nil {
			if path == root {
				return err
			}
			if isNotExist(err) {
				st.skipped++
			} else {
				st.recordError(path, err)
			}
		}
		return nil
	})
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOENT)
}
