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
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/observiq/bindplane-otel-collector/internal/logging"
	"github.com/observiq/bindplane-otel-collector/internal/report"
	"github.com/observiq/bindplane-otel-collector/opamp"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func managerReload(client *Client, managerConfigPath string) opamp.ReloadFunc {
	return func(contents []byte) (bool, error) {
		// Unmarshal config and only pull fields out that are allowed to be updated.
		var newConfig opamp.Config
		if err := yaml.Unmarshal(contents, &newConfig); err != nil {
			return false, fmt.Errorf("failed to validate config %s", ManagerConfigName)
		}

		// Check if the updatable fields are equal
		// If so then exit
		if client.currentConfig.CmpUpdatableFields(newConfig) {
			return false, nil
		}

		updatedKeys := []string{}
		if !opamp.CmpStringPtr(client.currentConfig.AgentName, newConfig.AgentName) {
			updatedKeys = append(updatedKeys, "agent_name")
		}

		if client.currentConfig.MeasurementsInterval != newConfig.MeasurementsInterval {
			updatedKeys = append(updatedKeys, "measurements_interval")
		}

		if !maps.Equal(client.currentConfig.ExtraMeasurementsAttributes, newConfig.ExtraMeasurementsAttributes) {
			updatedKeys = append(updatedKeys, "extra_measurements_attributes")
		}

		if !opamp.CmpDurationPtr(client.currentConfig.TopologyInterval, newConfig.TopologyInterval) {
			updatedKeys = append(updatedKeys, "topology_interval")
		}

		client.logger.Info("Manager config update detected", zap.Strings("updated_keys", updatedKeys))
		// Going to do an update prep a rollback
		rollbackFunc, cleanupFunc, err := prepRollback(managerConfigPath)
		if err != nil {
			return false, fmt.Errorf("failed to prep for rollback: %w", err)
		}

		defer func() {
			// Cleanup rollback
			if err := cleanupFunc(); err != nil {
				client.logger.Warn("Failed to cleanup rollback file", zap.Error(err))
			}
		}()

		//create a copies for rollback
		rollBackCfg := client.currentConfig.Copy()
		rollbackIdent := client.ident.Copy()

		// Updatable config fields
		client.currentConfig.AgentName = newConfig.AgentName
		client.currentConfig.Labels = newConfig.Labels
		client.currentConfig.MeasurementsInterval = newConfig.MeasurementsInterval
		client.currentConfig.ExtraMeasurementsAttributes = newConfig.ExtraMeasurementsAttributes
		client.currentConfig.TopologyInterval = newConfig.TopologyInterval

		// Update identity
		client.ident.agentName = newConfig.AgentName
		client.ident.labels = newConfig.Labels

		// Read the raw on-disk file (preserves ${env:...} references)
		rawContents, err := os.ReadFile(filepath.Clean(managerConfigPath))
		if err != nil {
			// Rollback file
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for manager config", zap.Error(rollbackErr))
			}
			client.ident = rollbackIdent
			client.currentConfig = *rollBackCfg
			return false, fmt.Errorf("failed to read manager config for update: %w", err)
		}

		// Unmarshal into a generic map (yaml.v3 does NOT resolve env vars)
		var rawMap map[string]any
		if err := yaml.Unmarshal(rawContents, &rawMap); err != nil {
			// Rollback file
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for manager config", zap.Error(rollbackErr))
			}
			client.ident = rollbackIdent
			client.currentConfig = *rollBackCfg
			return false, fmt.Errorf("failed to parse manager config for update: %w", err)
		}

		// Update only the fields that are allowed to change via OpAMP
		if client.currentConfig.AgentName != nil {
			rawMap["agent_name"] = *client.currentConfig.AgentName
		} else {
			delete(rawMap, "agent_name")
		}
		if client.currentConfig.Labels != nil {
			rawMap["labels"] = *client.currentConfig.Labels
		} else {
			delete(rawMap, "labels")
		}
		if client.currentConfig.MeasurementsInterval != 0 {
			rawMap["measurements_interval"] = client.currentConfig.MeasurementsInterval
		} else {
			delete(rawMap, "measurements_interval")
		}
		if client.currentConfig.ExtraMeasurementsAttributes != nil {
			rawMap["extra_measurements_attributes"] = client.currentConfig.ExtraMeasurementsAttributes
		} else {
			delete(rawMap, "extra_measurements_attributes")
		}
		if client.currentConfig.TopologyInterval != nil {
			rawMap["topology_interval"] = *client.currentConfig.TopologyInterval
		} else {
			delete(rawMap, "topology_interval")
		}

		// Marshal the map back (env var references in untouched fields are preserved)
		newContents, err := yaml.Marshal(rawMap)
		if err != nil {
			// Rollback file
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for manager config", zap.Error(rollbackErr))
			}
			client.ident = rollbackIdent
			client.currentConfig = *rollBackCfg
			return false, fmt.Errorf("failed to reformat manager config: %w", err)
		}

		// Save config file to disk
		if err := updateConfigFile(ManagerConfigName, managerConfigPath, newContents); err != nil {
			// Rollback file
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for collector config", zap.Error(rollbackErr))
			}
			client.ident = rollbackIdent
			client.currentConfig = *rollBackCfg
			return false, err
		}

		// Set the agent description
		if err := client.opampClient.SetAgentDescription(client.ident.ToAgentDescription()); err != nil {
			// Rollback file
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for collector config", zap.Error(rollbackErr))
			}
			client.ident = rollbackIdent
			client.currentConfig = *rollBackCfg
			return false, fmt.Errorf("failed to set agent description: %w ", err)
		}

		// Set new measurements interval and attributes
		client.measurementsSender.SetInterval(client.currentConfig.MeasurementsInterval)
		client.measurementsSender.SetExtraAttributes(client.currentConfig.ExtraMeasurementsAttributes)
		client.topologySender.SetInterval(client.currentConfig.TopologyInterval)

		return true, nil
	}
}

