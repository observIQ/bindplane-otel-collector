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

package path

import (
	"os"
	"os/exec"

	"go.uber.org/zap"
)

// AIXInstallDir is the default install directory on AIX.
const AIXInstallDir = "/opt/observiq-otel-collector"

// DefaultConfigOverrides is not used on AIX, but is required
// by InstallDir.
var DefaultConfigOverrides = []string{}

// aixSubsystem is the SRC subsystem the install script creates, as in the service package.
const aixSubsystem = "observiq-otel-collector"

// InstallDir returns the filepath to the install directory: the collector home
// from the SRC environment, then the directory of the program the SRC subsystem
// runs, then the default.
func InstallDir(logger *zap.Logger, _ []string) (string, error) {
	// #nosec G204 -- fixed command and subsystem name
	out, err := exec.Command("lssrc", "-S", "-s", aixSubsystem).Output()
	if err != nil {
		logger.Debug("Could not read the SRC subsystem definition", zap.Error(err))
	}
	return srcInstallDir(os.Getenv("BINDPLANE_COLLECTOR_HOME"), out, AIXInstallDir), nil
}
