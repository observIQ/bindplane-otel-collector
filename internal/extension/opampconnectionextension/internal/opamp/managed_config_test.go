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

package opamp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedConfigGetCurrentConfigHash(t *testing.T) {
	managedConfig := &ManagedConfig{
		ConfigPath:        "./path.yaml",
		Reload:            NoopReloadFunc,
		currentConfigHash: []byte("hello world"),
	}

	actual := managedConfig.GetCurrentConfigHash()
	require.Equal(t, managedConfig.currentConfigHash, actual)
}

func TestManagedConfigComputeConfigHash(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Missing config file, for required",
			testFunc: func(t *testing.T) {
				managedConfig := &ManagedConfig{
					ConfigPath: "./path.yaml",
					required:   true,
				}

				err := managedConfig.ComputeConfigHash()
				assert.Error(t, err)
			},
		},
		{
			desc: "Missing config file, for not required",
			testFunc: func(t *testing.T) {
				managedConfig := &ManagedConfig{
					ConfigPath: "./path.yaml",
				}

				err := managedConfig.ComputeConfigHash()
				assert.NoError(t, err)
			},
		},
		{
			desc: "Successful hash",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				cfgPath := filepath.Join(tmpDir, "config.yaml")
				err := os.WriteFile(cfgPath, []byte("hello world"), 0600)
				assert.NoError(t, err)

				managedConfig := &ManagedConfig{
					ConfigPath: cfgPath,
					required:   true,
				}

				expected := []byte{0xb9, 0x4d, 0x27, 0xb9, 0x93, 0x4d, 0x3e, 0x8, 0xa5, 0x2e, 0x52, 0xd7, 0xda, 0x7d, 0xab, 0xfa, 0xc4, 0x84, 0xef, 0xe3, 0x7a, 0x53, 0x80, 0xee, 0x90, 0x88, 0xf7, 0xac, 0xe2, 0xef, 0xcd, 0xe9}

				err = managedConfig.ComputeConfigHash()
				assert.NoError(t, err)
				assert.Equal(t, expected, managedConfig.currentConfigHash)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestNewManagedConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Missing config file",
			testFunc: func(t *testing.T) {
				managedConfig, err := NewManagedConfig("./path.yml", NoopReloadFunc, true)
				assert.Error(t, err)
				assert.Nil(t, managedConfig)
			},
		},
		{
			desc: "Success",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				cfgPath := filepath.Join(tmpDir, "config.yaml")
				err := os.WriteFile(cfgPath, []byte("hello world"), 0600)
				assert.NoError(t, err)

				expected := &ManagedConfig{
					ConfigPath:        cfgPath,
					Reload:            NoopReloadFunc,
					currentConfigHash: []byte{0xb9, 0x4d, 0x27, 0xb9, 0x93, 0x4d, 0x3e, 0x8, 0xa5, 0x2e, 0x52, 0xd7, 0xda, 0x7d, 0xab, 0xfa, 0xc4, 0x84, 0xef, 0xe3, 0x7a, 0x53, 0x80, 0xee, 0x90, 0x88, 0xf7, 0xac, 0xe2, 0xef, 0xcd, 0xe9},
				}

				managedConfig, err := NewManagedConfig(cfgPath, NoopReloadFunc, true)
				assert.NoError(t, err)
				assert.Equal(t, expected.ConfigPath, managedConfig.ConfigPath)
				assert.Equal(t, expected.currentConfigHash, managedConfig.currentConfigHash)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
