// Copyright observIQ, Inc.
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

package removeemptyvaluesprocessor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestValidStruct(t *testing.T) {
	require.NoError(t, componenttest.CheckConfigStruct(&Config{}))
}

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	testCases := []struct {
		id       component.ID
		expected component.Config
	}{
		{
			id:       component.NewIDWithName(typeStr, "defaults"),
			expected: createDefaultConfig(),
		},
		{
			id: component.NewIDWithName(typeStr, "exclude_keys"),
			expected: &Config{
				RemoveNulls:      false,
				RemoveEmptyLists: true,
				RemoveEmptyMaps:  true,
				EmptyStringValues: []string{
					"-",
				},
				ExcludeKeys: []MapKey{
					{
						field: "body",
						key:   "key",
					},
					{
						field: "resource",
						key:   "key.something",
					},
					{
						field: "attributes",
						key:   "attribute.key",
					},
				},
			},
		},
		{
			id: component.NewIDWithName(typeStr, "exclude_fields"),
			expected: &Config{
				RemoveNulls:       true,
				EmptyStringValues: []string{},
				ExcludeKeys: []MapKey{
					{
						field: "body",
					},
					{
						field: "resource",
					},
					{
						field: "attributes",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.id.String(), func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()

			sub, err := cm.Sub(tc.id.String())
			require.NoError(t, err)
			require.NoError(t, component.UnmarshalConfig(sub, cfg))

			assert.NoError(t, component.ValidateConfig(cfg))
			require.Equal(t, tc.expected, cfg)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		c           Config
		expectedErr string
	}{
		{
			name: "Valid Config",
			c: Config{
				RemoveNulls:      true,
				RemoveEmptyLists: true,
				RemoveEmptyMaps:  true,
				EmptyStringValues: []string{
					"-",
					"",
				},
				ExcludeKeys: []MapKey{
					{
						field: "body",
					},
					{
						field: "attributes",
						key:   "some.key",
					},
				},
			},
		},
		{
			name: "Default Config",
			c:    *createDefaultConfig().(*Config),
		},
		{
			name: "Invalid Config",
			c: Config{
				RemoveNulls:      true,
				RemoveEmptyLists: true,
				RemoveEmptyMaps:  true,
				EmptyStringValues: []string{
					"-",
					"",
				},
				ExcludeKeys: []MapKey{
					{
						field: "bodies",
					},
				},
			},
			expectedErr: "exclude_keys[0]: invalid field (bodies), field must be body, attributes, or resource",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
