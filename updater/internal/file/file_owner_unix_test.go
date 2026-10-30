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

//go:build !windows

package file

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type chownCall struct {
	path     string
	uid, gid int
}

// recordLchown swaps lchown for a recorder that returns err.
func recordLchown(t *testing.T, err error) *[]chownCall {
	t.Helper()
	calls := &[]chownCall{}
	orig := lchown
	lchown = func(path string, uid, gid int) error {
		*calls = append(*calls, chownCall{path, uid, gid})
		return err
	}
	t.Cleanup(func() { lchown = orig })
	return calls
}

func owner(t *testing.T, path string) (int, int) {
	t.Helper()
	fi, err := os.Lstat(path)
	require.NoError(t, err)
	st := fi.Sys().(*syscall.Stat_t)
	return int(st.Uid), int(st.Gid)
}

func TestCopyFileKeepsOwner(t *testing.T) {
	inFile := filepath.Join("testdata", "test.txt")

	t.Run("overwrite chowns to the replaced file's owner", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, os.WriteFile(outFile, []byte("old"), 0600))
		uid, gid := owner(t, outFile)
		calls := recordLchown(t, nil)

		require.NoError(t, CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile))
		require.Equal(t, []chownCall{{outFile, uid, gid}}, *calls)
	})

	t.Run("overwrite of a new file doesn't chown", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "test.txt")
		calls := recordLchown(t, nil)

		require.NoError(t, CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile))
		require.Empty(t, *calls)
	})

	t.Run("backup and rollback chown to the input file's owner", func(t *testing.T) {
		uid, gid := owner(t, inFile)
		calls := recordLchown(t, nil)

		backup := filepath.Join(t.TempDir(), "backup.txt")
		require.NoError(t, CopyFileNoOverwrite(zaptest.NewLogger(t), inFile, backup))
		restored := filepath.Join(t.TempDir(), "restored.txt")
		require.NoError(t, CopyFileRollback(zaptest.NewLogger(t), inFile, restored))
		require.Equal(t, []chownCall{{backup, uid, gid}, {restored, uid, gid}}, *calls)
	})

	t.Run("a chown the updater isn't allowed to make doesn't fail the copy", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, os.WriteFile(outFile, []byte("old"), 0600))
		recordLchown(t, syscall.EPERM)

		require.NoError(t, CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile))
		got, err := os.ReadFile(outFile)
		require.NoError(t, err)
		want, err := os.ReadFile(inFile)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("overwrite keeps a supplementary group", func(t *testing.T) {
		groups, err := os.Getgroups()
		require.NoError(t, err)
		gid := -1
		for _, g := range groups {
			if g != os.Getegid() {
				gid = g
				break
			}
		}
		if gid == -1 {
			t.Skip("needs a supplementary group")
		}
		outFile := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, os.WriteFile(outFile, []byte("old"), 0600))
		require.NoError(t, os.Lchown(outFile, -1, gid))

		require.NoError(t, CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile))
		_, got := owner(t, outFile)
		require.Equal(t, gid, got)
	})

	t.Run("overwrite as root keeps another user's owner", func(t *testing.T) {
		if os.Geteuid() != 0 {
			t.Skip("needs root")
		}
		outFile := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, os.WriteFile(outFile, []byte("old"), 0600))
		require.NoError(t, os.Lchown(outFile, 12345, 12345))

		require.NoError(t, CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile))
		uid, gid := owner(t, outFile)
		require.Equal(t, []int{12345, 12345}, []int{uid, gid})
	})
}
