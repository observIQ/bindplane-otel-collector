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

package install

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/observiq/bindplane-otel-collector/updater/internal/action"
	rb_mocks "github.com/observiq/bindplane-otel-collector/updater/internal/rollback/mocks"
	"github.com/observiq/bindplane-otel-collector/updater/internal/service/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestInstallArtifacts(t *testing.T) {
	t.Run("Installs artifacts correctly", func(t *testing.T) {
		outDir := t.TempDir()
		svc := mocks.NewMockService(t)
		rb := rb_mocks.NewMockRollbacker(t)

		installer := &archiveInstaller{
			latestDir:  filepath.Join("testdata", "example-install"),
			installDir: outDir,
			backupDir:  filepath.Join("testdata", "rollback"),
			svc:        svc,
			logger:     zaptest.NewLogger(t),
		}

		latestJarPath := filepath.Join(installer.latestDir, "opentelemetry-java-contrib-jmx-metrics.jar")
		_, err := os.Create(latestJarPath)
		require.NoError(t, err)
		err = os.WriteFile(latestJarPath, []byte("# The new jar file"), 0660)
		require.NoError(t, err)

		outDirConfig := filepath.Join(outDir, "config.yaml")
		outDirLogging := filepath.Join(outDir, "logging.yaml")
		outDirManager := filepath.Join(outDir, "manager.yaml")

		err = os.WriteFile(outDirConfig, []byte("# The original config file"), 0600)
		require.NoError(t, err)
		err = os.WriteFile(outDirLogging, []byte("# The original logging file"), 0600)
		require.NoError(t, err)
		err = os.WriteFile(outDirManager, []byte("# The original manager file"), 0600)
		require.NoError(t, err)

		svc.On("Update").Once().Return(nil)
		svc.On("Start").Once().Return(nil)

		actions := []action.RollbackableAction{}
		rb.On("AppendAction", mock.Anything).Run(func(args mock.Arguments) {
			action := args.Get(0).(action.RollbackableAction)
			actions = append(actions, action)
		})

		err = installer.Install(rb)
		require.NoError(t, err)

		contentsEqual(t, outDirConfig, "# The original config file")
		contentsEqual(t, outDirManager, "# The original manager file")
		contentsEqual(t, outDirLogging, "# The original logging file")

		require.FileExists(t, filepath.Join(outDir, "opentelemetry-java-contrib-jmx-metrics.jar"))
		require.FileExists(t, filepath.Join(outDir, "test.txt"))
		require.DirExists(t, filepath.Join(outDir, "test-folder"))
		require.FileExists(t, filepath.Join(outDir, "test-folder", "another-test.txt"))

		contentsEqual(t, filepath.Join(outDir, "opentelemetry-java-contrib-jmx-metrics.jar"), "# The new jar file")
		contentsEqual(t, filepath.Join(outDir, "test.txt"), "This is a test file\n")
		contentsEqual(t, filepath.Join(outDir, "test-folder", "another-test.txt"), "This is a nested text file\n")

		copyTestTxtAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("test.txt"),
			filepath.Join(installer.installDir, "test.txt"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyTestTxtAction.FileCreated = true

		copyJarAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("opentelemetry-java-contrib-jmx-metrics.jar"),
			filepath.Join(installer.installDir, "opentelemetry-java-contrib-jmx-metrics.jar"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyJarAction.FileCreated = true

		copyNestedTestTxtAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("test-folder", "another-test.txt"),
			filepath.Join(installer.installDir, "test-folder", "another-test.txt"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyNestedTestTxtAction.FileCreated = true

		require.Equal(t, len(actions), 5)
		require.Contains(t, actions, copyJarAction)
		require.Contains(t, actions, copyNestedTestTxtAction)
		require.Contains(t, actions, copyTestTxtAction)
		require.Contains(t, actions, action.NewServiceUpdateAction(installer.logger, installer.installDir))
		require.Contains(t, actions, action.NewServiceStartAction(svc))
	})

	if runtime.GOOS != "windows" {
		t.Run("Installs artifacts correctly when linux jmx jar", func(t *testing.T) {
			jarDir := t.TempDir()
			specialJarPath := filepath.Join(jarDir, "opentelemetry-java-contrib-jmx-metrics.jar")
			_, err := os.Create(specialJarPath)
			require.NoError(t, err)
			err = os.WriteFile(specialJarPath, []byte("# The original jar file"), 0600)
			require.NoError(t, err)
			outDir := filepath.Join(jarDir, "installdir")
			os.MkdirAll(outDir, 0700)

			svc := mocks.NewMockService(t)
			rb := rb_mocks.NewMockRollbacker(t)

			installer := &archiveInstaller{
				latestDir:  filepath.Join("testdata", "example-install"),
				installDir: outDir,
				backupDir:  filepath.Join("testdata", "rollback"),
				svc:        svc,
				logger:     zaptest.NewLogger(t),
			}

			latestJarPath := filepath.Join(installer.latestDir, "opentelemetry-java-contrib-jmx-metrics.jar")
			_, err = os.Create(latestJarPath)
			require.NoError(t, err)
			err = os.WriteFile(latestJarPath, []byte("# The new jar file"), 0660)
			require.NoError(t, err)

			outDirConfig := filepath.Join(outDir, "config.yaml")
			outDirLogging := filepath.Join(outDir, "logging.yaml")
			outDirManager := filepath.Join(outDir, "manager.yaml")

			err = os.WriteFile(outDirConfig, []byte("# The original config file"), 0600)
			require.NoError(t, err)
			err = os.WriteFile(outDirLogging, []byte("# The original logging file"), 0600)
			require.NoError(t, err)
			err = os.WriteFile(outDirManager, []byte("# The original manager file"), 0600)
			require.NoError(t, err)

			svc.On("Update").Once().Return(nil)
			svc.On("Start").Once().Return(nil)

			actions := []action.RollbackableAction{}
			rb.On("AppendAction", mock.Anything).Run(func(args mock.Arguments) {
				action := args.Get(0).(action.RollbackableAction)
				actions = append(actions, action)
			})

			err = installer.Install(rb)
			require.NoError(t, err)

			contentsEqual(t, outDirConfig, "# The original config file")
			contentsEqual(t, outDirManager, "# The original manager file")
			contentsEqual(t, outDirLogging, "# The original logging file")

			require.FileExists(t, filepath.Join(jarDir, "opentelemetry-java-contrib-jmx-metrics.jar"))
			require.FileExists(t, filepath.Join(outDir, "test.txt"))
			require.DirExists(t, filepath.Join(outDir, "test-folder"))
			require.FileExists(t, filepath.Join(outDir, "test-folder", "another-test.txt"))

			contentsEqual(t, filepath.Join(jarDir, "opentelemetry-java-contrib-jmx-metrics.jar"), "# The new jar file")
			contentsEqual(t, filepath.Join(outDir, "test.txt"), "This is a test file\n")
			contentsEqual(t, filepath.Join(outDir, "test-folder", "another-test.txt"), "This is a nested text file\n")

			copyTestTxtAction, err := action.NewCopyFileAction(
				installer.logger,
				filepath.Join("test.txt"),
				filepath.Join(installer.installDir, "test.txt"),
				installer.backupDir,
			)
			require.NoError(t, err)
			copyTestTxtAction.FileCreated = true

			copyJarAction, err := action.NewCopyFileAction(
				installer.logger,
				filepath.Join("opentelemetry-java-contrib-jmx-metrics.jar"),
				filepath.Join(jarDir, "opentelemetry-java-contrib-jmx-metrics.jar"),
				installer.backupDir,
			)
			require.NoError(t, err)
			copyJarAction.FileCreated = false

			copyNestedTestTxtAction, err := action.NewCopyFileAction(
				installer.logger,
				filepath.Join("test-folder", "another-test.txt"),
				filepath.Join(installer.installDir, "test-folder", "another-test.txt"),
				installer.backupDir,
			)
			require.NoError(t, err)
			copyNestedTestTxtAction.FileCreated = true

			require.Equal(t, len(actions), 5)
			require.Contains(t, actions, copyJarAction)
			require.Contains(t, actions, copyNestedTestTxtAction)
			require.Contains(t, actions, copyTestTxtAction)
			require.Contains(t, actions, action.NewServiceUpdateAction(installer.logger, installer.installDir))
			require.Contains(t, actions, action.NewServiceStartAction(svc))
		})
	} else {
		t.Skip()
	}

	t.Run("Update fails", func(t *testing.T) {
		outDir := t.TempDir()
		svc := mocks.NewMockService(t)
		rb := rb_mocks.NewMockRollbacker(t)
		installer := &archiveInstaller{
			latestDir:  filepath.Join("testdata", "example-install"),
			installDir: outDir,
			backupDir:  filepath.Join("testdata", "rollback"),
			svc:        svc,
			logger:     zaptest.NewLogger(t),
		}

		latestJarPath := filepath.Join(installer.latestDir, "opentelemetry-java-contrib-jmx-metrics.jar")
		_, err := os.Create(latestJarPath)
		require.NoError(t, err)
		err = os.WriteFile(latestJarPath, []byte("# The new jar file"), 0660)
		require.NoError(t, err)

		svc.On("Update").Once().Return(errors.New("uninstall failed"))

		actions := []action.RollbackableAction{}
		rb.On("AppendAction", mock.Anything).Run(func(args mock.Arguments) {
			action := args.Get(0).(action.RollbackableAction)
			actions = append(actions, action)
		})

		err = installer.Install(rb)
		require.ErrorContains(t, err, "failed to update service")
		copyTestTxtAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("test.txt"),
			filepath.Join(installer.installDir, "test.txt"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyTestTxtAction.FileCreated = true

		copyNestedTestTxtAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("test-folder", "another-test.txt"),
			filepath.Join(installer.installDir, "test-folder", "another-test.txt"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyNestedTestTxtAction.FileCreated = true

		copyJarAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("opentelemetry-java-contrib-jmx-metrics.jar"),
			filepath.Join(installer.installDir, "opentelemetry-java-contrib-jmx-metrics.jar"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyJarAction.FileCreated = true

		require.Equal(t, len(actions), 3)
		require.Contains(t, actions, copyJarAction)
		require.Contains(t, actions, copyNestedTestTxtAction)
		require.Contains(t, actions, copyTestTxtAction)
	})

	t.Run("Start fails", func(t *testing.T) {
		outDir := t.TempDir()
		svc := mocks.NewMockService(t)
		rb := rb_mocks.NewMockRollbacker(t)
		installer := &archiveInstaller{
			latestDir:  filepath.Join("testdata", "example-install"),
			installDir: outDir,
			backupDir:  filepath.Join("testdata", "rollback"),
			svc:        svc,
			logger:     zaptest.NewLogger(t),
		}

		latestJarPath := filepath.Join(installer.latestDir, "opentelemetry-java-contrib-jmx-metrics.jar")
		_, err := os.Create(latestJarPath)
		require.NoError(t, err)
		err = os.WriteFile(latestJarPath, []byte("# The new jar file"), 0660)
		require.NoError(t, err)

		svc.On("Update").Once().Return(nil)
		svc.On("Start").Once().Return(errors.New("start failed"))

		actions := []action.RollbackableAction{}
		rb.On("AppendAction", mock.Anything).Run(func(args mock.Arguments) {
			action := args.Get(0).(action.RollbackableAction)
			actions = append(actions, action)
		})

		err = installer.Install(rb)
		require.ErrorContains(t, err, "failed to start service")

		copyTestTxtAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("test.txt"),
			filepath.Join(installer.installDir, "test.txt"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyTestTxtAction.FileCreated = true

		copyNestedTestTxtAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("test-folder", "another-test.txt"),
			filepath.Join(installer.installDir, "test-folder", "another-test.txt"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyNestedTestTxtAction.FileCreated = true

		copyJarAction, err := action.NewCopyFileAction(
			installer.logger,
			filepath.Join("opentelemetry-java-contrib-jmx-metrics.jar"),
			filepath.Join(installer.installDir, "opentelemetry-java-contrib-jmx-metrics.jar"),
			installer.backupDir,
		)
		require.NoError(t, err)
		copyJarAction.FileCreated = true

		require.Equal(t, len(actions), 4)
		require.Contains(t, actions, copyJarAction)
		require.Contains(t, actions, copyNestedTestTxtAction)
		require.Contains(t, actions, copyTestTxtAction)
		require.Contains(t, actions, action.NewServiceUpdateAction(installer.logger, installer.installDir))
	})

	t.Run("Latest dir does not exist", func(t *testing.T) {
		outDir := t.TempDir()
		svc := mocks.NewMockService(t)
		rb := rb_mocks.NewMockRollbacker(t)
		installer := &archiveInstaller{
			latestDir:  filepath.Join("testdata", "non-existent-dir"),
			installDir: outDir,
			svc:        svc,
			logger:     zaptest.NewLogger(t),
		}

		err := installer.Install(rb)
		require.ErrorContains(t, err, "failed to install new files")
	})
}

func contentsEqual(t *testing.T, path, expectedContents string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	// Replace \r\n with \n to normalize for windows tests.
	contents = bytes.ReplaceAll(contents, []byte("\r\n"), []byte("\n"))
	require.Equal(t, []byte(expectedContents), contents)
}
