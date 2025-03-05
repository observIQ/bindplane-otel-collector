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

package observiq

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewAgentConfigManager(t *testing.T) {
	logger := zap.NewNop()

	expected := &AgentConfigManager{
		configMap: make(map[string]*opamp.ManagedConfig),
		logger:    logger.Named("config manager"),
	}

	actual := NewAgentConfigManager(logger)
	require.Equal(t, expected, actual)
}

func TestAddConfig(t *testing.T) {
	manager := NewAgentConfigManager(zap.NewNop())

	configName := "config.json"
	cfgPath := "path/to/config.json"
	managedConfig := &opamp.ManagedConfig{
		ConfigPath: cfgPath,
		Reload:     opamp.NoopReloadFunc,
	}

	manager.AddConfig(configName, managedConfig)
	require.Equal(t, managedConfig, manager.configMap[configName])
}

func TestComposeEffectiveConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "File missing from disk error",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				manager := NewAgentConfigManager(zap.NewNop())
				managedCfg := &opamp.ManagedConfig{
					ConfigPath: filepath.Join(tmpDir, "not_real.yaml"),
					Reload:     opamp.NoopReloadFunc,
				}
				manager.AddConfig("not_real.yaml", managedCfg)

				effCfg, err := manager.ComposeEffectiveConfig()
				assert.ErrorContains(t, err, "error reading config file")
				assert.Nil(t, effCfg)
			},
		},
		{
			desc: "Multi Config Files",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configOne := "one.yaml"
				configOnePath := filepath.Join(tmpDir, configOne)
				configOneContents := []byte(`key: value`)

				configTwo := "two.yaml"
				configTwoPath := filepath.Join(tmpDir, configTwo)
				configTwoContents := []byte(`key2: value2`)

				err := os.WriteFile(configOnePath, configOneContents, 0600)
				assert.NoError(t, err)

				err = os.WriteFile(configTwoPath, configTwoContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				manager.AddConfig(configOne, &opamp.ManagedConfig{
					ConfigPath: configOnePath,
					Reload:     opamp.NoopReloadFunc,
				})
				manager.AddConfig(configTwo, &opamp.ManagedConfig{
					ConfigPath: configTwoPath,
					Reload:     opamp.NoopReloadFunc,
				})

				expected := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							configOne: {
								Body:        configOneContents,
								ContentType: opamp.YAMLContentType,
							},
							configTwo: {
								Body:        configTwoContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expected, effCfg)

			},
		},
		{
			desc: "Multi Config Files, report.yaml is skipped",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configOne := "one.yaml"
				configOnePath := filepath.Join(tmpDir, configOne)
				configOneContents := []byte(`key: value`)

				configTwo := ReportConfigName

				err := os.WriteFile(configOnePath, configOneContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				manager.AddConfig(configOne, &opamp.ManagedConfig{
					ConfigPath: configOnePath,
					Reload:     opamp.NoopReloadFunc,
				})
				manager.AddConfig(configTwo, &opamp.ManagedConfig{
					ConfigPath: "report.yaml",
					Reload:     opamp.NoopReloadFunc,
				})

				expected := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							configOne: {
								Body:        configOneContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expected, effCfg)

			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestApplyConfigChanges(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "No remote config To Apply",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ManagerConfigName)
				configContents := []byte(`key: value`)

				err := os.WriteFile(configPath, configContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				mangedConfig, err := opamp.NewManagedConfig(configPath, opamp.NoopReloadFunc, true)
				assert.NoError(t, err)
				manager.AddConfig(ManagerConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{},
				}

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}
				changed, err := manager.ApplyConfigChanges(remoteConfig)
				assert.NoError(t, err)
				assert.False(t, changed)

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)
			},
		},
		{
			desc: "Remote config contains unchanged file",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ManagerConfigName)
				configContents := []byte(`key: value`)

				err := os.WriteFile(configPath, configContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				mangedConfig, err := opamp.NewManagedConfig(configPath, opamp.NoopReloadFunc, true)
				assert.NoError(t, err)
				manager.AddConfig(ManagerConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}
				changed, err := manager.ApplyConfigChanges(remoteConfig)

				assert.NoError(t, err)
				assert.False(t, changed)

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)
			},
		},
		{
			desc: "Remote config contains unchanged file, but disk differs",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ManagerConfigName)
				configContents := []byte(`key: value`)

				err := os.WriteFile(configPath, configContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				mangedConfig, err := opamp.NewManagedConfig(configPath, opamp.NoopReloadFunc, true)
				assert.NoError(t, err)
				manager.AddConfig(ManagerConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				// change contents of on disk so they don't match memory
				err = os.WriteFile(configPath, []byte("bad: config"), 0600)
				assert.NoError(t, err)

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				changed, err := manager.ApplyConfigChanges(remoteConfig)
				assert.NoError(t, err)
				assert.True(t, changed)

				// Verify on disk matches what is expected
				diskContents, err := os.ReadFile(configPath)
				assert.NoError(t, err)
				assert.Equal(t, configContents, diskContents)

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)
			},
		},
		{
			desc: "Remote config contains unknown file",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ManagerConfigName)
				configContents := []byte(`key: value`)

				err := os.WriteFile(configPath, configContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())

				mangedConfig, err := opamp.NewManagedConfig(configPath, opamp.NoopReloadFunc, true)
				assert.NoError(t, err)
				manager.AddConfig(ManagerConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							"other.yaml": {
								Body:        []byte("other: value"),
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}
				changed, err := manager.ApplyConfigChanges(remoteConfig)
				assert.NoError(t, err)
				assert.False(t, changed)

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)
			},
		},
		{
			desc: "Remote config contains untracked known file",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ManagerConfigName)
				configContents := []byte(`key: value`)

				newFileContents := []byte(`logger: value`)

				err := os.WriteFile(configPath, configContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				mangedConfig, err := opamp.NewManagedConfig(configPath, opamp.NoopReloadFunc, true)
				assert.NoError(t, err)
				manager.AddConfig(ManagerConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
							LoggingConfigName: {
								Body:        newFileContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ManagerConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
							LoggingConfigName: {
								Body:        newFileContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}
				changed, err := manager.ApplyConfigChanges(remoteConfig)

				assert.NoError(t, err)
				assert.True(t, changed)
				assert.FileExists(t, filepath.Join(".", LoggingConfigName))

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)

				// Cleanup
				err = os.Remove(filepath.Join(".", LoggingConfigName))
				assert.NoError(t, err)
			},
		},
		{
			desc: "Remote config is report.yaml",
			testFunc: func(*testing.T) {
				newFileContents := []byte(`logger: value`)

				manager := NewAgentConfigManager(zap.NewNop())
				mangedConfig, err := opamp.NewManagedConfig("report.yml", opamp.NoopReloadFunc, false)
				assert.NoError(t, err)
				manager.AddConfig(ReportConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							ReportConfigName: {
								Body:        newFileContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{},
					},
				}
				changed, err := manager.ApplyConfigChanges(remoteConfig)
				assert.NoError(t, err)
				assert.False(t, changed)

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)
			},
		},
		{
			desc: "Remote config contains changes to file",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, LoggingConfigName)
				configContents := []byte(`key: value`)

				newFileContents := []byte(`logger: value`)

				err := os.WriteFile(configPath, configContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				mangedConfig, err := opamp.NewManagedConfig(configPath, func(data []byte) (changed bool, err error) {
					err = os.WriteFile(configPath, data, 0600)
					assert.NoError(t, err)
					return true, err
				}, true)
				assert.NoError(t, err)
				manager.AddConfig(LoggingConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							LoggingConfigName: {
								Body:        newFileContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							LoggingConfigName: {
								Body:        newFileContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}
				changed, err := manager.ApplyConfigChanges(remoteConfig)
				assert.NoError(t, err)
				assert.True(t, changed)

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)
			},
		},
		{
			desc: "Remote config contains changes to file, reload fails",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, LoggingConfigName)
				configContents := []byte(`key: value`)

				newFileContents := []byte(`logger: value`)

				expectedError := errors.New("oops")

				err := os.WriteFile(configPath, configContents, 0600)
				assert.NoError(t, err)

				manager := NewAgentConfigManager(zap.NewNop())
				mangedConfig, err := opamp.NewManagedConfig(configPath, func(_ []byte) (changed bool, err error) {
					return false, expectedError
				}, true)
				assert.NoError(t, err)
				manager.AddConfig(LoggingConfigName, mangedConfig)

				remoteConfig := &protobufs.AgentRemoteConfig{
					Config: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							LoggingConfigName: {
								Body:        newFileContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}

				expectedEffCfg := &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							LoggingConfigName: {
								Body:        configContents,
								ContentType: opamp.YAMLContentType,
							},
						},
					},
				}
				changed, err := manager.ApplyConfigChanges(remoteConfig)
				assert.ErrorIs(t, err, expectedError)
				assert.False(t, changed)

				// Verify effective config is as expected
				effCfg, err := manager.ComposeEffectiveConfig()
				assert.NoError(t, err)
				assert.Equal(t, expectedEffCfg, effCfg)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
