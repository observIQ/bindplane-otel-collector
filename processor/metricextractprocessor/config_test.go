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
			name: "valid config (expr)",
			config: &Config{
				Match:      strp("true"),
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
		},
		{
			name: "valid config (ottl)",
			config: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["message"]`,
				MetricName:  "metric",
				MetricUnit:  "unit",
				MetricType:  gaugeDoubleType,
			},
		},
		{
			name: "invalid metric type",
			config: &Config{
				Match:      strp("true"),
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: "invalid",
			},
			expectedErr: errMetricTypeInvalid.Error(),
		},
		{
			name: "missing extract (expr)",
			config: &Config{
				Match:      strp("true"),
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: errExprExtractMissing.Error(),
		},
		{
			name: "missing extract (ottl)",
			config: &Config{
				OTTLMatch:  strp("true"),
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: errOTTLExtractMissing.Error(),
		},
		{
			name: "invalid match (expr)",
			config: &Config{
				Match:      strp("++"),
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: "invalid match",
		},
		{
			name: "invalid match (ottl)",
			config: &Config{
				OTTLMatch:   strp("++"),
				OTTLExtract: "message",
				MetricName:  "metric",
				MetricUnit:  "unit",
				MetricType:  gaugeDoubleType,
			},
			expectedErr: "invalid ottl_match",
		},
		{
			name: "invalid extract (expr)",
			config: &Config{
				Match:      strp("true"),
				Extract:    "++",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: "invalid extract",
		},
		{
			name: "invalid extract (ottl)",
			config: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: "++",
				MetricName:  "metric",
				MetricUnit:  "unit",
				MetricType:  gaugeDoubleType,
			},
			expectedErr: "invalid ottl_extract",
		},
		{
			name: "invalid attribute (expr)",
			config: &Config{
				Match:      strp("true"),
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
		{
			name: "invalid attribute (ottl)",
			config: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["message"]`,
				MetricName:  "metric",
				MetricUnit:  "unit",
				MetricType:  gaugeDoubleType,
				OTTLAttributes: map[string]string{
					"invalid": "++",
				},
			},
			expectedErr: "invalid ottl_attributes",
		},
		{
			name: "mixed ottl and expr config",
			config: &Config{
				OTTLMatch:  strp("true"),
				Extract:    "message",
				MetricName: "metric",
				MetricUnit: "unit",
				MetricType: gaugeDoubleType,
			},
			expectedErr: "cannot use ottl fields (ottl_match, ottl_extract, ottl_attributes) and expr fields (match, extract, attributes)",
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

func strp(s string) *string {
	return &s
}
