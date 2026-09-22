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
	"path/filepath"
)

// maxID is the reserved id the kernel reads as "leave unchanged", so it is
// the exclusive upper bound for -uid and -gid.
const maxID uint = 0xFFFFFFFF

// chownTarget is the owner the -chown tree is handed to. The zero value means
// no chown was requested.
type chownTarget struct {
	path     string
	uid, gid int
}

// newChownTarget validates the -chown, -uid and -gid flags.
func newChownTarget(path string, uid, gid uint) (chownTarget, error) {
	if path == "" {
		if uid != 0 || gid != 0 {
			return chownTarget{}, errors.New("-uid and -gid require -chown")
		}
		return chownTarget{}, nil
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || path == "/" {
		return chownTarget{}, fmt.Errorf("-chown path %q must be absolute and not /", path)
	}
	if uid == 0 || uid >= maxID {
		return chownTarget{}, fmt.Errorf("-uid must be between 1 and %d", maxID-1)
	}
	// gid 0 is allowed: OpenShift runs containers as an arbitrary uid whose
	// primary group is root, so its volumes are handed to uid:0.
	if gid >= maxID {
		return chownTarget{}, fmt.Errorf("-gid must be between 0 and %d", maxID-1)
	}
	return chownTarget{
		path: path,
		uid:  int(uid), // #nosec G115 -- range checked above
		gid:  int(gid), // #nosec G115 -- range checked above
	}, nil
}

// chownTree changes the owner of root and everything below it to uid:gid,
// like chown -Rh: symlinks are reowned but never followed, so a link planted
// in the tree by the unprivileged collector cannot redirect the caller onto a
// host path. It stops at the first failure, since the collector cannot start
// without ownership of its storage.
func chownTree(root string, uid, gid int, lchown func(path string, uid, gid int) error) error {
	return filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return lchown(path, uid, gid)
	})
}
