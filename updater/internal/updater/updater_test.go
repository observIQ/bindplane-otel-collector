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

package updater

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/packagestate"
	"github.com/observiq/bindplane-otel-collector/updater/internal/action"
	install_mocks "github.com/observiq/bindplane-otel-collector/updater/internal/install/mocks"
	rollback_mocks "github.com/observiq/bindplane-otel-collector/updater/internal/rollback/mocks"
	service_mocks "github.com/observiq/bindplane-otel-collector/updater/internal/service/mocks"
	"github.com/observiq/bindplane-otel-collector/updater/internal/state"
	state_mocks "github.com/observiq/bindplane-otel-collector/updater/internal/state/mocks"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewUpdater(t *testing.T) {
	t.Run("New updater is created successfully", func(t *testing.T) {
		installDir := "testdata"
		logger := zaptest.NewLogger(t)
		updater, err := NewUpdater(logger, installDir)
		require.NoError(t, err)
		require.NotNil(t, updater)
		assert.NotNil(t, updater.installer)
		assert.NotNil(t, updater.svc)
		assert.NotNil(t, updater.rollbacker)
		assert.NotNil(t, updater.monitor)
		assert.NotNil(t, updater.logger)
		require.Equal(t, installDir, updater.installDir)
	})

	t.Run("New updater fails due to missing package statuses", func(t *testing.T) {
		installDir := t.TempDir()
		logger := zaptest.NewLogger(t)
		updater, err := NewUpdater(logger, installDir)
		require.ErrorContains(t, err, "failed to create monitor")
		require.Nil(t, updater)
	})
}

func TestUpdaterUpdate(t *testing.T) {
	t.Run("Update is successful", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(nil)
		installer.On("Install", rollbacker).Times(1).Return(nil)
		monitor.On("MonitorForSuccess", mock.Anything, packagestate.CollectorPackageName).Times(1).Return(nil)

		err := updater.Update()
		require.NoError(t, err)
	})

	t.Run("Service stop fails", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		svc.On("Stop").Times(1).Return(errors.New("insufficient permissions"))

		err := updater.Update()
		require.ErrorContains(t, err, "failed to stop service")
	})

	t.Run("Backup fails", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		err := errors.New("insufficient permissions")

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(err)
		monitor.On("SetState", packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err).Times(1).Return(nil)
		rollbacker.On("Rollback").Times(1).Return()

		err = updater.Update()
		require.ErrorContains(t, err, "failed to backup")
	})

	t.Run("Backup fails, set state fails", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		err := errors.New("insufficient permissions")

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(err)
		monitor.On("SetState", packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err).Times(1).Return(errors.New("insufficient permissions"))
		rollbacker.On("Rollback").Times(1).Return()

		err = updater.Update()
		require.ErrorContains(t, err, "failed to backup")
	})

	t.Run("Install fails", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		err := errors.New("insufficient permissions")

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(nil)
		installer.On("Install", rollbacker).Times(1).Return(err)
		monitor.On("SetState", packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err).Times(1).Return(nil)
		rollbacker.On("Rollback").Times(1).Return()

		err = updater.Update()
		require.ErrorContains(t, err, "failed to install")
	})

	t.Run("Install fails, set state fails", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		err := errors.New("insufficient permissions")

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(nil)
		installer.On("Install", rollbacker).Times(1).Return(err)
		monitor.On("SetState", packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err).Times(1).Return(errors.New("insufficient permissions"))
		rollbacker.On("Rollback").Times(1).Return()

		err = updater.Update()
		require.ErrorContains(t, err, "failed to install")
	})

	t.Run("Monitor for success fails to monitor", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		err := errors.New("insufficient permissions")

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(nil)
		installer.On("Install", rollbacker).Times(1).Return(nil)
		monitor.On("MonitorForSuccess", mock.Anything, packagestate.CollectorPackageName).Times(1).Return(err)
		monitor.On("SetState", packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err).Times(1).Return(nil)
		rollbacker.On("Rollback").Times(1).Return()

		err = updater.Update()
		require.ErrorContains(t, err, "failed while monitoring for success")
	})

	t.Run("Monitor for success fails to monitor, set state fails", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		err := errors.New("insufficient permissions")

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(nil)
		installer.On("Install", rollbacker).Times(1).Return(nil)
		monitor.On("MonitorForSuccess", mock.Anything, packagestate.CollectorPackageName).Times(1).Return(err)
		monitor.On("SetState", packagestate.CollectorPackageName, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, err).Times(1).Return(errors.New("insufficient permissions"))
		rollbacker.On("Rollback").Times(1).Return()

		err = updater.Update()
		require.ErrorContains(t, err, "failed while monitoring for success")
	})

	t.Run("Monitor for success finds error in package statuses", func(t *testing.T) {
		installDir := t.TempDir()

		installer := install_mocks.NewMockInstaller(t)
		svc := service_mocks.NewMockService(t)
		rollbacker := rollback_mocks.NewMockRollbacker(t)
		monitor := state_mocks.NewMockMonitor(t)

		updater := &Updater{
			installDir:               installDir,
			installer:                installer,
			svc:                      svc,
			rollbacker:               rollbacker,
			monitor:                  monitor,
			logger:                   zaptest.NewLogger(t),
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		svc.On("Stop").Times(1).Return(nil)
		rollbacker.On("AppendAction", action.NewServiceStopAction(svc)).Times(1).Return()
		rollbacker.On("Backup").Times(1).Return(nil)
		installer.On("Install", rollbacker).Times(1).Return(nil)
		monitor.On("MonitorForSuccess", mock.Anything, packagestate.CollectorPackageName).Times(1).Return(state.ErrFailedStatus)
		rollbacker.On("Rollback").Times(1).Return()

		err := updater.Update()
		require.ErrorContains(t, err, "failed while monitoring for success")
	})
}

