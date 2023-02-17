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

package aggregationprocessor

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/observiq/observiq-otel-collector/processor/aggregationprocessor/internal/aggregate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

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
			id: component.NewIDWithName(typeStr, ""),
			expected: &Config{
				Interval: 3 * time.Minute,
				Include:  `^test\.thing$$`,
				Aggregations: []aggregate.AggregationType{
					aggregate.LastType,
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
		input       Config
		expectedErr string
	}{
		{
			name: "Config with all aggregations",
			input: Config{
				Interval: 5 * time.Second,
				Include:  "^.*$",
				Aggregations: []aggregate.AggregationType{
					aggregate.AvgType,
					aggregate.MinType,
					aggregate.MaxType,
					aggregate.LastType,
					aggregate.FirstType,
				},
			},
		},
		{
			name: "Config with no aggregations",
			input: Config{
				Interval:     5 * time.Second,
				Include:      "^.*$",
				Aggregations: []aggregate.AggregationType{},
			},
			expectedErr: "at least one aggregation must be specified",
		},
		{
			name: "Config with default aggregations",
			input: Config{
				Interval: 5 * time.Second,
				Include:  "^.*$",
			},
		},
		{
			name: "Config with invalid regex",
			input: Config{
				Interval: 5 * time.Second,
				Include:  "^(",
				Aggregations: []aggregate.AggregationType{
					aggregate.AvgType,
				},
			},
			expectedErr: "`include` regex must be valid",
		},
		{
			name: "Config with invalid interval",
			input: Config{
				Interval: -5 * time.Second,
				Include:  "^.*$",
				Aggregations: []aggregate.AggregationType{
					aggregate.AvgType,
				},
			},
			expectedErr: "aggregation interval must be positive",
		},
		{
			name: "Config with invalid aggregation type",
			input: Config{
				Interval: 5 * time.Second,
				Include:  "^.*$",
				Aggregations: []aggregate.AggregationType{
					aggregate.AggregationType("invalid"),
				},
			},
			expectedErr: "invalid aggregate type for `type`: invalid",
		},
		{
			name: "Config with duplicate aggregations",
			input: Config{
				Interval: 5 * time.Second,
				Include:  "^.*$",
				Aggregations: []aggregate.AggregationType{
					aggregate.AvgType,
					aggregate.AvgType,
				},
			},
			expectedErr: "each aggregation type can only be specified once (avg specified more than once)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidStruct(t *testing.T) {
	require.NoError(t, componenttest.CheckConfigStruct(&Config{}))
}
