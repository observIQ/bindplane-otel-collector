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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/journaldreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/sqlqueryreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/syslogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/tcplogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/udplogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/windowseventlogreceiver"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol"
)

const pluginDirPath = "../../plugins"

// TestValidateSuppliedPlugins ensures each plugin that ships with the collector loads with the current
// version of the receiver
func TestValidateSuppliedPlugins(t *testing.T) {
	entries, err := os.ReadDir(pluginDirPath)
	require.NoError(t, err)

	// Create mock host and load up factories that are used in the current plugin set
	host := &MockHost{}
	// NOTE if adding a new plugin with new receiver, processors, or extensions ensure the factory is added to the below function
	loadUsedPluginFactories(t, host)

	// Used to get factories
	emitterFactory := createLogEmitterFactory(nil)

	tmp := t.TempDir()
	t.Setenv("OIQ_OTEL_COLLECTOR_HOME", tmp)

	for _, entry := range entries {
		entryName := entry.Name()
		t.Run(fmt.Sprintf("Loading %s", entry.Name()), func(t *testing.T) {
			t.Parallel()
			fullFilePath, err := filepath.Abs(filepath.Join(pluginDirPath, entryName))
			require.NoError(t, err, "Failed to determine path of file %s", entryName)

			// Load the plugin
			plugin, err := LoadPlugin(fullFilePath)
			require.NoError(t, err, "Failed to load file %s", entryName)

			cfg := component.ID{}
			err = cfg.UnmarshalText([]byte("test"))
			require.NoError(t, err)

			// Render the config
			renderedCfg, err := plugin.Render(map[string]any{}, cfg)
			require.NoError(t, err, "Failed to render config for plugin %s", entryName)

			// Check receivers and filter out checking those that are not supported by the OS
			for id := range renderedCfg.Receivers {
				componentID := component.ID{}
				err = componentID.UnmarshalText([]byte(id))
				require.NoError(t, err, "Failed to parse component ID %s for plugin %s", id, entryName)

				switch componentID.Type() {
				case windowseventlogreceiver.NewFactory().Type():
					if runtime.GOOS != "windows" {
						return
					}
				case journaldreceiver.NewFactory().Type():
					if runtime.GOOS != "linux" {
						return
					}
				case syslogreceiver.NewFactory().Type():
					if runtime.GOOS == "windows" {
						return
					}
				}
			}

			// Setup to parse rendered config through actual receiver config logic
			factories, err := renderedCfg.GetRequiredFactories(host, emitterFactory)
			require.NoError(t, err, "Failed to get factories for plugin %s", entryName)

			cfgProviderSettings, err := renderedCfg.GetConfigProviderSettings()
			require.NoError(t, err, "Failed to get config provider for plugin %s", entryName)

			configProvider, err := otelcol.NewConfigProvider(*cfgProviderSettings)
			require.NoError(t, err)

			_, err = configProvider.Get(context.Background(), *factories)
			require.NoError(t, err, "Failed to validate config for plugin %s", entryName)

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

			parameterMap := plugin.ApplyDefaults(map[string]any{})

			require.NoError(t, plugin.checkDefined(parameterMap))
			require.NoError(t, plugin.checkSupported(parameterMap))
			require.NoError(t, plugin.checkType(parameterMap))
			// We explicitly don't call checkRequired here, since parameters may not have specified defaults
		})
	}
}

func loadUsedPluginFactories(t *testing.T, host *MockHost) {
	t.Helper()
	// Receivers
	host.On("GetFactory", component.KindReceiver, filelogreceiver.NewFactory().Type()).Return(filelogreceiver.NewFactory())
	host.On("GetFactory", component.KindReceiver, tcplogreceiver.NewFactory().Type()).Return(tcplogreceiver.NewFactory())
	host.On("GetFactory", component.KindReceiver, udplogreceiver.NewFactory().Type()).Return(udplogreceiver.NewFactory())
	host.On("GetFactory", component.KindReceiver, syslogreceiver.NewFactory().Type()).Return(syslogreceiver.NewFactory())
	host.On("GetFactory", component.KindReceiver, prometheusreceiver.NewFactory().Type()).Return(prometheusreceiver.NewFactory())
	host.On("GetFactory", component.KindReceiver, sqlqueryreceiver.NewFactory().Type()).Return(sqlqueryreceiver.NewFactory())
	host.On("GetFactory", component.KindReceiver, journaldreceiver.NewFactory().Type()).Return(journaldreceiver.NewFactory())
	host.On("GetFactory", component.KindReceiver, windowseventlogreceiver.NewFactory().Type()).Return(windowseventlogreceiver.NewFactory())

	// Extensions
	host.On("GetFactory", component.KindExtension, filestorage.NewFactory().Type()).Return(filestorage.NewFactory())

	// Processors
	host.On("GetFactory", component.KindProcessor, filterprocessor.NewFactory().Type()).Return(filterprocessor.NewFactory())
	host.On("GetFactory", component.KindProcessor, metricstransformprocessor.NewFactory().Type()).Return(metricstransformprocessor.NewFactory())
	host.On("GetFactory", component.KindProcessor, transformprocessor.NewFactory().Type()).Return(transformprocessor.NewFactory())
}
