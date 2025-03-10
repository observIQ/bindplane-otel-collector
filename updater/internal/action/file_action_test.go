// Copyright  Bindplane, Inc.
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

package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewCopyFileAction(t *testing.T) {
	t.Run("out file does not exist", func(t *testing.T) {
		scratchDir := t.TempDir()
		testInstallDir := filepath.Join("testdata", "copyfileaction")
		backupDir := path.BackupDir(testInstallDir)
		outFile := filepath.Join(scratchDir, "test.txt")
		inFile := filepath.Join(testInstallDir, "latest", "test.txt")

		a, err := NewCopyFileAction(zaptest.NewLogger(t), inFile, outFile, backupDir)
		require.NoError(t, err)

		require.Equal(t, &CopyFileAction{
			FromPathRel: inFile,
			ToPath:      outFile,
			FileCreated: true,
			backupDir:   backupDir,
			logger:      a.logger,
		}, a)
	})

	t.Run("out file exists", func(t *testing.T) {
		scratchDir := t.TempDir()
		testInstallDir := filepath.Join("testdata", "copyfileaction")
		backupDir := path.BackupDir(testInstallDir)
		outFile := filepath.Join(scratchDir, "test.txt")
		inFile := filepath.Join(testInstallDir, "latest", "test.txt")

		f, err := os.Create(outFile)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		a, err := NewCopyFileAction(zaptest.NewLogger(t), inFile, outFile, backupDir)
		require.NoError(t, err)

		require.Equal(t, &CopyFileAction{
			FromPathRel: inFile,
			ToPath:      outFile,
			FileCreated: false,
			backupDir:   backupDir,
			logger:      a.logger,
		}, a)
	})
}

func TestCopyFileActionRollback(t *testing.T) {
	t.Run("deletes out file if it does not exist", func(t *testing.T) {
		scratchDir := t.TempDir()
		testInstallDir := filepath.Join("testdata", "copyfileaction")
		backupDir := path.BackupDir(testInstallDir)
		outFile := filepath.Join(scratchDir, "test.txt")
		inFile := filepath.Join(testInstallDir, "tmp", "latest", "test.txt")

		a, err := NewCopyFileAction(zaptest.NewLogger(t), inFile, outFile, backupDir)
		require.NoError(t, err)

		inBytes, err := os.ReadFile(inFile)
		require.NoError(t, err)

		err = os.WriteFile(outFile, inBytes, 0600)
		require.NoError(t, err)

		err = a.Rollback()
		require.NoError(t, err)

		require.NoFileExists(t, outFile)
	})

	t.Run("Rolls back out file when it exists", func(t *testing.T) {
		scratchDir := t.TempDir()
		testInstallDir := filepath.Join("testdata", "copyfileaction")
		backupDir := path.BackupDir(testInstallDir)
		outFile := filepath.Join(scratchDir, "test.txt")
		inFileRel := "test.txt"
		inFile := filepath.Join(testInstallDir, "tmp", "latest", inFileRel)
		originalFile := filepath.Join(testInstallDir, "tmp", "rollback", "test.txt")

		originalBytes, err := os.ReadFile(originalFile)
		require.NoError(t, err)

		err = os.WriteFile(outFile, originalBytes, 0600)
		require.NoError(t, err)

		a, err := NewCopyFileAction(zaptest.NewLogger(t), inFileRel, outFile, backupDir)
		require.NoError(t, err)

		// Overwrite original file with latest file
		inBytes, err := os.ReadFile(inFile)
		require.NoError(t, err)

		err = os.WriteFile(outFile, inBytes, 0600)
		require.NoError(t, err)

		err = a.Rollback()
		require.NoError(t, err)

		require.FileExists(t, outFile)

		rolledBackBytes, err := os.ReadFile(outFile)
		require.NoError(t, err)

		require.Equal(t, originalBytes, rolledBackBytes)
	})

	t.Run("Fails if backup file doesn't exist", func(t *testing.T) {
		scratchDir := t.TempDir()
		testInstallDir := filepath.Join("testdata", "copyfileaction")
		backupDir := path.BackupDir(testInstallDir)
		outFile := filepath.Join(scratchDir, "test.txt")
		inFile := filepath.Join(testInstallDir, "tmp", "latest", "not_in_backup.txt")
		originalFile := filepath.Join(testInstallDir, "tmp", "rollback", "test.txt")

		// The latest file exists in the directory already, but for some reason is not copied to backup
		originalBytes, err := os.ReadFile(originalFile)
		require.NoError(t, err)

		err = os.WriteFile(outFile, originalBytes, 0600)
		require.NoError(t, err)

		a, err := NewCopyFileAction(zaptest.NewLogger(t), inFile, outFile, backupDir)
		require.NoError(t, err)

		// Overwrite original file with latest file
		latestBytes, err := os.ReadFile(inFile)
		require.NoError(t, err)

		err = os.WriteFile(outFile, latestBytes, 0600)
		require.NoError(t, err)

		err = a.Rollback()
		require.ErrorContains(t, err, "failed to copy file")
		require.FileExists(t, outFile)

		finalBytes, err := os.ReadFile(outFile)
		require.NoError(t, err)
		require.Equal(t, latestBytes, finalBytes)
	})

}
