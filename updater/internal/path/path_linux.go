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

package path

import (
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

// LinuxInstallDir is the install directory of the collector on linux.
const LinuxInstallDir = "/opt/observiq-otel-collector"

// SystemdFilePath is the path for systemd service
const SystemdFilePath = "/usr/lib/systemd/system/observiq-otel-collector.service"

// SysVFilePath is the path for sysv service
const SysVFilePath = "/etc/init.d/observiq-otel-collector"

// InstallDir returns the filepath to the install directory
func InstallDir(_ *zap.Logger) (string, error) {
	return LinuxInstallDir, nil
}

// LinuxServiceCmdName returns the filename of the service command available
// on this Linux OS. Will be one of systemctl and service
func LinuxServiceCmdName() string {
	var path string
	var err error
	path, err = exec.LookPath("systemctl")
	if err != nil {
		path, err = exec.LookPath("service")
	}
	if err != nil {
		// Defaulting to systemctl in the most common path
		// This replicates prior behavior where this was
		// a static define
		path = "/usr/bin/systemctl"
	}
	_, filename := filepath.Split(path)
	return filename
}

// LinuxServiceFilePath returns the full path to the service file
func LinuxServiceFilePath() string {
	serviceCmd := LinuxServiceCmdName()

	if serviceCmd == "service" {
		return SysVFilePath
	}

	return SystemdFilePath
}
