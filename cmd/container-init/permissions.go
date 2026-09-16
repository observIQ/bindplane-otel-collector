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
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// permissionsCommand prepares host volumes for a collector that runs as an
// unprivileged user. It is intended to run as root in an init container.
const permissionsCommand = "permissions"

// maxReportedErrors caps the per-entry warnings so a large tree cannot flood the logs.
const maxReportedErrors = 10

// errACLUnsupported means the filesystem holding a path has no POSIX ACL support.
var errACLUnsupported = errors.New("filesystem does not support posix acls")

// usageError marks invalid command line input. It exits with status 2 like
// the flag package does.
type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

// fsOps abstracts the Linux-only syscalls behind the permissions command so
// the tree walk can be tested on any platform.
type fsOps interface {
	// Lgetxattr returns the xattr value of path without following symlinks.
	// exists is false when the attribute is not set.
	Lgetxattr(path, name string) (value []byte, exists bool, err error)
	// Lsetxattr sets the xattr value of path without following symlinks.
	Lsetxattr(path, name string, value []byte) error
	// Lchown changes the owner of path without following symlinks.
	Lchown(path string, uid, gid uint32) error
	// Owner returns the uid and gid recorded in fi.
	Owner(path string, fi fs.FileInfo) (uid, gid uint32, ok bool)
	// Device returns the filesystem device id recorded in fi.
	Device(path string, fi fs.FileInfo) (dev uint64, ok bool)
}

type permissionsOptions struct {
	uid, gid       uint32
	chownPaths     []string
	grantReadPaths []string
	dryRun         bool
}

type repeatedFlag []string

func (r *repeatedFlag) String() string { return strings.Join(*r, ",") }

func (r *repeatedFlag) Set(v string) error {
	*r = append(*r, v)
	return nil
}

// parsePermissionsArgs parses and validates the permissions command flags.
// Validation failures are reported to output together with the usage text.
func parsePermissionsArgs(args []string, output io.Writer) (*permissionsOptions, error) {
	fset := flag.NewFlagSet(permissionsCommand, flag.ContinueOnError)
	fset.SetOutput(output)
	opts := &permissionsOptions{}
	uid := fset.Uint("uid", 0, "owner uid for -chown paths (required)")
	gid := fset.Uint("gid", 0, "group gid for -chown paths and -grant-read acl entries (required)")
	fset.Var((*repeatedFlag)(&opts.chownPaths), "chown", "absolute path to recursively chown to uid:gid, created if missing (repeatable)")
	fset.Var((*repeatedFlag)(&opts.grantReadPaths), "grant-read", "absolute path to recursively grant gid read access to using posix acls, skipped if missing (repeatable)")
	fset.BoolVar(&opts.dryRun, "dry-run", false, "log the changes without applying them")
	if err := fset.Parse(args); err != nil {
		return nil, &usageError{err}
	}

	err := validatePermissionsOptions(*uid, *gid, fset.Args(), opts)
	if err != nil {
		fmt.Fprintf(output, "%s: %v\n", permissionsCommand, err)
		fset.Usage()
		return nil, &usageError{err}
	}
	opts.uid = uint32(*uid) // #nosec G115 -- range checked by validatePermissionsOptions
	opts.gid = uint32(*gid) // #nosec G115 -- range checked by validatePermissionsOptions
	return opts, nil
}

func validatePermissionsOptions(uid, gid uint, extra []string, opts *permissionsOptions) error {
	if len(extra) > 0 {
		return fmt.Errorf("unexpected arguments: %v", extra)
	}
	for name, v := range map[string]uint{"-uid": uid, "-gid": gid} {
		if v == 0 || v >= uint(aclUndefinedID) {
			return fmt.Errorf("%s must be between 1 and %d", name, aclUndefinedID-1)
		}
	}
	if len(opts.chownPaths)+len(opts.grantReadPaths) == 0 {
		return errors.New("at least one -chown or -grant-read path is required")
	}
	seen := map[string]bool{}
	for _, p := range append(append([]string{}, opts.chownPaths...), opts.grantReadPaths...) {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || p == "/" {
			return fmt.Errorf("path %q must be an absolute, clean path other than /", p)
		}
		if seen[p] {
			return fmt.Errorf("path %q is listed more than once", p)
		}
		seen[p] = true
	}
	return nil
}

