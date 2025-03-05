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

// an elevated user is needed to run the service tests
//go:build windows && integration

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func TestWindowsServiceInstall(t *testing.T) {
	t.Run("Test Update", func(t *testing.T) {
		tempDir := t.TempDir()
		testProductName := "Test Product"
		testServiceName := "windows-service"

		serviceJSON := filepath.Join(tempDir, "windows-service.json")
		testServiceProgram := filepath.Join(tempDir, "windows-service.exe")
		serviceGoFile, err := filepath.Abs(filepath.Join("testdata", "test-windows-service.go"))
		require.NoError(t, err)

		writeServiceFile(t, serviceJSON, filepath.Join("testdata", "windows-service.json"), serviceGoFile)
		compileProgram(t, serviceGoFile, testServiceProgram)

		installService(t,
			testServiceProgram,
			testServiceName,
			"Test Windows Service - Initial Display Name",
			"This is the test windows service; initial desription",
			mgr.StartAutomatic,
			false)

		t.Cleanup(func() {
			uninstallService(t, testServiceName)
			time.Sleep(100 * time.Millisecond)
		})

		w := &windowsService{
			newServiceFilePath: serviceJSON,
			serviceName:        testServiceName,
			productName:        testProductName,
			installDir:         tempDir,
			logger:             zaptest.NewLogger(t),
		}

		err = w.Update()
		require.NoError(t, err)

		//We want to check that the service was actually loaded
		requireServiceLoadedStatus(t, true)

		requireServiceConfigMatches(t,
			fmt.Sprintf("\"%s\"", testServiceProgram),
			"windows-service",
			mgr.StartAutomatic,
			"Test Windows Service",
			"This is a windows service to test",
			true,
			[]string{
				"--config",
				fmt.Sprintf("\"%s\"", filepath.Join(tempDir, "test.yaml")),
			},
		)
	})

	t.Run("Test update (space in install folder)", func(t *testing.T) {
		tempDir := filepath.Join(t.TempDir(), "temp dir with spaces")
		require.NoError(t, os.MkdirAll(tempDir, 0777))
		testProductName := "Test Product"
		testServiceName := "windows-service"

		serviceJSON := filepath.Join(tempDir, "windows-service.json")
		testServiceProgram := filepath.Join(tempDir, "windows-service.exe")
		serviceGoFile, err := filepath.Abs(filepath.Join("testdata", "test-windows-service.go"))
		require.NoError(t, err)

		writeServiceFile(t, serviceJSON, filepath.Join("testdata", "windows-service.json"), serviceGoFile)
		compileProgram(t, serviceGoFile, testServiceProgram)

		installService(t,
			testServiceProgram,
			testServiceName,
			"Test Windows Service - Initial Display Name",
			"This is the test windows service; initial desription",
			mgr.StartAutomatic,
			false)

		t.Cleanup(func() {
			uninstallService(t, testServiceName)
			time.Sleep(100 * time.Millisecond)
		})

		w := &windowsService{
			newServiceFilePath: serviceJSON,
			serviceName:        testServiceName,
			productName:        testProductName,
			installDir:         tempDir,
			logger:             zaptest.NewLogger(t),
		}

		err = w.Update()
		require.NoError(t, err)

		//We want to check that the service was actually loaded
		requireServiceLoadedStatus(t, true)

		requireServiceConfigMatches(t,
			testServiceProgram,
			"windows-service",
			mgr.StartAutomatic,
			"Test Windows Service",
			"This is a windows service to test",
			true,
			[]string{
				"--config",
				filepath.Join(tempDir, "test.yaml"),
			},
		)
	})

	t.Run("Test stop + start", func(t *testing.T) {
		tempDir := t.TempDir()
		testProductName := "Test Product"
		testServiceName := "windows-service"

		serviceJSON := filepath.Join(tempDir, "windows-service.json")
		testServiceProgram := filepath.Join(tempDir, "windows-service.exe")
		serviceGoFile, err := filepath.Abs(filepath.Join("testdata", "test-windows-service.go"))
		require.NoError(t, err)

		writeServiceFile(t, serviceJSON, filepath.Join("testdata", "windows-service.json"), serviceGoFile)
		compileProgram(t, serviceGoFile, testServiceProgram)

		installService(t,
			testServiceProgram,
			testServiceName,
			"Test Windows Service - Initial Display Name",
			"This is the test windows service; initial desription",
			mgr.StartManual,
			false)

		t.Cleanup(func() {
			uninstallService(t, testServiceName)
			time.Sleep(100 * time.Millisecond)
		})

		w := &windowsService{
			newServiceFilePath: serviceJSON,
			serviceName:        "windows-service",
			productName:        testProductName,
			installDir:         tempDir,
			logger:             zaptest.NewLogger(t),
		}

		// We want to check that the service was actually loaded
		requireServiceLoadedStatus(t, true)

		err = w.Start()
		require.NoError(t, err)

		requireServiceRunningStatus(t, true)

		err = w.Stop()
		require.NoError(t, err)

		requireServiceRunningStatus(t, false)
	})

	t.Run("Test invalid path for input file on update", func(t *testing.T) {
		tempDir := t.TempDir()
		testProductName := "Test Product"
		testServiceName := "windows-service"

		serviceJSON := filepath.Join(tempDir, "windows-service.json")
		testServiceProgram := filepath.Join(tempDir, "windows-service.exe")
		serviceGoFile, err := filepath.Abs(filepath.Join("testdata", "test-windows-service.go"))
		require.NoError(t, err)

		writeServiceFile(t, serviceJSON, filepath.Join("testdata", "windows-service.json"), serviceGoFile)
		compileProgram(t, serviceGoFile, testServiceProgram)

		installService(t,
			testServiceProgram,
			testServiceName,
			"Test Windows Service - Initial Display Name",
			"This is the test windows service; initial desription",
			mgr.StartManual,
			false)

		t.Cleanup(func() {
			uninstallService(t, testServiceName)
			time.Sleep(100 * time.Millisecond)
		})

		w := &windowsService{
			newServiceFilePath: filepath.Join(tempDir, "not-a-valid-service.json"),
			serviceName:        testServiceName,
			productName:        testProductName,
			installDir:         tempDir,
			logger:             zaptest.NewLogger(t),
		}

		err = w.Update()
		require.ErrorContains(t, err, "The system cannot find the file specified.")
		requireServiceLoadedStatus(t, true)
		requireServiceRunningStatus(t, false)
	})

	t.Run("Update fails if not installed", func(t *testing.T) {
		tempDir := t.TempDir()
		testProductName := "Test Product"
		testServiceName := "windows-service"

		serviceJSON := filepath.Join(tempDir, "windows-service.json")
		testServiceProgram := filepath.Join(tempDir, "windows-service.exe")
		serviceGoFile, err := filepath.Abs(filepath.Join("testdata", "test-windows-service.go"))
		require.NoError(t, err)

		writeServiceFile(t, serviceJSON, filepath.Join("testdata", "windows-service.json"), serviceGoFile)
		compileProgram(t, serviceGoFile, testServiceProgram)

		w := &windowsService{
			newServiceFilePath: serviceJSON,
			serviceName:        testServiceName,
			installDir:         tempDir,
			productName:        testProductName,
		}

		err = w.Update()
		require.ErrorContains(t, err, "failed to open service")
		requireServiceLoadedStatus(t, false)
	})

	t.Run("Start fails if service not found", func(t *testing.T) {
		tempDir := t.TempDir()
		testProductName := "Test Product"

		serviceJSON := filepath.Join(tempDir, "windows-service.json")

		w := &windowsService{
			newServiceFilePath: serviceJSON,
			serviceName:        "windows-service",
			productName:        testProductName,
			installDir:         tempDir,
			logger:             zaptest.NewLogger(t),
		}

		err := w.Start()
		require.ErrorContains(t, err, "failed to open service")
	})

	t.Run("Stop fails if service not found", func(t *testing.T) {
		tempDir := t.TempDir()
		testProductName := "Test Product"

		serviceJSON := filepath.Join(tempDir, "windows-service.json")

		w := &windowsService{
			newServiceFilePath: serviceJSON,
			serviceName:        "windows-service",
			productName:        testProductName,
			installDir:         tempDir,
			logger:             zaptest.NewLogger(t),
		}

		err := w.Stop()
		require.ErrorContains(t, err, "failed to open service")
	})

	t.Run("Test backup works", func(t *testing.T) {
		tempDir := t.TempDir()
		installDir, err := filepath.Abs(filepath.Join(tempDir, "install directory"))
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(path.BackupDir(installDir), 0775))

		testProductName := "Test Product"
		testServiceName := "windows-service"

		serviceJSON := filepath.Join(installDir, "windows-service.json")
		testServiceProgram := filepath.Join(installDir, "windows-service.exe")
		serviceGoFile, err := filepath.Abs(filepath.Join("testdata", "test-windows-service.go"))
		require.NoError(t, err)

		writeServiceFile(t, serviceJSON, filepath.Join("testdata", "windows-service.json"), serviceGoFile)
		compileProgram(t, serviceGoFile, testServiceProgram)

		installService(t,
			testServiceProgram,
			testServiceName,
			"Test Windows Service - Initial Display Name",
			"This is the test windows service; initial desription",
			mgr.StartManual,
			false)

		t.Cleanup(func() {
			uninstallService(t, testServiceName)
			time.Sleep(100 * time.Millisecond)
		})

		w := &windowsService{
			newServiceFilePath: serviceJSON,
			serviceName:        "windows-service",
			productName:        testProductName,
			installDir:         installDir,
			logger:             zaptest.NewLogger(t),
		}

		require.NoError(t, w.Update())

		//We want to check that the service was actually loaded
		requireServiceLoadedStatus(t, true)

		requireServiceConfigMatches(t,
			testServiceProgram,
			"windows-service",
			mgr.StartAutomatic,
			"Test Windows Service",
			"This is a windows service to test",
			true,
			[]string{
				"--config",
				filepath.Join(installDir, "test.yaml"),
			},
		)

		// Take a backup; Assert the backup makes sense.
		// It will not be the same as the original service file due to expansion of INSTALLDIR
		// which is OK and expected.
		err = w.Backup()
		require.NoError(t, err)

		backupSvcFile := path.BackupServiceFile(installDir)

		svcCfg, err := readWindowsServiceConfig(backupSvcFile)
		require.NoError(t, err)

		assert.Equal(t, &windowsServiceConfig{
			Path: "windows-service.exe",
			Service: windowsServiceDefinitionConfig{
				Start:       "delayed",
				DisplayName: "Test Windows Service",
				Description: "This is a windows service to test",
				Arguments:   fmt.Sprintf("--config &quot;%s&quot;", filepath.Join(installDir, "test.yaml")),
			},
		}, svcCfg)

	})
}

