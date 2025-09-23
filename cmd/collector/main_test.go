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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Must is a helper function for tests that panics if there is an error creating the object of type T
func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func TestGetDefaultCollectorConfigPathENV(t *testing.T) {
	fakeConfigPath := "./fake/path/config.yaml"

	t.Setenv(configPathENV, fakeConfigPath)

	expected := []string{fakeConfigPath}
	actual := getDefaultCollectorConfigPaths()

	require.Equal(t, expected, actual)
}

func TestGetDefaultCollectorConfigPathNone(t *testing.T) {
	expected := []string{"./config.yaml"}
	actual := getDefaultCollectorConfigPaths()

	require.Equal(t, expected, actual)
}

func TestGetDefaultLoggingConfigPathENV(t *testing.T) {
	fakeLoggingPath := "./fake/path/logging.yaml"

	t.Setenv(loggingPathENV, fakeLoggingPath)

	expected := fakeLoggingPath
	actual := getDefaultLoggingConfigPath()

	require.Equal(t, expected, actual)
}

func TestGetDefaultLoggingConfigPathNone(t *testing.T) {
	expected := "./logging.yaml"
	actual := getDefaultLoggingConfigPath()

	require.Equal(t, expected, actual)
}

func TestGetDefaultManagerConfigPathENV(t *testing.T) {
	fakeManagerPath := "./fake/path/Manager.yaml"

	t.Setenv(managerPathENV, fakeManagerPath)

	expected := fakeManagerPath
	actual := getDefaultManagerConfigPath()

	require.Equal(t, expected, actual)
}

func TestGetDefaultManagerConfigPathNone(t *testing.T) {
	expected := "./manager.yaml"
	actual := getDefaultManagerConfigPath()

	require.Equal(t, expected, actual)
}

func TestCheckManagerNoConfig(t *testing.T) {
	manager := "./manager.yaml"
	err := checkManagerConfig(&manager)
	require.ErrorIs(t, err, os.ErrNotExist)

	tmp := "\000"
	err = checkManagerConfig(&tmp)
	require.Error(t, err)
}

func TestCheckManagerConfigNoFile(t *testing.T) {
	t.Setenv(endpointENV, "0.0.0.0")

	t.Setenv(agentNameENV, "agent name")

	t.Setenv(agentIDENV, "01HX2DWEQZ045KQR3VG0EYEZ94")

	t.Setenv(secretkeyENV, "secretKey")

	t.Setenv(labelsENV, "this is a label")

	tmpdir := t.TempDir()
	manager := filepath.Join(tmpdir, "manager.yaml")
	err := checkManagerConfig(&manager)
	require.NoError(t, err)

	actual, _ := opamp.ParseConfig(manager)
	expected := &opamp.Config{
		Endpoint:  "0.0.0.0",
		AgentID:   Must(opamp.ParseAgentID("01HX2DWEQZ045KQR3VG0EYEZ94")),
		AgentName: new(string),
		SecretKey: new(string),
		Labels:    new(string),
		// InstanceID is set internally by the client, so we must copy the actual value to the expected value config
		InstanceID: actual.InstanceID,
		ExtraMeasurementsAttributes: map[string]string{
			"bindplane_instance_id": actual.InstanceID,
		},
	}
	*expected.AgentName = "agent name"
	*expected.SecretKey = "secretKey"
	*expected.Labels = "this is a label"

	require.Equal(t, expected, actual)
}

