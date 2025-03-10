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

package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("Existing file is removed", func(t *testing.T) {
		// We don't use t.TempDir here, because we can't clean up the out directory on windows.
		// We also don't clean up the out directory; It's in the temporary directory and may be cleaned up manually at any time.
		tmpDir, err := os.MkdirTemp("", "test-logger-existing-file")
		require.NoError(t, err)
		// Remove previous log directory if it exists
		require.NoError(t, os.RemoveAll(filepath.Join(tmpDir, "log")))
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "log"), 0775))

		updaterLogPath, err := filepath.Abs(filepath.Join(tmpDir, "log", "updater.log"))
		require.NoError(t, err)

		initialBytes := []byte("Some existing bytes")
		require.NoError(t, os.WriteFile(updaterLogPath, initialBytes, 0660))

		logger, err := NewLogger(tmpDir)
		require.NoError(t, err)

		currentBytes, err := os.ReadFile(updaterLogPath)
		require.NoError(t, err)

		if bytes.HasPrefix(currentBytes, initialBytes) {
			t.Fatalf("The log file was not deleted (current bytes: '%s')", currentBytes)
		}

		logger.Info("This is a log message")
		require.NoError(t, logger.Sync())

		require.FileExists(t, updaterLogPath)
	})

	t.Run("Logger creates file if existing file does not exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-logger-no-existing-file")
		require.NoError(t, err)
		// Remove previous log directory if it exists
		require.NoError(t, os.RemoveAll(filepath.Join(tmpDir, "log")))
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "log"), 0775))

		updaterLogPath, err := filepath.Abs(filepath.Join(tmpDir, "log", "updater.log"))
		require.NoError(t, err)

		logger, err := NewLogger(tmpDir)
		require.NoError(t, err)

		logger.Info("This is a log message")
		require.NoError(t, logger.Sync())

		require.FileExists(t, updaterLogPath)
	})
}