func collectorReload(client *Client, collectorConfigPath string) opamp.ReloadFunc {
	return func(contents []byte) (bool, error) {
		rollbackFunc, cleanupFunc, err := prepRollback(collectorConfigPath)
		client.logger.Info("Rollback prepped", zap.String("collectorConfigPath", collectorConfigPath))
		if err != nil {
			return false, fmt.Errorf("failed to prep for rollback: %w", err)
		}

		defer func() {
			// Cleanup rollback
			if err := cleanupFunc(); err != nil {
				client.logger.Warn("Failed to cleanup rollback file", zap.Error(err))
			}
		}()

		// Write new config file
		if err := updateConfigFile(CollectorConfigName, collectorConfigPath, contents); err != nil {
			return false, err
		}
		client.logger.Info("Config file updated", zap.String("name", CollectorConfigName), zap.String("collectorConfigPath", collectorConfigPath))
		// Stop collector monitoring as we are going to restart it
		client.stopCollectorMonitoring()

		client.logger.Info("Collector monitoring stopped")
		// Setup new monitoring after collector has been restarted
		defer client.startCollectorMonitoring(context.Background())

		// Reload collector
		if err := client.collector.Restart(context.Background()); err != nil {
			client.logger.Info("OTEL Collector restart error", zap.Error(err))

			// Rollback file
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for collector config", zap.Error(rollbackErr))
			}

			// Restart collector with original file
			if rollbackErr := client.collector.Restart(context.Background()); rollbackErr != nil {
				client.logger.Error("Collector failed for restart during rollback", zap.Error(rollbackErr))
			}

			return false, fmt.Errorf("collector failed to restart: %w", err)
		}
		client.logger.Info("OTEL Collector restarted")
		// Reset Snapshot Reporter
		report.GetSnapshotReporter().Reset()
		client.logger.Info("Snapshot reporter reset")

		return true, nil
	}
}

