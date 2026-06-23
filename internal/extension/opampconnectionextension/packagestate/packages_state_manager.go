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

// Package packagestate contains structures for reading and writing the package status
package packagestate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

// CollectorPackageName is the name for the top level packages for this collector
const CollectorPackageName = "observiq-otel-collector"

// DefaultFileName is the default name of the file use to store state
const DefaultFileName = "package_statuses.json"

// StateManager tracks Package states
//
//go:generate mockery --name StateManager --filename mock_state_manager.go --structname MockStateManager
type StateManager interface {
	// LoadStatuses retrieves the previously saved PackagesStatuses.
	// If none were saved returns error
	LoadStatuses() (*protobufs.PackageStatuses, error)

	// SaveStatuses saves the given PackageStatuses
	SaveStatuses(statuses *protobufs.PackageStatuses) error
}

// FileStateManager manages state on disk via a JSON file
type FileStateManager struct {
	jsonPath string
	logger   *zap.Logger
}

type packageState struct {
	Name          string                      `json:"name"`
	AgentVersion  string                      `json:"agent_version"`
	AgentHash     []byte                      `json:"agent_hash"`
	ServerVersion string                      `json:"server_version"`
	ServerHash    []byte                      `json:"server_hash"`
	Status        protobufs.PackageStatusEnum `json:"status"`
	ErrorMessage  string                      `json:"error_message"`
}
type packageStates struct {
	AllPackagesHash []byte                   `json:"all_packages_hash"`
	AllErrorMessage string                   `json:"all_error_message"`
	PackageStates   map[string]*packageState `json:"package_states"`
}

// NewFileStateManager creates a new PackagesStateManager
func NewFileStateManager(logger *zap.Logger, jsonPath string) StateManager {
	return &FileStateManager{
		jsonPath: filepath.Clean(jsonPath),
		logger:   logger,
	}
}

// LoadStatuses retrieves the PackagesStatuses from a saved json file
func (p *FileStateManager) LoadStatuses() (*protobufs.PackageStatuses, error) {
	p.logger.Debug("Loading package statuses")

	statusesBytes, err := os.ReadFile(p.jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package statuses json: %w", err)
	}

	var packageStates packageStates
	if err := json.Unmarshal(statusesBytes, &packageStates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal package statuses: %w", err)
	}

	return packageStatesToStatuses(packageStates), nil
}

// SaveStatuses saves the given PackageStatuses into a json file
func (p *FileStateManager) SaveStatuses(statuses *protobufs.PackageStatuses) error {
	p.logger.Debug("Saving package statuses")

	// If there is any problem saving the new package statuses, make sure that we delete any existing file
	// in order to not have outdated data as its better to start fresh
	if err := os.Remove(p.jsonPath); err != nil {
		p.logger.Debug("Failed to delete package statuses json", zap.Error(err))
	}

	states := packageStatusesToStates(statuses)

	data, err := json.Marshal(states)
	if err != nil {
		return fmt.Errorf("failed to marshal package statuses: %w", err)
	}

	// Write data to a package_statuses.json file, with 0600 file permission
	if err := os.WriteFile(p.jsonPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write package statuses json: %w", err)
	}

	return nil
}

func packageStatusesToStates(statuses *protobufs.PackageStatuses) *packageStates {
	states := &packageStates{
		AllPackagesHash: statuses.GetServerProvidedAllPackagesHash(),
		AllErrorMessage: statuses.GetErrorMessage(),
	}

	packageStates := map[string]*packageState{}
	for name, packageStatus := range statuses.Packages {
		packageState := &packageState{
			Name:          packageStatus.GetName(),
			AgentVersion:  packageStatus.GetAgentHasVersion(),
			AgentHash:     packageStatus.GetAgentHasHash(),
			ServerVersion: packageStatus.GetServerOfferedVersion(),
			ServerHash:    packageStatus.GetServerOfferedHash(),
			Status:        packageStatus.GetStatus(),
			ErrorMessage:  packageStatus.GetErrorMessage(),
		}
		packageStates[name] = packageState
	}
	states.PackageStates = packageStates

	return states
}

func packageStatesToStatuses(states packageStates) *protobufs.PackageStatuses {
	statuses := &protobufs.PackageStatuses{
		ServerProvidedAllPackagesHash: states.AllPackagesHash,
		ErrorMessage:                  states.AllErrorMessage,
	}

	packages := map[string]*protobufs.PackageStatus{}
	for name, packageState := range states.PackageStates {
		packageStatus := &protobufs.PackageStatus{
			Name:                 packageState.Name,
			AgentHasVersion:      packageState.AgentVersion,
			AgentHasHash:         packageState.AgentHash,
			ServerOfferedVersion: packageState.ServerVersion,
			ServerOfferedHash:    packageState.ServerHash,
			Status:               packageState.Status,
			ErrorMessage:         packageState.ErrorMessage,
		}
		packages[name] = packageStatus
	}
	statuses.Packages = packages

	return statuses
}