func TestCheckManagerConfigNoFileTLS(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func()
		expectFunc  func() *opamp.TLSConfig
		expectedErr error
	}{
		{
			name: "no-tls",
			setupFunc: func() {
				return
			},
			expectFunc: func() *opamp.TLSConfig {
				return nil
			},
			expectedErr: nil,
		},
		{
			name: "skip-verify",
			setupFunc: func() {
				t.Setenv(tlsSkipVerifyENV, "true")
			},
			expectFunc: func() *opamp.TLSConfig {
				return &opamp.TLSConfig{
					InsecureSkipVerify: true,
				}
			},
			expectedErr: nil,
		},
		{
			name: "skip-verify_invalid",
			setupFunc: func() {
				t.Setenv(tlsSkipVerifyENV, "invalid")
			},
			expectFunc: func() *opamp.TLSConfig {
				return nil
			},
			expectedErr: errors.New("invalid value 'invalid' for environment option 'OPAMP_TLS_SKIP_VERIFY'"),
		},
		{
			name: "tls",
			setupFunc: func() {
				t.Setenv(tlsCaENV, "/tls/ca.crt")
			},
			expectFunc: func() *opamp.TLSConfig {
				ca := "/tls/ca.crt"
				return &opamp.TLSConfig{
					CAFile: &ca,
				}
			},
			expectedErr: nil,
		},
		{
			name: "mtls",
			setupFunc: func() {
				t.Setenv(tlsKeyENV, "/tls/tls.key")
				t.Setenv(tlsCertENV, "/tls/tls.crt")
			},
			expectFunc: func() *opamp.TLSConfig {
				key := "/tls/tls.key"
				cert := "/tls/tls.crt"
				return &opamp.TLSConfig{
					KeyFile:  &key,
					CertFile: &cert,
				}
			},
			expectedErr: nil,
		},
		{
			name: "tls_all",
			setupFunc: func() {
				t.Setenv(tlsSkipVerifyENV, "true")
				t.Setenv(tlsCaENV, "/tls/ca.crt")
				t.Setenv(tlsKeyENV, "/tls/tls.key")
				t.Setenv(tlsCertENV, "/tls/tls.crt")
			},
			expectFunc: func() *opamp.TLSConfig {
				ca := "/tls/ca.crt"
				key := "/tls/tls.key"
				cert := "/tls/tls.crt"
				return &opamp.TLSConfig{
					InsecureSkipVerify: true,
					CAFile:             &ca,
					KeyFile:            &key,
					CertFile:           &cert,
				}
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupFunc()
			defer func() {
				os.Unsetenv(tlsSkipVerifyENV)
				os.Unsetenv(tlsCaENV)
				os.Unsetenv(tlsKeyENV)
				os.Unsetenv(tlsCertENV)
			}()

			actual, err := configureTLS()
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				return
			}
			require.Equal(t, tc.expectFunc(), actual)
		})
	}
}

func TestManagerConfigNoAgentIDWillSet(t *testing.T) {
	tmpDir := t.TempDir()
	manager := filepath.Join(tmpDir, "manager.yaml")
	data := []byte("")
	require.NoError(t, os.WriteFile(manager, data, 0600))
	err := checkManagerConfig(&manager)
	require.NoError(t, err)

	cfgBytes, err := os.ReadFile(manager)
	require.NoError(t, err)

	var config opamp.Config
	require.NoError(t, yaml.Unmarshal(cfgBytes, &config))
	require.NotEqual(t, opamp.EmptyAgentID, config.AgentID)
}

// TestManagerConfigWillNotOverwriteCurrentAgentID tests that if the agent ID is a ULID it will not overwrite it
func TestManagerConfigWillNotOverwriteCurrentAgentID(t *testing.T) {
	tmpDir := t.TempDir()
	manager := filepath.Join(tmpDir, "manager.yaml")

	id := ulid.Make()
	data := []byte(fmt.Sprintf(`
---
agent_id: %s
`, id))
	require.NoError(t, os.WriteFile(manager, data, 0600))
	err := checkManagerConfig(&manager)
	require.NoError(t, err)

	cfgBytes, err := os.ReadFile(manager)
	require.NoError(t, err)

	var config opamp.Config
	require.NoError(t, yaml.Unmarshal(cfgBytes, &config))
	require.Equal(t, config.AgentID.String(), id.String())
}

// TestManagerConfigWillUpdateLegacyAgentID tests that if the agent ID is a Legacy ID (UUID format) it will overwrite with a new ULID
func TestManagerConfigWillUpdateLegacyAgentID(t *testing.T) {
	tmpDir := t.TempDir()
	manager := filepath.Join(tmpDir, "manager.yaml")

	legacyID := uuid.NewString()
	data := []byte(fmt.Sprintf(`
---
agent_id: %s
`, legacyID))
	require.NoError(t, os.WriteFile(manager, data, 0600))
	err := checkManagerConfig(&manager)
	require.NoError(t, err)

	cfgBytes, err := os.ReadFile(manager)
	require.NoError(t, err)

	var config opamp.Config
	require.NoError(t, yaml.Unmarshal(cfgBytes, &config))
	require.NotEqual(t, opamp.EmptyAgentID, config.AgentID)
}