func reportReload(client *Client) opamp.ReloadFunc {
	return func(contents []byte) (bool, error) {
		if err := client.reportManager.ResetConfig(contents); err != nil {
			client.logger.Error("Failure in applying report config", zap.Error(err))
			return false, fmt.Errorf("failed to apply report config: %w", err)
		}

		return true, nil
	}
}

func loggerReload(client *Client, loggerConfigPath string) opamp.ReloadFunc {
	return func(contents []byte) (bool, error) {
		rollbackFunc, cleanupFunc, err := prepRollback(loggerConfigPath)
		if err != nil {
			return false, fmt.Errorf("failed to prep for rollback: %w", err)
		}

		defer func() {
			// Cleanup rollback
			if err := cleanupFunc(); err != nil {
				client.logger.Warn("Failed to cleanup rollback file", zap.Error(err))
			}
		}()

		// Write new config file
		if err := updateConfigFile(LoggingConfigName, loggerConfigPath, contents); err != nil {
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for logging config", zap.Error(rollbackErr))
			}
			return false, err
		}

		// Parse new logging config
		l, err := logging.NewLoggerConfig(loggerConfigPath)
		if err != nil {
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for logging config", zap.Error(rollbackErr))
			}
			return false, err
		}

		// Parse out options
		opts, err := l.Options()
		if err != nil {
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for logging config", zap.Error(rollbackErr))
			}
			return false, fmt.Errorf("failed updating logging config: %w", err)
		}

		// Create new logger for client
		logger, err := zap.NewProduction(opts...)
		if err != nil {
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for logging config", zap.Error(rollbackErr))
			}
			return false, fmt.Errorf("failed updating logging config: %w", err)
		}

		// Apply logging opts to collector
		rollbackOpts := client.collector.GetLoggingOpts()
		client.collector.SetLoggingOpts(opts)
		if err := client.collector.Restart(context.Background()); err != nil {
			if rollbackErr := rollbackFunc(); rollbackErr != nil {
				client.logger.Error("Rollback failed for logging config", zap.Error(rollbackErr))
			}

			// Restart collector with original logging opts
			client.collector.SetLoggingOpts(rollbackOpts)
			if rollbackErr := client.collector.Restart(context.Background()); rollbackErr != nil {
				client.logger.Error("Collector failed for restart during rollback", zap.Error(rollbackErr))
			}

			return false, fmt.Errorf("failed apply logging update to collector: %w", err)
		}

		// Assign new client logger
		client.logger = logger.Named("opamp")

		return true, nil
	}
}

func updateConfigFile(configName, configPath string, contents []byte) error {
	// Write file
	if err := os.WriteFile(configPath, contents, 0600); err != nil {
		return fmt.Errorf("failed to update config file %s: %w", configName, err)
	}

	return nil
}

func prepRollback(configPath string) (rollbackFunc func() error, cleanupFunc func() error, err error) {
	rollbackPath := fmt.Sprintf("%s.rollback", configPath)

	// Create rollback file
	err = copyFile(configPath, rollbackPath)
	if err != nil {
		return
	}

	// Create rollback func
	rollbackFunc = func() error {
		return copyFile(rollbackPath, configPath)
	}

	// Create cleanupFUnc
	cleanupFunc = func() error {
		return os.Remove(rollbackPath)
	}

	return
}

func copyFile(originPath, newPath string) error {
	cleanOriginPath := filepath.Clean(originPath)
	data, err := os.ReadFile(cleanOriginPath)
	if err != nil {
		return fmt.Errorf("failed to read origin file: %w", err)
	}

	err = os.WriteFile(newPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}

	return nil
}

// saveAgentID persists the agent ID to the manager config file
func saveAgentID(configPath string, agentID opamp.AgentID) error {
	// Read existing config
	cleanPath := filepath.Clean(configPath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to read manager config: %w", err)
	}

	// Unmarshal existing config
	var config opamp.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal manager config: %w", err)
	}

	// Update agent_id field
	config.AgentID = agentID

	// Marshal back to yaml
	newData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal manager config: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(cleanPath, newData, 0600); err != nil {
		return fmt.Errorf("failed to write manager config: %w", err)
	}

	return nil
}