func TestStartType(t *testing.T) {
	testCases := []struct {
		cfgStartType string
		startType    uint32
		delayed      bool
		expectedErr  string
	}{
		{
			cfgStartType: "auto",
			startType:    mgr.StartAutomatic,
			delayed:      false,
		},
		{
			cfgStartType: "demand",
			startType:    mgr.StartManual,
			delayed:      false,
		},
		{
			cfgStartType: "disabled",
			startType:    mgr.StartDisabled,
			delayed:      false,
		},
		{
			cfgStartType: "delayed",
			startType:    mgr.StartAutomatic,
			delayed:      true,
		},
		{
			cfgStartType: "not-a-real-start-type",
			expectedErr:  "invalid start type in service config",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cfgStartType: %s", tc.cfgStartType), func(t *testing.T) {
			st, d, err := winapiStartType(tc.cfgStartType)
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			} else {
				assert.Equal(t, tc.startType, st)
				assert.Equal(t, tc.delayed, d)
			}
		})
	}
}

func installService(t *testing.T, binPath, serviceName, displayName, description string, startType uint32, delayed bool) {
	t.Helper()

	m, err := mgr.Connect()
	if err != nil {
		require.Fail(t, "failed to connect to service manager", "failed to connect to service manager: %s", err)
	}
	defer m.Disconnect()

	s, err := m.CreateService(serviceName, binPath, mgr.Config{
		DisplayName:      displayName,
		Description:      description,
		StartType:        startType,
		DelayedAutoStart: delayed,
	})
	require.NoError(t, err)
	require.NoError(t, s.Close())
}