func TestManagerConfigWillErrorOnInvalidOpAmpConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manager := filepath.Join(tmpDir, "manager.yaml")
	data := []byte(`
---
agent_id:
  - some-kind-of-array
  - should-blow-up
`)
	require.NoError(t, os.WriteFile(manager, data, 0600))
	err := checkManagerConfig(&manager)
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to interpret config file")
}

func TestManagerConfigCheckFileModes(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		fileMode    os.FileMode
		expectedErr error
	}{
		{
			name:        "read_only",
			fileMode:    0400,
			expectedErr: errors.New("failed to rewrite manager config with identifying fields"),
		},
		{
			name:     "valid_read_write",
			fileMode: 0600,
		},
	}

	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := filepath.Join(tmpDir, fmt.Sprintf("manager-%d.yaml", idx))
			require.NoError(t, os.WriteFile(manager, []byte(""), tc.fileMode))
			err := checkManagerConfig(&manager)
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_checkForCollectorRollbackConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "No Rollback file",
			testFunc: func(t *testing.T) {
				originalContents := []byte("origin config contents")
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "collector.yaml")
				err := os.WriteFile(configPath, originalContents, 0600)
				require.NoError(t, err)

				err = checkForCollectorRollbackConfig(configPath)
				require.NoError(t, err)

				currentConfigContents, err := os.ReadFile(configPath)
				require.NoError(t, err)
				require.Equal(t, originalContents, currentConfigContents)
			},
		},
		{
			desc: "Can't read rollback file",
			testFunc: func(t *testing.T) {
				// Skip windows test case as file permissions don't work the same on windows
				if runtime.GOOS == "windows" {
					t.Log("Skipping test case on windows")
					return
				}

				originalContents := []byte("origin config contents")
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "collector.yaml")
				err := os.WriteFile(configPath, originalContents, 0600)
				require.NoError(t, err)

				rollbackPath := fmt.Sprintf("%s.rollback", configPath)
				rollbackFile, err := os.OpenFile(rollbackPath, os.O_CREATE|os.O_RDWR, 0220)
				require.NoError(t, err)
				require.NoError(t, rollbackFile.Close())

				err = checkForCollectorRollbackConfig(configPath)
				require.ErrorContains(t, err, "error while reading in collector rollback file")

				currentConfigContents, err := os.ReadFile(configPath)
				require.NoError(t, err)
				require.Equal(t, originalContents, currentConfigContents)
			},
		},
		{
			desc: "Can't write rollback contents to config",
			testFunc: func(t *testing.T) {
				// Skip windows test case as file permissions don't work the same on windows
				if runtime.GOOS == "windows" {
					t.Log("Skipping test case on windows")
					return
				}

				originalContents := []byte("origin config contents")
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "collector.yaml")
				err := os.WriteFile(configPath, originalContents, 0400)
				require.NoError(t, err)

				rollbackPath := fmt.Sprintf("%s.rollback", configPath)
				err = os.WriteFile(rollbackPath, []byte("rollback"), 0600)
				require.NoError(t, err)

				err = checkForCollectorRollbackConfig(configPath)
				require.ErrorContains(t, err, "error while writing rollback contents onto config")

				currentConfigContents, err := os.ReadFile(configPath)
				require.NoError(t, err)
				require.Equal(t, originalContents, currentConfigContents)
			},
		},
		{
			desc: "Successful Rollback copy",
			testFunc: func(t *testing.T) {
				originalContents := []byte("origin config contents")
				rollbackContents := []byte("rollback contents")
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "collector.yaml")
				err := os.WriteFile(configPath, originalContents, 0600)
				require.NoError(t, err)

				rollbackPath := fmt.Sprintf("%s.rollback", configPath)
				err = os.WriteFile(rollbackPath, rollbackContents, 0600)
				require.NoError(t, err)

				err = checkForCollectorRollbackConfig(configPath)
				require.NoError(t, err)

				currentConfigContents, err := os.ReadFile(configPath)
				require.NoError(t, err)
				require.Equal(t, rollbackContents, currentConfigContents)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
