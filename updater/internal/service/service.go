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

// Package service handles interacting with the OS specific service manager
package service

import (
	"bytes"
	"os"
	"path/filepath"
)

// Service represents a controllable service
//
//go:generate mockery --name Service --filename mock_service.go --structname MockService
type Service interface {
	// Start the service
	Start() error

	// Stop the service
	Stop() error

	// Updates the old service configuration to the new one
	Update() error

	// Backup backs the current service configuration
	Backup() error
}

// replaceInstallDir replaces "[INSTALLDIR]" with the given installDir string.
// This is meant to mimic windows "formatted" string syntax.
func replaceInstallDir(unformattedBytes []byte, installDir string) []byte {
	installDirClean := filepath.Clean(installDir) + string(os.PathSeparator)
	return bytes.ReplaceAll(unformattedBytes, []byte("[INSTALLDIR]"), []byte(installDirClean))
}
