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
//go:build linux && integration_sysv

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
func TestLinuxSysVServiceInstall(t *testing.T) {
	t.Run("Test SysV install + uninstall", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		l := &linuxSysVService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.NoError(t, err)
		require.FileExists(t, installedServicePath)

		//We want to check that the service was actually loaded
		requireSysVServiceEnabledStatus(t, true)

		err = l.uninstall()
		require.NoError(t, err)
		require.NoFileExists(t, installedServicePath)

		//Make sure the service is no longer listed
		requireSysVServiceEnabledStatus(t, false)
	})

	t.Run("Test SysV stop + start", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		l := &linuxSysVService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.NoError(t, err)
		require.FileExists(t, installedServicePath)

		// We want to check that the service was actually loaded
		requireSysVServiceEnabledStatus(t, true)

		err = l.Start()
		require.NoError(t, err)

		requireSysVServiceRunningStatus(t, true)

		err = l.Stop()
		require.NoError(t, err)

		requireSysVServiceRunningStatus(t, false)

		err = l.uninstall()
		require.NoError(t, err)
		require.NoFileExists(t, installedServicePath)

		// Make sure the service is no longer listed
		requireSysVServiceEnabledStatus(t, false)
	})

	t.Run("Test SysV invalid path for input file", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		l := &linuxSysVService{
			newServiceFilePath:       filepath.Join("testdata", "does-not-exist.service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.ErrorContains(t, err, "failed to open input file")
		requireSysVServiceEnabledStatus(t, false)
	})

	t.Run("Test SysV invalid path for output file for install", func(t *testing.T) {
		installedServicePath := "/usr/lib/SysV/system/dir-does-not-exist/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		l := &linuxSysVService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.install()
		require.ErrorContains(t, err, "failed to open output file")
		requireSysVServiceEnabledStatus(t, false)
	})

	t.Run("Uninstall SysV fails if not installed", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		l := &linuxSysVService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.uninstall()
		require.ErrorContains(t, err, "chkconfig off failed:")
		requireSysVServiceEnabledStatus(t, false)
	})

	t.Run("Start SysV fails if service not found", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		l := &linuxSysVService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.Start()
		require.ErrorContains(t, err, "running service failed")
	})

	t.Run("Stop SysV fails if service not found", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		l := &linuxSysVService{
			newServiceFilePath:       filepath.Join("testdata", "linux-service"),
			serviceName:              "linux-service",
			installedServiceFilePath: installedServicePath,
			logger:                   zaptest.NewLogger(t),
		}

		err := l.Stop()
		require.ErrorContains(t, err, "running service failed")
	})

	t.Run("Backup SysV installed service succeeds", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		newServiceFile := filepath.Join("testdata", "linux-service")
		serviceFileContents, err := os.ReadFile(newServiceFile)
		require.NoError(t, err)

		installDir := t.TempDir()
		require.NoError(t, os.MkdirAll(path.BackupDir(installDir), 0775))

		d := &linuxSysVService{
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
		requireSysVServiceEnabledStatus(t, true)

		err = d.Backup()
		require.NoError(t, err)
		require.FileExists(t, path.BackupServiceFile(installDir))

		backupServiceContents, err := os.ReadFile(path.BackupServiceFile(installDir))

		require.Equal(t, serviceFileContents, backupServiceContents)
		require.NoError(t, d.uninstall())
	})

	t.Run("Backup SysV installed service fails if not installed", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		newServiceFile := filepath.Join("testdata", "linux-service")

		installDir := t.TempDir()
		require.NoError(t, os.MkdirAll(path.BackupDir(installDir), 0775))

		d := &linuxSysVService{
			newServiceFilePath:       newServiceFile,
			installedServiceFilePath: installedServicePath,
			serviceName:              "linux-service",
			installDir:               installDir,
			logger:                   zaptest.NewLogger(t),
		}

		err := d.Backup()
		require.ErrorContains(t, err, "failed to copy service file")
	})

	t.Run("Backup SysV installed service fails if output file already exists", func(t *testing.T) {
		installedServicePath := "/etc/init.d/linux-service"
		uninstallSysVService(t, installedServicePath, "linux-service")

		newServiceFile := filepath.Join("testdata", "linux-service")

		installDir := t.TempDir()
		require.NoError(t, os.MkdirAll(path.BackupDir(installDir), 0775))

		d := &linuxSysVService{
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
		requireSysVServiceEnabledStatus(t, true)

		// Write the backup file before creating it; Backup should
		// not ever overwrite an existing file
		os.WriteFile(path.BackupServiceFile(installDir), []byte("file exists"), 0600)

		err = d.Backup()
		require.ErrorContains(t, err, "failed to copy service file")
	})
}

// uninstallSysVService is a helper that uninstalls the service manually for test setup, in case it is somehow leftover.
func uninstallSysVService(t *testing.T, installedPath, serviceName string) {
	cmd := exec.Command("service", serviceName, "stop")
	_ = cmd.Run()

	cmd = exec.Command("chkconfig", serviceName, "off")
	_ = cmd.Run()

	err := os.RemoveAll(installedPath)
	require.NoError(t, err)
}

const exitCodeServiceDisabled = 1

func requireSysVServiceEnabledStatus(t *testing.T, enabled bool) {
	t.Helper()

	cmd := exec.Command("chkconfig", "linux-service")
	err := cmd.Run()

	if enabled {
		// If the service should be enabled, then we expect a 0 exit code, so no error is given
		require.NoError(t, err)
		return
	}

	eErr, ok := err.(*exec.ExitError)
	require.True(t, ok, "chkconfig exited with non-ExitError: %s", eErr)
	require.Equal(t, exitCodeServiceDisabled, eErr.ExitCode(), "unexpected exit code when asserting service is enabled: %d", eErr.ExitCode())
}

const exitCodeServiceInactive = 3

func requireSysVServiceRunningStatus(t *testing.T, running bool) {
	cmd := exec.Command("service", "linux-service", "status")
	err := cmd.Run()

	if running {
		// exit code 0 indicates service is running
		require.NoError(t, err)
		return
	}

	eErr, ok := err.(*exec.ExitError)
	require.True(t, ok, "service status exited with non-ExitError: %s", eErr)
	require.Equal(t, exitCodeServiceInactive, eErr.ExitCode(), "unexpected exit code when asserting service is not running: %d", eErr.ExitCode())
}
