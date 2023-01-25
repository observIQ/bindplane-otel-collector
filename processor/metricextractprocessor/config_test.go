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

package metricextractprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateDefaultProcessorConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.Equal(t, defaultMatch, cfg.Match)
	require.Equal(t, defaultMetricName, cfg.MetricName)
	require.Equal(t, defaultMetricUnit, cfg.MetricUnit)
}

func TestConfigValidate(t *testing.T) {
	var testCases = []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "valid config",
			config: &Config{
				Match:      "true",
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
		},
		{
			name: "invalid metric type",
			config: &Config{
				Match:      "true",
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: "invalid",
			},
			expectedErr: errMetricTypeInvalid.Error(),
		},
		{
			name: "missing extract",
			config: &Config{
				Match:      "true",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: errExtractMissing.Error(),
		},
		{
			name: "invalid match",
			config: &Config{
				Match:      "++",
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: "invalid match",
		},
		{
			name: "invalid extract",
			config: &Config{
				Match:      "true",
				Extract:    "++",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: "invalid extract",
		},
		{
			name: "invalid attribute",
			config: &Config{
				Match:      "true",
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
				Attributes: map[string]string{
					"invalid": "++",
				},
			},
			expectedErr: "invalid attributes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectedErr == "" {
				require.NoError(t, err)
				return
			}

			require.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
