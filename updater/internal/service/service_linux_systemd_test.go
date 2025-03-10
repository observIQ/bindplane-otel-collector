// Copyright  Bindplane, Inc..
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

// an elevated user is needed to run the service tests
//go:build linux && integration

package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// NOTE: These tests must run as root in order to pass
func TestLinuxSystemdServiceInstall(t *testing.T) {
	t.Run("Test systemd install + uninstall", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		l := &linuxSystemdService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.NoError(t, err)
		require.FileExists(t, installedServicePath)

		//We want to check that the service was actually loaded
		requireSystemdServiceLoadedStatus(t, true)

		err = l.uninstall()
		require.NoError(t, err)
		require.NoFileExists(t, installedServicePath)

		//Make sure the service is no longer listed
		requireSystemdServiceLoadedStatus(t, false)
	})

	t.Run("Test systemd stop + start", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		l := &linuxSystemdService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.NoError(t, err)
		require.FileExists(t, installedServicePath)

		// We want to check that the service was actually loaded
		requireSystemdServiceLoadedStatus(t, true)

		err = l.Start()
		require.NoError(t, err)

		requireSystemdServiceRunningStatus(t, true)

		err = l.Stop()
		require.NoError(t, err)

		requireSystemdServiceRunningStatus(t, false)

		err = l.uninstall()
		require.NoError(t, err)
		require.NoFileExists(t, installedServicePath)

		// Make sure the service is no longer listed
		requireSystemdServiceLoadedStatus(t, false)
	})

	t.Run("Test systemd invalid path for input file", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		l := &linuxSystemdService{
			newServiceFilePath:       filepath.Join("testdata", "does-not-exist.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.ErrorContains(t, err, "failed to open input file")
		requireSystemdServiceLoadedStatus(t, false)
	})

	t.Run("Test systemd invalid path for output file for install", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/dir-does-not-exist/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		l := &linuxSystemdService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.ErrorContains(t, err, "failed to open output file")
		requireSystemdServiceLoadedStatus(t, false)
	})

	t.Run("Uninstall systemd fails if not installed", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		l := &linuxSystemdService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.uninstall()
		require.ErrorContains(t, err, "failed to disable unit")
		requireSystemdServiceLoadedStatus(t, false)
	})

	t.Run("Start systemd fails if service not found", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		l := &linuxSystemdService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.Start()
		require.ErrorContains(t, err, "running systemctl failed")
	})

	t.Run("Stop systemd fails if service not found", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		l := &linuxSystemdService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.Stop()
		require.ErrorContains(t, err, "running systemctl failed")
	})

	t.Run("Backup systemd installed service succeeds", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		newServiceFile := filepath.Join("testdata", "linux-service.service")
		serviceFileContents, err := os.ReadFile(newServiceFile)
		require.NoError(t, err)

		installDir := t.TempDir()
		require.NoError(t, os.MkdirAll(path.BackupDir(installDir), 0775))

		d := &linuxSystemdService{
			newServiceFilePath:       newServiceFile,
			installedServiceFilePath: installedServicePath,
			serviceName:              "linux-service",
			installDir:               installDir,
			logger:                   zaptest.NewLogger(t),
		}

		err = d.install()
		require.NoError(t, err)
		require.FileExists(t, installedServicePath)

		// We want to check that the service was actually loaded
		requireSystemdServiceLoadedStatus(t, true)

		err = d.Backup()
		require.NoError(t, err)
		require.FileExists(t, path.BackupServiceFile(installDir))

		backupServiceContents, err := os.ReadFile(path.BackupServiceFile(installDir))

		require.Equal(t, serviceFileContents, backupServiceContents)
		require.NoError(t, d.uninstall())
	})

	t.Run("Backup systemd installed service fails if not installed", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		newServiceFile := filepath.Join("testdata", "linux-service.service")

		installDir := t.TempDir()
		require.NoError(t, os.MkdirAll(path.BackupDir(installDir), 0775))

		d := &linuxSystemdService{
			newServiceFilePath:       newServiceFile,
			installedServiceFilePath: installedServicePath,
			serviceName:              "linux-service",
			installDir:               installDir,
			logger:                   zaptest.NewLogger(t),
		}

		err := d.Backup()
		require.ErrorContains(t, err, "failed to copy service file")
	})

	t.Run("Backup systemd installed service fails if output file already exists", func(t *testing.T) {
		installedServicePath := "/usr/lib/systemd/system/linux-service.service"
		uninstallSystemdService(t, installedServicePath, "linux-service")

		newServiceFile := filepath.Join("testdata", "linux-service.service")

		installDir := t.TempDir()
		require.NoError(t, os.MkdirAll(path.BackupDir(installDir), 0775))

		d := &linuxSystemdService{
			newServiceFilePath:       newServiceFile,
			installedServiceFilePath: installedServicePath,
			serviceName:              "linux-service",
			installDir:               installDir,
			logger:                   zaptest.NewLogger(t),
		}

		err := d.install()
		require.NoError(t, err)
		require.FileExists(t, installedServicePath)

		// We want to check that the service was actually loaded
		requireSystemdServiceLoadedStatus(t, true)

		// Write the backup file before creating it; Backup should
		// not ever overwrite an existing file
		os.WriteFile(path.BackupServiceFile(installDir), []byte("file exists"), 0600)

		err = d.Backup()
		require.ErrorContains(t, err, "failed to copy service file")
	})
}

// uninstallSystemdService is a helper that uninstalls the service manually for test setup, in case it is somehow leftover.
func uninstallSystemdService(t *testing.T, installedPath, serviceName string) {
	cmd := exec.Command("systemctl", "stop", serviceName)
	_ = cmd.Run()

	cmd = exec.Command("systemctl", "disable", serviceName)
	_ = cmd.Run()

	err := os.RemoveAll(installedPath)
	require.NoError(t, err)

	cmd = exec.Command("systemctl", "daemon-reload")
	_ = cmd.Run()
}

const exitCodeServiceNotFound = 4
const exitCodeServiceInactive = 3

func requireSystemdServiceLoadedStatus(t *testing.T, loaded bool) {
	t.Helper()

	cmd := exec.Command("systemctl", "status", "linux-service")
	err := cmd.Run()
	require.Error(t, err, "expected non-zero exit code from 'systemctl status linux-service'")

	eErr, ok := err.(*exec.ExitError)
	if loaded {
		// If the service should be loaded, then we expect a 0 exit code, so no error is given
		require.Equal(t, exitCodeServiceInactive, eErr.ExitCode(), "unexpected exit code when asserting service is unloaded: %d", eErr.ExitCode())
		return
	}

	require.True(t, ok, "systemctl status exited with non-ExitError: %s", eErr)
	require.Equal(t, exitCodeServiceNotFound, eErr.ExitCode(), "unexpected exit code when asserting service is unloaded: %d", eErr.ExitCode())
}

func requireSystemdServiceRunningStatus(t *testing.T, running bool) {
	cmd := exec.Command("systemctl", "status", "linux-service")
	err := cmd.Run()

	if running {
		// exit code 0 indicates service is loaded & running
		require.NoError(t, err)
		return
	}

	eErr, ok := err.(*exec.ExitError)
	require.True(t, ok, "systemctl status exited with non-ExitError: %s", eErr)
	require.Equal(t, exitCodeServiceInactive, eErr.ExitCode(), "unexpected exit code when asserting service is not running: %d", eErr.ExitCode())
}