// runPermissions applies the permissions command: every -chown tree is
// recursively chowned to uid:gid and every -grant-read tree receives ACL
// entries giving gid read access, including default entries so files
// created later inherit them.
func runPermissions(args []string, ops fsOps) error {
	opts, err := parsePermissionsArgs(args, os.Stderr)
	if err != nil {
		return err
	}

	for _, p := range opts.chownPaths {
		st, err := chownTree(p, opts.uid, opts.gid, ops, opts.dryRun)
		log.Printf("chown %s to %d:%d: %s", p, opts.uid, opts.gid, st)
		if err != nil {
			return fmt.Errorf("chown %s: %w", p, err)
		}
	}

	for _, p := range opts.grantReadPaths {
		st, err := grantReadTree(p, opts.gid, ops, opts.dryRun)
		switch {
		case isNotExist(err):
			log.Printf("grant-read %s: skipped, path does not exist", p)
		case errors.Is(err, errACLUnsupported):
			log.Printf("grant-read %s: WARNING skipped, %v; gid %d may be unable to read files under it", p, err, opts.gid)
		case err != nil:
			return fmt.Errorf("grant-read %s: %w", p, err)
		default:
			log.Printf("grant-read %s for gid %d: %s", p, opts.gid, st)
		}
	}

	if opts.dryRun {
		log.Printf("dry run: no changes were applied")
	}
	return nil
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

// walkTree visits root and every regular file and directory below it that
// lives on the same filesystem. Symlinks, special files and directories on
// other filesystems are skipped and counted. Errors on root abort the walk;
// errors on other entries are counted and the walk continues.
func walkTree(root string, ops fsOps, st *treeStats, visit func(path string, fi fs.FileInfo, isDir bool) error) error {
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
		if err := visit(path, fi, d.IsDir()); err != nil {
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

// chownTree recursively changes the owner of root to uid:gid, creating root
// when it does not exist. Any failure is an error because the collector
// cannot start without write access to its storage.
func chownTree(root string, uid, gid uint32, ops fsOps, dryRun bool) (treeStats, error) {
	var st treeStats
	if _, err := os.Lstat(root); isNotExist(err) {
		if dryRun {
			log.Printf("dry run: would create directory %s", root)
			return st, nil
		}
		if err := os.MkdirAll(root, 0750); err != nil {
			return st, err
		}
		log.Printf("created directory %s", root)
	}

	err := walkTree(root, ops, &st, func(path string, fi fs.FileInfo, _ bool) error {
		curUID, curGID, ok := ops.Owner(path, fi)
		if ok && curUID == uid && curGID == gid {
			st.unchanged++
			return nil
		}
		if !dryRun {
			if err := ops.Lchown(path, uid, gid); err != nil {
				return err
			}
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

// grantReadTree recursively grants gid read access below root using POSIX
// ACLs: read on files, read and search on directories, plus a default entry
// on directories so entries created later inherit the grant. A missing root
// returns a not-exist error and a filesystem without ACL support returns
// errACLUnsupported; both are for the caller to decide on.
func grantReadTree(root string, gid uint32, ops fsOps, dryRun bool) (treeStats, error) {
	var st treeStats
	err := walkTree(root, ops, &st, func(path string, fi fs.FileInfo, isDir bool) error {
		changed, err := grantRead(path, fi, isDir, gid, ops, dryRun)
		if err != nil {
			if path == root && isNotSupported(err) {
				return errACLUnsupported
			}
			return err
		}
		if changed {
			st.changed++
		} else {
			st.unchanged++
		}
		return nil
	})
	return st, err
}

func grantRead(path string, fi fs.FileInfo, isDir bool, gid uint32, ops fsOps, dryRun bool) (bool, error) {
	want := permRead
	if isDir {
		want |= permExec
	}
	access, changed, err := grantedACL(path, aclXattrAccess, gid, want, aclFromMode(fi.Mode()), ops, dryRun)
	if err != nil {
		return false, err
	}
	if !isDir {
		return changed, nil
	}
	// Only directories may carry a default ACL; the kernel rejects it elsewhere.
	_, defaultChanged, err := grantedACL(path, aclXattrDefault, gid, permRead|permExec, access.base(), ops, dryRun)
	if err != nil {
		return false, err
	}
	return changed || defaultChanged, nil
}

// grantedACL reads the named ACL of path, falling back to seed when it is not
// set, adds the group grant and writes it back if that changes anything. It
// returns the ACL as it was before the grant.
func grantedACL(path, name string, gid uint32, perm uint16, seed acl, ops fsOps, dryRun bool) (acl, bool, error) {
	raw, exists, err := ops.Lgetxattr(path, name)
	if err != nil {
		return nil, false, fmt.Errorf("get %s: %w", name, err)
	}
	current := seed
	if exists {
		if current, err = decodeACL(raw); err != nil {
			return nil, false, fmt.Errorf("%s: %w", name, err)
		}
	}

	next := current.grantGroup(gid, perm)
	if exists && bytes.Equal(next.encode(), current.encode()) {
		return current, false, nil
	}
	if err := next.validate(); err != nil {
		return nil, false, fmt.Errorf("%s: %w", name, err)
	}
	if !dryRun {
		if err := ops.Lsetxattr(path, name, next.encode()); err != nil {
			return nil, false, fmt.Errorf("set %s: %w", name, err)
		}
	}
	return current, true, nil
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOENT)
}

func isNotSupported(err error) bool {
	return errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP)
}
