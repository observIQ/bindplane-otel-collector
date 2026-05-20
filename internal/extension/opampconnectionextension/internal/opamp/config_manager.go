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
	"path/filepath"

	"github.com/open-telemetry/opamp-go/protobufs"
)

const (
	// YAMLContentType content type for .yml or .yaml file
	YAMLContentType = "text/yaml"

	// JSONContentType content type for .json file
	JSONContentType = "text/json"
)

// ConfigManager handles remote configuration of local configs
//
//go:generate mockery --name ConfigManager --filename mock_config_manager.go --structname MockConfigManager
type ConfigManager interface {
	// AddConfig adds a config to be tracked by the config manager with it's corresponding validator function.
	AddConfig(configName string, reloader *ManagedConfig)

	// ComposeEffectiveConfig reads in all config files and calculates the effective config
	ComposeEffectiveConfig() (*protobufs.EffectiveConfig, error)

	// ApplyConfigChanges compares the remoteConfig to the existing and applies changes.
	// Calculates new effective config
	ApplyConfigChanges(remoteConfig *protobufs.AgentRemoteConfig) (changed bool, err error)
}

// DetermineContentType looks at the file extension for the given filepath and returns the content type
func DetermineContentType(filePath string) string {
	extension := filepath.Ext(filePath)

	switch extension {
	case ".json":
		return JSONContentType
	case ".yml", ".yaml":
		return YAMLContentType
	default:
		return ""
	}
}
