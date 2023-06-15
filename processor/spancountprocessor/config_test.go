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

package spancountprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateDefaultProcessorConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.Equal(t, defaultInterval, cfg.Interval)
	require.Equal(t, defaultMetricName, cfg.MetricName)
	require.Equal(t, defaultMetricUnit, cfg.MetricUnit)
}

func TestConfig_Validate(t *testing.T) {
	ottlMatch := "true"
	testCases := []struct {
		name   string
		config *Config
		err    string
	}{
		{
			name:   "default",
			config: createDefaultConfig().(*Config),
		},
		{
			name: "both match and ottl_match set",
			config: &Config{
				Match:     strp("true"),
				OTTLMatch: &ottlMatch,
			},
			err: "only one of match and ottl_match can be set",
		},
		{
			name: "both attributes and ottl attributes are set",
			config: &Config{
				Attributes:     map[string]string{"thing": "true"},
				OTTLAttributes: map[string]string{"thing": "true"},
			},
			err: "only one of attributes and ottl_attributes can be set",
		},
		{
			name: "both match and ottl attributes are set",
			config: &Config{
				Match:          strp("true"),
				OTTLAttributes: map[string]string{"thing": "true"},
			},
			err: "cannot use match with ottl_attributes",
		},
		{
			name: "both match and ottl attributes are set",
			config: &Config{
				OTTLMatch:  strp("true"),
				Attributes: map[string]string{"thing": "true"},
			},
			err: "cannot use ottl_match with attributes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