// uninstallService is a helper that uninstalls the service manually for test setup, in case it is somehow leftover.
func uninstallService(t *testing.T, serviceName string) {
	m, err := mgr.Connect()
	require.NoError(t, err)
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		// Failed to open the service, we assume it doesn't exist
		t.Logf("failed to open service: %s", err)
		return
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	// If we get an error, the service is likely already in a stopped state.
	if err == nil {
		for status.State != svc.Stopped {
			time.Sleep(100 * time.Millisecond)
			status, err = s.Query()
			require.NoError(t, err)
		}
	} else {
		t.Logf("failed to stop service: %s", err)
	}

	err = s.Delete()
	require.NoError(t, err)

	s.Close()

	const serviceNotExistErrStr = "The specified service does not exist as an installed service."
	for {
		s, err := m.OpenService(serviceName)
		if err != nil {
			if err.Error() == serviceNotExistErrStr {
				// This is expected when the service is uninstalled.
				t.Logf("Service no longer exists: %s", err)
				break
			}
			require.FailNow(t, "failed to uninstall service", "got unexpected error when waiting for service deletion: %s", err)
		}

		if err := s.Close(); err != nil {
			require.FailNow(t, "failed to uninstall service", "got unexpected error when closing service handle: %s", err)
		}
		// rest with the handle closed to let the service manager remove the service
		time.Sleep(50 * time.Millisecond)
	}
}

