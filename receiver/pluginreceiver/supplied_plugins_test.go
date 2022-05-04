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

package pluginreceiver

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pluginDirPath = "../../plugins"

// TestValidateSuppliedPlugins ensures each plugin that ships with the collector loads with the current
// version of the receiver
func TestValidateSuppliedPlugins(t *testing.T) {
	entries, err := os.ReadDir(pluginDirPath)
	require.NoError(t, err)

	for _, entry := range entries {
		entryName := entry.Name()
		t.Run(fmt.Sprintf("Loading %s", entry.Name()), func(t *testing.T) {
			t.Parallel()
			fullFilePath, err := filepath.Abs(filepath.Join(pluginDirPath, entryName))
			assert.NoError(t, err, "Failed to determine path of file %s", entryName)

			// Load the plugin
			plugin, err := LoadPlugin(fullFilePath)
			assert.NoError(t, err, "Failed to load file %s", entryName)

			_, err = plugin.RenderComponents(map[string]interface{}{})
			assert.NoError(t, err, "Failed to render components for plugin %s", entryName)
		})
	}
}

// TestValidateSuppliedPluginsLoadSuppliedDefaults ensures each plugin can be loaded if the defaults
// are supplied for configuration (e.g. no mismatched types for defaults, no unsupported values for defaults)
func TestValidateSuppliedPluginsLoadSuppliedDefaults(t *testing.T) {
	entries, err := os.ReadDir(pluginDirPath)
	require.NoError(t, err)

	for _, entry := range entries {
		entryName := entry.Name()
		t.Run(fmt.Sprintf("Loading %s", entryName), func(t *testing.T) {
			t.Parallel()
			fullFilePath, err := filepath.Abs(filepath.Join(pluginDirPath, entryName))
			require.NoError(t, err, "Failed to determine path of file %s", entryName)

			// Load the plugin
			plugin, err := LoadPlugin(fullFilePath)
			require.NoError(t, err, "Failed to load file %s", entryName)

			parameterMap := plugin.ApplyDefaults(map[string]interface{}{})

			require.NoError(t, plugin.checkDefined(parameterMap))
			require.NoError(t, plugin.checkSupported(parameterMap))
			require.NoError(t, plugin.checkType(parameterMap))
			// We explicitly don't call checkRequired here, since parameters may not have specified defaults
		})
	}
}
