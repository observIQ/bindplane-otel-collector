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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// DefaultLinuxInstallDir is the install directory of the collector on linux.
const DefaultLinuxInstallDir = "/opt/observiq-otel-collector"

// SystemdFilePath is the path for systemd service
const SystemdFilePath = "/usr/lib/systemd/system/observiq-otel-collector.service"

// SysVFilePath is the path for sysv service
const SysVFilePath = "/etc/init.d/observiq-otel-collector"

// DefaultConfigOverrides is a list of config files that can be used to override
// package installation behavior. The updater needs to respect these config
// options, such as BDOT_CONFIG_HOME
var DefaultConfigOverrides = []string{
	"/etc/default/observiq-otel-collector",
	"/etc/sysconfig/observiq-otel-collector",
}

const (
	configHomeOverrideKey = "BDOT_CONFIG_HOME"
)

// InstallDir returns the filepath to the install directory
func InstallDir(logger *zap.Logger, configOverrides []string) (string, error) {
	// Check for a custom install directory
	for _, config := range configOverrides {
		installDir, err := customInstallDir(logger, config)
		if err != nil {
			return "", fmt.Errorf("error checking config file %q: %w", config, err)
		}
		if installDir != "" {
			return installDir, nil
		}
	}

	return DefaultLinuxInstallDir, nil
}

// customInstallDir checks the provided config file for a custom install directory.
// Returns an error if there is an unexpected issue reading the file. Does not
// return an error if the file does not exist.
// If a custom install dir is not found in the config file, it returns an empty string.
func customInstallDir(logger *zap.Logger, config string) (string, error) {
	_, err := os.Stat(config)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read package override file %s: %w", config, err)
	}

	// Buffer is larger enough to read the entire
	// user defined config override file.
	buff := make([]byte, 4000)

	// #nosec G304 -- config is a file path from const DefaultConfigOverrides
	// Test cases will override configOverrides to test this code path.
	file, err := os.Open(config)
	if err != nil {
		return "", fmt.Errorf("open package override file %s: %w", config, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("close package override file", zap.String("file", config), zap.Error(err))
		}
	}()

	_, err = file.Read(buff)
	if err != nil {
		return "", fmt.Errorf("read package override file %s: %w", config, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(buff)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, configHomeOverrideKey+"=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				if value != "" {
					return value, nil
				}
				// If the value is empty, continue to the next line
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning config file %s: %w", config, err)
	}

	return "", nil
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
