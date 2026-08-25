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

package service

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRollStderrFile(t *testing.T) {
	t.Run("missing file is a no-op", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "observiq_collector.err")
		require.NoError(t, rollStderrFile(path))
		require.NoFileExists(t, path+".1")
	})

	t.Run("empty file is left in place", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "observiq_collector.err")
		require.NoError(t, os.WriteFile(path, nil, 0660))
		require.NoError(t, rollStderrFile(path))
		require.FileExists(t, path)
		require.NoFileExists(t, path+".1")
	})

	t.Run("non-empty file rolls to backup, replacing previous backup", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "observiq_collector.err")
		require.NoError(t, os.WriteFile(path, []byte("current"), 0660))
		require.NoError(t, os.WriteFile(path+".1", []byte("old backup"), 0660))

		require.NoError(t, rollStderrFile(path))

		require.NoFileExists(t, path)
		backup, err := os.ReadFile(path + ".1")
		require.NoError(t, err)
		require.Equal(t, "current", string(backup))
	})
}

func TestCapStderrFile(t *testing.T) {
	t.Run("under cap is untouched", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "observiq_collector.err")
		require.NoError(t, os.WriteFile(path, []byte("small"), 0660))

		require.NoError(t, capStderrFile(path, stderrMaxBytes))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "small", string(content))
		require.NoFileExists(t, path+".1")
	})

	t.Run("threshold 1 archives any non-empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "observiq_collector.err")
		require.NoError(t, os.WriteFile(path, []byte("last run"), 0660))

		require.NoError(t, capStderrFile(path, 1))

		backup, err := os.ReadFile(path + ".1")
		require.NoError(t, err)
		require.Equal(t, "last run", string(backup))
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Empty(t, content)
	})

	t.Run("over cap copies to backup and truncates, appends continue", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "observiq_collector.err")
		big := bytes.Repeat([]byte("x"), stderrMaxBytes)
		require.NoError(t, os.WriteFile(path, big, 0660))

		// Hold an O_APPEND handle across the cap, mirroring the stderr handle.
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0660)
		require.NoError(t, err)
		defer f.Close()

		require.NoError(t, capStderrFile(path, stderrMaxBytes))

		backup, err := os.ReadFile(path + ".1")
		require.NoError(t, err)
		require.Equal(t, big, backup)

		// The held handle must keep working and write at the new EOF.
		_, err = f.WriteString("after truncate")
		require.NoError(t, err)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "after truncate", string(content))
	})
}
