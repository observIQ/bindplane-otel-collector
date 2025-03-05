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

package rollback

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	action_mocks "github.com/observiq/bindplane-otel-collector/updater/internal/action/mocks"
	service_mocks "github.com/observiq/bindplane-otel-collector/updater/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestRollbackerBackup(t *testing.T) {
	t.Run("Successfully backs up everything", func(t *testing.T) {
		outDir := t.TempDir()
		installDir := filepath.Join("testdata", "rollbacker")

		svc := service_mocks.NewMockService(t)
		svc.On("Backup").Return(nil)

		rb := &filesystemRollbacker{
			originalSvc: svc,
			backupDir:   outDir,
			installDir:  installDir,
			logger:      zaptest.NewLogger(t),
		}

		installJarPath := filepath.Join(rb.installDir, "..", "opentelemetry-java-contrib-jmx-metrics.jar")
		_, err := os.Create(installJarPath)
		require.NoError(t, err)
		err = os.WriteFile(installJarPath, []byte("# The old jar file"), 0660)
		require.NoError(t, err)

		err = rb.Backup()
		require.NoError(t, err)

		require.FileExists(t, filepath.Join(outDir, "opentelemetry-java-contrib-jmx-metrics.jar"))
		require.FileExists(t, filepath.Join(outDir, "some-file.txt"))
		require.FileExists(t, filepath.Join(outDir, "plugins-dir", "plugin.txt"))
		require.NoDirExists(t, filepath.Join(outDir, "tmp-dir"))
	})

	t.Run("Service backup fails", func(t *testing.T) {
		outDir := t.TempDir()
		installDir := filepath.Join("testdata", "rollbacker")

		svc := service_mocks.NewMockService(t)
		svc.On("Backup").Return(fmt.Errorf("invalid permissions"))

		rb := &filesystemRollbacker{
			originalSvc: svc,
			backupDir:   outDir,
			installDir:  installDir,
			logger:      zaptest.NewLogger(t),
		}

		err := rb.Backup()
		require.ErrorContains(t, err, "failed to backup service configuration")
	})

	t.Run("Removes pre-existing backup", func(t *testing.T) {
		outDir := t.TempDir()
		installDir := filepath.Join("testdata", "rollbacker")
		leftoverFile := filepath.Join(outDir, "leftover-file.txt")

		svc := service_mocks.NewMockService(t)
		svc.On("Backup").Return(nil)

		err := os.MkdirAll(outDir, 0750)
		require.NoError(t, err)
		err = os.WriteFile(leftoverFile, []byte("leftover file"), 0600)
		require.NoError(t, err)

		rb := &filesystemRollbacker{
			originalSvc: svc,
			backupDir:   outDir,
			installDir:  installDir,
			logger:      zaptest.NewLogger(t),
		}

		err = rb.Backup()
		require.NoError(t, err)

		require.FileExists(t, filepath.Join(outDir, "opentelemetry-java-contrib-jmx-metrics.jar"))
		require.FileExists(t, filepath.Join(outDir, "some-file.txt"))
		require.FileExists(t, filepath.Join(outDir, "plugins-dir", "plugin.txt"))
		require.NoDirExists(t, filepath.Join(outDir, "tmp-dir"))
		require.NoFileExists(t, leftoverFile)
	})
}

func TestRollbackerRollback(t *testing.T) {
	t.Run("Runs rollback actions in the correct order", func(t *testing.T) {
		seq := 0

		rb := &filesystemRollbacker{
			logger: zaptest.NewLogger(t),
		}

		for i := 0; i < 10; i++ {
			actionNum := i
			action := action_mocks.NewMockRollbackableAction(t)
			action.On("Rollback").Run(func(_ mock.Arguments) {
				// Rollback should be done in reverse order; So action 0
				// should be done last (10th action, seq == 9), while
				// the last action (action 9) should be done first (seq == 0)
				expectedSeq := 10 - actionNum - 1
				assert.Equal(t, expectedSeq, seq, "Expected action %d to occur at sequence %d", seq, expectedSeq)
				seq++
			}).Return(nil)

			rb.AppendAction(action)
		}

		rb.Rollback()
	})

	t.Run("Continues despite rollback errors", func(t *testing.T) {
		seq := 0

		rb := &filesystemRollbacker{
			logger: zaptest.NewLogger(t),
		}

		for i := 0; i < 10; i++ {
			actionNum := i
			action := action_mocks.NewMockRollbackableAction(t)

			call := action.On("Rollback").Run(func(_ mock.Arguments) {
				// Rollback should be done in reverse order; So action 0
				// should be done last (10th action, seq == 9), while
				// the last action (action 9) should be done first (seq == 0)
				expectedSeq := 10 - actionNum - 1
				assert.Equal(t, expectedSeq, seq, "Expected action %d to occur at sequence %d", seq, expectedSeq)
				seq++
			})

			if actionNum == 5 {
				call.Return(errors.New("failed to rollback"))
			} else {
				call.Return(nil)
			}

			rb.AppendAction(action)
		}

		rb.Rollback()
	})
}
