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

package observiq

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

const (
	// CollectorConfigName is the key of the collector config in OpAmp
	CollectorConfigName = "collector.yaml"
	// ManagerConfigName is the key of the manager config in OpAmp
	ManagerConfigName = "manager.yaml"
	// LoggingConfigName is the key of the logging config in OpAmp
	LoggingConfigName = "logging.yaml"
	// ReportConfigName is the key of the report config in OpAmp
	ReportConfigName = "report.yaml"
)

// acceptableConfigs is a lookup of configs that are able to be written/updated
var acceptableConfigs = map[string]struct{}{
	CollectorConfigName: {},
	ManagerConfigName:   {},
	LoggingConfigName:   {},
	ReportConfigName:    {},
}

// Enforce interface
var _ opamp.ConfigManager = (*AgentConfigManager)(nil)

// AgentConfigManager keeps track of active configs for the agent
type AgentConfigManager struct {
	configMap map[string]*opamp.ManagedConfig
	logger    *zap.Logger
}

// NewAgentConfigManager creates a new AgentConfigManager
func NewAgentConfigManager(defaultLogger *zap.Logger) *AgentConfigManager {
	return &AgentConfigManager{
		configMap: make(map[string]*opamp.ManagedConfig),
		logger:    defaultLogger.Named("config manager"),
	}
}

// AddConfig adds a config to be tracked by the config manager.
// If the config already is tracked it'll be overwritten with the new managed config
func (a *AgentConfigManager) AddConfig(configName string, managedConfig *opamp.ManagedConfig) {
	a.configMap[configName] = managedConfig
}

// ComposeEffectiveConfig reads in all config files and calculates the effective config
func (a *AgentConfigManager) ComposeEffectiveConfig() (*protobufs.EffectiveConfig, error) {
	contentMap := make(map[string]*protobufs.AgentConfigFile, len(a.configMap))

	for configName, managedConfig := range a.configMap {
		// Report config is in memory and should not be considered part of the effective config
		if configName == ReportConfigName {
			continue
		}

		// Read in config file
		cleanPath := filepath.Clean(managedConfig.ConfigPath)
		configContents, err := os.ReadFile(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", configName, err)
		}

		// Add to contentMap
		contentMap[configName] = &protobufs.AgentConfigFile{
			Body:        configContents,
			ContentType: opamp.DetermineContentType(managedConfig.ConfigPath),
		}
	}

	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: contentMap,
		},
	}, nil
}

// ApplyConfigChanges compares the remoteConfig to the existing and applies changes
func (a *AgentConfigManager) ApplyConfigChanges(remoteConfig *protobufs.AgentRemoteConfig) (changed bool, returnErr error) {
	remoteConfigMap := remoteConfig.GetConfig().GetConfigMap()

	// No remote config Map
	if remoteConfigMap == nil {
		return
	}

	// loop through all remote configs and compare then with existing configs
	for configName, remoteContents := range remoteConfigMap {
		// For security check the log file we want is acceptable
		if _, ok := acceptableConfigs[configName]; !ok {
			a.logger.Warn("Not supported config received skipping", zap.String("config", configName))
			continue
		}

		managedConfig, ok := a.configMap[configName]
		// This check is impossible to hit now but will be a use case we'll have in the future so leaving this tested code in for now.
		if !ok {
			// We don't current track this config file we should add it
			if err := a.trackNewConfig(configName, remoteContents.GetBody()); err != nil {
				returnErr = err
				return
			}
			changed = true
			continue
		}

		var configChanged bool
		var err error
		switch configName {
		// Special case for report config as it's an in memory only config and we always want to apply it
		case ReportConfigName:
			configChanged, err = managedConfig.Reload(remoteContents.GetBody())
		default:
			// Update the config file
			configChanged, err = a.updateExistingConfig(configName, managedConfig, remoteContents.GetBody())
		}

		if err != nil {
			returnErr = err
			return
		}
		changed = changed || configChanged
	}

	return
}

func (a *AgentConfigManager) updateExistingConfig(configName string, managedConfig *opamp.ManagedConfig, newContents []byte) (changed bool, err error) {
	remoteHash := opamp.ComputeHash(newContents)

	// Nothing to update
	if bytes.Equal(managedConfig.GetCurrentConfigHash(), remoteHash) {
		// Verify disk matches memory and remote. This will ensure if any changes were made to the config on disk
		// the will be overwritten by what is expected.
		return verifyDiskContents(managedConfig.ConfigPath, managedConfig.GetCurrentConfigHash(), newContents)
	}

	a.logger.Info("Applying changes to config file",
		zap.String("OnDiskCollectorConfig", managedConfig.ConfigPath))
	changed, err = managedConfig.Reload(newContents)
	if err != nil {
		err = fmt.Errorf("failed to reload config: %s: %w", configName, err)
		return
	}

	// If the config changed recompute the hash for it
	if changed {
		err = managedConfig.ComputeConfigHash()
		if err != nil {
			err = fmt.Errorf("failed hash compute for config %s: %w", configName, err)
			return
		}
	}

	return
}

// verifyDiskContents verifies the contents saved on disk match the in memory hash.
// If not overwrite them with the passed in contents
func verifyDiskContents(configPath string, memHash, contents []byte) (changed bool, err error) {
	cleanPath := filepath.Clean(configPath)
	currContents, err := os.ReadFile(cleanPath)
	if err != nil {
		return false, fmt.Errorf("error reading current config contents: %w", err)
	}

	diskHash := opamp.ComputeHash(currContents)

	// Disk equals what's in memory and what's expected so do nothing
	if bytes.Equal(diskHash, memHash) {
		return false, nil
	}

	// Disk file doesn't match memory so overwrite with correct contents
	if err := os.WriteFile(cleanPath, contents, 0600); err != nil {
		return false, fmt.Errorf("failed to write contents to config file: %w", err)
	}

	return true, nil
}

func (a *AgentConfigManager) trackNewConfig(configName string, contents []byte) error {
	a.logger.Info("Untracked config found", zap.String("config", configName))

	// Write out the file
	if err := os.WriteFile(configName, contents, 0600); err != nil {
		return fmt.Errorf("failed to write new config file %s: %w", configName, err)
	}

	managedConfig, err := opamp.NewManagedConfig(filepath.Join(".", configName), opamp.NoopReloadFunc, false)
	if err != nil {
		return err
	}

	// Track new config
	a.AddConfig(configName, managedConfig)

	return nil
}
