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

	"github.com/google/go-cmp/cmp"
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
				Aggregations: []AggregateConfig{
					{
						Type: aggregate.LastType,
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
			if diff := cmp.Diff(tc.expected, cfg); diff != "" {
				t.Errorf("Config mismatch (-expected +actual):\n%s", diff)
			}
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
				Aggregations: []AggregateConfig{
					{
						Type: aggregate.AvgType,
					},
					{
						Type: aggregate.MinType,
					},
					{
						Type: aggregate.MaxType,
					},
					{
						Type: aggregate.LastType,
					},
					{
						Type: aggregate.FirstType,
					},
				},
			},
		},
		{
			name: "Config with no aggregations",
			input: Config{
				Interval:     5 * time.Second,
				Include:      "^.*$",
				Aggregations: []AggregateConfig{},
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
				Aggregations: []AggregateConfig{
					{
						Type: aggregate.AvgType,
					},
				},
			},
			expectedErr: "`include` regex must be valid",
		},
		{
			name: "Config with invalid interval",
			input: Config{
				Interval: -5 * time.Second,
				Include:  "^.*$",
				Aggregations: []AggregateConfig{
					{
						Type: aggregate.AvgType,
					},
				},
			},
			expectedErr: "aggregation interval must be positive",
		},
		{
			name: "Config with invalid aggregation type",
			input: Config{
				Interval: 5 * time.Second,
				Include:  "^.*$",
				Aggregations: []AggregateConfig{
					{
						Type: aggregate.AggregationType("invalid"),
					},
				},
			},
			expectedErr: "invalid aggregate type for `type`: invalid",
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

func TestAggregateConfig_MetricNameString(t *testing.T) {
	t.Run("metric name is not specified", func(t *testing.T) {
		metricName := AggregateConfig{
			Type:       aggregate.AvgType,
			MetricName: "",
		}.MetricNameString()
		require.Equal(t, "$0", metricName)
	})

	t.Run("metric name is specified", func(t *testing.T) {
		metricName := AggregateConfig{
			Type:       aggregate.AvgType,
			MetricName: "test.metric",
		}.MetricNameString()
		require.Equal(t, "test.metric", metricName)
	})
}

func TestValidStruct(t *testing.T) {
	require.NoError(t, componenttest.CheckConfigStruct(&Config{}))
}
