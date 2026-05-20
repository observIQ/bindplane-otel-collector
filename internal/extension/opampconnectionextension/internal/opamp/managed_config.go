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
	"fmt"
	"os"
	"path/filepath"
)

// ReloadFunc is a function that handles reloading a config given the new contents
// Reload function should return true for changed is the in memory or on disk copy of the config
// was changed in any way. If neither was altered the changed return value should be false.
type ReloadFunc func([]byte) (changed bool, err error)

// NoopReloadFunc used as a noop reload function if unsure of how to reload
func NoopReloadFunc([]byte) (bool, error) {
	return false, nil
}

// NewManagedConfig creates a new Managed config and computes its hash
func NewManagedConfig(configPath string, reload ReloadFunc, required bool) (*ManagedConfig, error) {
	managedConfig := &ManagedConfig{
		ConfigPath: configPath,
		Reload:     reload,
		required:   required,
	}

	if err := managedConfig.ComputeConfigHash(); err != nil {
		return nil, fmt.Errorf("failed to compute hash for config %w", err)
	}

	return managedConfig, nil
}

// ManagedConfig is a structure that can manage an on disk config file
type ManagedConfig struct {
	// ConfigPath is the path on disk where the configuration lives
	ConfigPath string

	// Reload will be called when any changes to this config occur.
	Reload ReloadFunc

	// currentConfigHash is the hash of the config currently being used
	currentConfigHash []byte

	// required signals if this config is required for operation
	// If false no error will be returned if this file is not found when computing hashes
	required bool
}

// GetCurrentConfigHash retrieves the current config hash
func (m *ManagedConfig) GetCurrentConfigHash() []byte {
	return m.currentConfigHash
}

// ComputeConfigHash reads in the config file, computes the hash for the contents, and saves it on the ManagedConfig.
func (m *ManagedConfig) ComputeConfigHash() error {
	cleanPath := filepath.Clean(m.ConfigPath)
	contents, err := os.ReadFile(cleanPath)
	if err != nil {
		// If file is not required and we couldn't read it return an error
		if m.required {
			return err
		}

		// File not required set contents to empty slice so a hash can still be computed
		contents = []byte{}
	}

	m.currentConfigHash = ComputeHash(contents)
	return nil
}