func TestGenerateServiceFiles(t *testing.T) {
	t.Run("Generate service files successfully", func(t *testing.T) {
		installDir := t.TempDir()
		logger := zaptest.NewLogger(t)
		u := &Updater{
			installDir:               installDir,
			logger:                   logger,
			installedSystemdUnitPath: filepath.Join("testdata", "observiq-otel-collector.service.golden"),
		}

		// Cleanup the directory after test
		defer os.RemoveAll(installDir)

		err := u.generateServiceFiles()
		require.NoError(t, err)

		// Compare the generated files with golden files
		compareFiles(t, filepath.Join(installDir, "install", "observiq-otel-collector.service"), u.installedSystemdUnitPath)

		// Check file permissions
		checkFilePermissions(t, filepath.Join(installDir, "install", "observiq-otel-collector.service"), 0640)
		checkFilePermissions(t, filepath.Join(installDir, "install", "observiq-otel-collector"), 0755)
	})
}

func compareFiles(t *testing.T, generatedFile, goldenFile string) {
	generated, err := os.ReadFile(generatedFile)
	require.NoError(t, err, fmt.Sprintf("file: %s", generatedFile))

	golden, err := os.ReadFile(goldenFile)
	require.NoError(t, err, fmt.Sprintf("file: %s", goldenFile))

	// Trim the contents to ignore trailing newlines or spaces
	generated = bytes.TrimSpace(generated)
	golden = bytes.TrimSpace(golden)

	require.Equal(t, string(golden), string(generated))
}

func checkFilePermissions(t *testing.T, filePath string, expectedPerm os.FileMode) {
	info, err := os.Stat(filePath)
	require.NoError(t, err)

	require.Equal(t, expectedPerm, info.Mode().Perm())
}

func TestReadGroupFromSystemdFile(t *testing.T) {
	t.Run("Extract Group from systemd unit file", func(t *testing.T) {
		u := &Updater{
			installedSystemdUnitPath: "testdata/observiq-otel-collector.service.golden",
		}

		group, err := u.readGroupFromSystemdFile()
		require.NoError(t, err)
		require.Equal(t, "bdot", group)
	})
}
