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
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
)

func TestLoadPlugin(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		expectedResult *Plugin
		expectedErr    error
	}{
		{
			name: "valid plugin",
			path: "./testdata/plugin-valid.yaml",
			expectedResult: &Plugin{
				Title:       "test-plugin",
				Template:    "receivers:",
				Version:     "0.0.0",
				Description: "A valid test plugin",
				Parameters: []Parameter{
					{
						Name:      "env",
						Type:      stringType,
						Default:   "prod",
						Supported: []any{"prod", "dev"},
						Required:  true,
					},
				},
			},
		},
		{
			name:        "invalid plugin",
			path:        "./testdata/plugin-invalid-yaml.yaml",
			expectedErr: errors.New("failed to unmarshal plugin from yaml"),
		},
		{
			name:        "missing file",
			path:        "./testdata/missing.yaml",
			expectedErr: errors.New("failed to read file"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := LoadPlugin(tc.path)
			switch tc.expectedErr {
			case nil:
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			default:
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestRenderComponents(t *testing.T) {
	testCases := []struct {
		name           string
		plugin         *Plugin
		values         map[string]any
		dataType       config.DataType
		expectedResult *RenderedConfig
		expectedErr    error
	}{
		{
			name: "invalid template error",
			plugin: &Plugin{
				Template: "{{.invalid",
			},
			expectedErr: errors.New("failed to create plugin template"),
		},
		{
			name: "template execution error",
			plugin: &Plugin{
				Template: `{{template "base" .}}`,
			},
			expectedErr: errors.New("failed to execute template"),
		},
		{
			name: "invalid yaml error",
			plugin: &Plugin{
				Template: "test template",
			},
			expectedErr: errors.New("failed to unmarshal yaml"),
		},
		{
			name: "valid template",
			plugin: &Plugin{
				Template: `
{{if .enabled}}
receivers:
  test:
{{end}}
service:
  pipelines:
    metrics:
      receivers: [test]`,
			},
			values: map[string]any{
				"enabled": true,
			},
			expectedResult: &RenderedConfig{
				Receivers: map[string]any{
					"test": nil,
				},
				Exporters: map[string]any{
					emitterTypeStr: nil,
				},
				Service: ServiceConfig{
					Pipelines: map[string]PipelineConfig{
						"metrics": {
							Receivers: []string{"test"},
							Exporters: []string{emitterTypeStr},
						},
					},
					Telemetry: TelemetryConfig{
						Metrics: MetricsConfig{
							Level: "none",
						},
					},
				},
			},
		},
		{
			name: "valid template with defaults",
			plugin: &Plugin{
				Template: `
{{if .enabled}}
receivers:
  test:
{{end}}
service:
  pipelines:
    metrics:`,
				Parameters: []Parameter{
					{
						Name:    "enabled",
						Default: true,
					},
				},
			},
			expectedResult: &RenderedConfig{
				Receivers: map[string]any{
					"test": nil,
				},
				Exporters: map[string]any{
					emitterTypeStr: nil,
				},
				Service: ServiceConfig{
					Pipelines: map[string]PipelineConfig{
						"metrics": {
							Exporters: []string{emitterTypeStr},
						},
					},
					Telemetry: TelemetryConfig{
						Metrics: MetricsConfig{
							Level: "none",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.plugin.Render(tc.values, config.NewComponentID(config.LogsDataType))
			switch tc.expectedErr {
			case nil:
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			default:
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	testCases := []struct {
		name           string
		plugin         *Plugin
		values         map[string]any
		expectedResult map[string]any
	}{
		{
			name: "with no parameters",
			plugin: &Plugin{
				Parameters: nil,
			},
			values: map[string]any{
				"param1": "value",
			},
			expectedResult: map[string]any{
				"param1": "value",
			},
		},
		{
			name: "with no defaults",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
					},
					{
						Name: "param2",
					},
				},
			},
			values: map[string]any{
				"param1": "value",
			},
			expectedResult: map[string]any{
				"param1": "value",
			},
		},
		{
			name: "with defaults",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name:    "param1",
						Default: "defaultValue1",
					},
					{
						Name:    "param2",
						Default: "defaultValue2",
					},
				},
			},
			values: map[string]any{
				"param1": "value",
			},
			expectedResult: map[string]any{
				"param1": "value",
				"param2": "defaultValue2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.plugin.ApplyDefaults(tc.values)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestCheckParameters(t *testing.T) {
	testCases := []struct {
		name        string
		plugin      *Plugin
		values      map[string]any
		expectedErr error
	}{
		{
			name:   "undefined parameters",
			plugin: &Plugin{},
			values: map[string]any{
				"param1": "value1",
			},
			expectedErr: errors.New("definition failure"),
		},
		{
			name: "missing required",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name:     "param1",
						Required: true,
					},
				},
			},
			values:      map[string]any{},
			expectedErr: errors.New("required failure"),
		},
		{
			name: "invalid string type",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: stringType,
					},
				},
			},
			values: map[string]any{
				"param1": 5,
			},
			expectedErr: errors.New("must be a string"),
		},
		{
			name: "invalid []string type (int)",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: stringArrayType,
					},
				},
			},
			values: map[string]any{
				"param1": 5,
			},
			expectedErr: errors.New("must be a []string"),
		},
		{
			name: "invalid []string type (slice with int)",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: stringArrayType,
					},
				},
			},
			values: map[string]any{
				"param1": []any{
					5,
				},
			},
			expectedErr: errors.New("parameter param1: expected string, but got"),
		},
		{
			name: "invalid int type",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: intType,
					},
				},
			},
			values: map[string]any{
				"param1": "value1",
			},
			expectedErr: errors.New("must be an int"),
		},
		{
			name: "invalid bool type",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: boolType,
					},
				},
			},
			values: map[string]any{
				"param1": "value1",
			},
			expectedErr: errors.New("must be a bool"),
		},
		{
			name: "unsupported type",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: "invalidType",
					},
				},
			},
			values: map[string]any{
				"param1": "value1",
			},
			expectedErr: errors.New("unsupported parameter type: invalidType"),
		},
		{
			name: "not supported value",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name:      "param1",
						Type:      stringType,
						Supported: []any{"value2"},
					},
				},
			},
			values: map[string]any{
				"param1": "value1",
			},
			expectedErr: errors.New("supported value failure"),
		},
		{
			name: "invalid non string timezone type",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: timezoneType,
					},
				},
			},
			values: map[string]any{
				"param1": true,
			},
			expectedErr: errors.New("must be a string"),
		},
		{
			name: "invalid string timezone type",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name: "param1",
						Type: timezoneType,
					},
				},
			},
			values: map[string]any{
				"param1": "Eastern",
			},
			expectedErr: errors.New("must be a valid timezone"),
		},
		{
			name: "valid parameters",
			plugin: &Plugin{
				Parameters: []Parameter{
					{
						Name:      "param1",
						Type:      stringType,
						Required:  true,
						Supported: []any{"value1"},
					},
					{
						Name: "param2",
						Type: stringArrayType,
					},
					{
						Name: "param3",
						Type: intType,
					},
					{
						Name: "param4",
						Type: boolType,
					},
					{
						Name:      "param5",
						Type:      stringType,
						Supported: []any{"value5"},
					},
					{
						Name: "param6",
						Type: timezoneType,
					},
				},
			},
			values: map[string]any{
				"param1": "value1",
				"param2": []any{"value2"},
				"param3": 5,
				"param4": true,
				"param6": "Pacific/Wallis",
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.plugin.CheckParameters(tc.values)
			switch tc.expectedErr {
			case nil:
				require.NoError(t, err)
			default:
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func Test_checkExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	testCases := []struct {
		name        string
		extenstions map[string]any
		pluginName  string
		expectedErr error
	}{
		{
			name:        "Valid Extensions",
			extenstions: map[string]any{"file_storage": map[string]any{"directory": tmpDir}},
			pluginName:  "plugin_one",
		},
		{
			name:        "Invalid Extensions Decoding",
			extenstions: map[string]any{"file_storage": "hello"},
			expectedErr: errors.New("'' expected a map, got 'string'"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p1 := strings.ReplaceAll(tc.pluginName, "/", "_")
			err := checkExtensions(tc.extenstions, tc.pluginName)
			switch tc.expectedErr {
			case nil:
				require.NoError(t, err)
				require.Equal(t, map[string]any{"directory": filepath.Join(tmpDir, p1)}, tc.extenstions["file_storage"])
			default:
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			}
		})
	}
}
