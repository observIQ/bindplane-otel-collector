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

// Package observiq contains OpAmp structures compatible with the observiq client
package observiq

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/observiq/bindplane-otel-collector/internal/version"
	"github.com/observiq/bindplane-otel-collector/packagestate"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

// Ensure interface is satisfied
var _ types.PackagesStateProvider = (*packagesStateProvider)(nil)

// packagesStateProvider represents a PackagesStateProvider which uses a PackageStateManager to persist PackageStatuses
type packagesStateProvider struct {
	packageStateManager packagestate.StateManager
	logger              *zap.Logger
}

// newPackagesStateProvider creates a new OpAmp PackagesStateProvider
func newPackagesStateProvider(logger *zap.Logger, jsonPath string) types.PackagesStateProvider {
	return &packagesStateProvider{
		packageStateManager: packagestate.NewFileStateManager(logger, jsonPath),
		logger:              logger,
	}
}

// AllPackagesHash not implemented so returns an error with this info
func (p *packagesStateProvider) AllPackagesHash() ([]byte, error) {
	p.logger.Debug("Retrieve all packages hash")

	return nil, errors.New("method not implemented: PackageStateProvider AllPackagesHash")
}

// SetAllPackagesHash not implemented so returns an error with this info
func (p *packagesStateProvider) SetAllPackagesHash(_ []byte) error {
	p.logger.Debug("Set all packages hash")

	return errors.New("method not implemented: PackageStateProvider SetAllPackagesHash")
}

// Packages not implemented so returns an error with this info
func (p *packagesStateProvider) Packages() ([]string, error) {
	p.logger.Debug("Retrieve package names")

	return nil, errors.New("method not implemented: PackageStateProvider Packages")
}

// PackageState not implemented so returns an error with this info
func (p *packagesStateProvider) PackageState(_ string) (state types.PackageState, err error) {
	p.logger.Debug("Retrieve package state")

	packageState := types.PackageState{}

	return packageState, errors.New("method not implemented: PackageStateProvider PackageState")
}

// SetPackageState not implemented so returns an error with this info
func (p *packagesStateProvider) SetPackageState(_ string, _ types.PackageState) error {
	p.logger.Debug("Set package state")

	return errors.New("method not implemented: PackageStateProvider SetPackageState")
}

// CreatePackage not implemented so returns an error with this info
func (p *packagesStateProvider) CreatePackage(_ string, _ protobufs.PackageType) error {
	p.logger.Debug("Create package")

	return errors.New("method not implemented: PackageStateProvider CreatePackage")
}

// FileContentHash not implemented so returns an error with this info
func (p *packagesStateProvider) FileContentHash(_ string) ([]byte, error) {
	p.logger.Debug("Retrieve package content hash")

	return nil, errors.New("method not implemented: PackageStateProvider FileContentHash")
}

// UpdateContent not implemented so returns an error with this info
func (p *packagesStateProvider) UpdateContent(_ context.Context, _ string, _ io.Reader, _, _ []byte) error {
	p.logger.Debug("Update package content")

	return errors.New("method not implemented: PackageStateProvider UpdateContent")
}

// DeletePackage not implemented so returns an error with this info
func (p *packagesStateProvider) DeletePackage(_ string) error {
	p.logger.Debug("Delete package")

	return errors.New("method not implemented: PackageStateProvider DeletePackage")
}

// LastReportedStatuses retrieves the PackagesStatuses from a saved json file
func (p *packagesStateProvider) LastReportedStatuses() (*protobufs.PackageStatuses, error) {
	p.logger.Debug("Retrieve last reported package statuses")

	packages := map[string]*protobufs.PackageStatus{
		packagestate.CollectorPackageName: {
			Name:            packagestate.CollectorPackageName,
			AgentHasVersion: version.Version(),
			Status:          protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
		},
	}
	packageStatuses := &protobufs.PackageStatuses{
		Packages: packages,
	}

	loadedStatues, err := p.packageStateManager.LoadStatuses()

	switch {
	// No File exists so return the status we constructed
	case errors.Is(err, os.ErrNotExist):
		p.logger.Debug("Package statuses json doesn't exist")
		return packageStatuses, nil

	// File existed but error while parsing it
	case err != nil:
		return packageStatuses, fmt.Errorf("failed loading package statuses: %w", err)

	// Successful load
	default:
		return loadedStatues, nil
	}
}

// SetLastReportedStatuses saves the given PackageStatuses into a json file
func (p *packagesStateProvider) SetLastReportedStatuses(statuses *protobufs.PackageStatuses) error {
	p.logger.Debug("Set last reported package statuses")

	return p.packageStateManager.SaveStatuses(statuses)
}