func requireServiceLoadedStatus(t *testing.T, loaded bool) {
	t.Helper()

	m, err := mgr.Connect()
	require.NoError(t, err, "failed to connect to service manager")
	defer m.Disconnect()

	s, err := m.OpenService("windows-service")
	if err != nil {
		require.False(t, loaded, "Could not connect open service, but service should be loaded")
		return
	}
	defer s.Close()

	require.True(t, loaded, "Connected to open service, but it should not be loaded")

}

func requireServiceConfigMatches(t *testing.T, binaryPath, name string, startType uint32, displayName, description string, delayed bool, args []string) {
	t.Helper()

	m, err := mgr.Connect()
	require.NoError(t, err, "failed to connect to service manager")
	defer m.Disconnect()

	s, err := m.OpenService(name)
	require.NoError(t, err, "failed to open service")
	defer s.Close()

	cfg, err := s.Config()
	require.NoError(t, err)

	expectedBinaryPathName := joinArgs(append([]string{binaryPath}, args...)...)
	assert.Equal(t, displayName, cfg.DisplayName)
	assert.Equal(t, description, cfg.Description)
	assert.Equal(t, delayed, cfg.DelayedAutoStart)
	assert.Equal(t, startType, cfg.StartType)
	assert.Equal(t, expectedBinaryPathName, cfg.BinaryPathName)
	// We always install as LocalSystem, which is the "super user" of the system
	assert.Equal(t, "LocalSystem", cfg.ServiceStartName)

	// Check Recovery Actions are set
	recoveryActions, err := s.RecoveryActions()
	require.NoError(t, err)

	for _, action := range recoveryActions {
		assert.Equal(t, mgr.ServiceRestart, action.Type)
		assert.Equal(t, defaultRecoveryDelay, action.Delay)
	}

	period, err := s.ResetPeriod()
	require.NoError(t, err)
	assert.Equal(t, uint32(defaultResetPeriod.Seconds()), period)
}

func requireServiceRunningStatus(t *testing.T, running bool) {
	t.Helper()

	m, err := mgr.Connect()
	require.NoError(t, err, "failed to connect to service manager")
	defer m.Disconnect()

	s, err := m.OpenService("windows-service")
	require.NoError(t, err, "Failed to open service")
	defer s.Close()

	status, err := s.Query()
	require.NoError(t, err, "Failed to query service state")

	if running {
		require.Contains(t, []svc.State{svc.StartPending, svc.Running}, status.State)
	} else {
		require.Contains(t, []svc.State{svc.StopPending, svc.Stopped}, status.State)
	}
}

func writeServiceFile(t *testing.T, outPath, inPath, serviceGoPath string) {
	t.Helper()

	b, err := os.ReadFile(inPath)
	require.NoError(t, err)

	fileStr := string(b)
	fileStr = os.Expand(fileStr, func(s string) string {
		switch s {
		case "SERVICE_PATH":
			return strings.ReplaceAll(serviceGoPath, `\`, `\\`)
		}
		return ""
	})

	err = os.WriteFile(outPath, []byte(fileStr), 0666)
	require.NoError(t, err)
}

func compileProgram(t *testing.T, inPath, outPath string) {
	t.Helper()

	cmd := exec.Command("go.exe", "build", "-o", outPath, inPath)
	err := cmd.Run()
	require.NoError(t, err)
}

func joinArgs(args ...string) string {
	sb := strings.Builder{}
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			sb.WriteString(`"`)
			sb.WriteString(arg)
			sb.WriteString(`"`)
		} else {
			sb.WriteString(arg)
		}
		sb.WriteString(" ")
	}

	str := sb.String()
	return str[:len(str)-1]
}
