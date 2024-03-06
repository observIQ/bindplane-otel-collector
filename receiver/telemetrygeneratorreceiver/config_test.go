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

package telemetrygeneratorreceiver // import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		errExpected bool
		errText     string
		payloads    int
		generators  []GeneratorConfig
	}{
		{
			desc:        "expected case, correct",
			errExpected: false,
			payloads:    1,
		},
		{
			desc:        "no generator type",
			errExpected: true,
			payloads:    1,
			errText:     "invalid generator type: ",
			generators: []GeneratorConfig{
				{
					Type: "",
				},
			},
		},
		{
			desc:        "invalid generator type",
			errExpected: true,
			payloads:    1,
			errText:     "invalid generator type: foo",
			generators: []GeneratorConfig{
				{
					Type: "foo",
				},
			},
		},
		{
			desc:        "payloads per second is 0",
			errExpected: true,
			errText:     "payloads_per_second must be at least 1",
			payloads:    0,
		},
		{
			desc:        "Filled out config",
			errExpected: false,
			errText:     "payloads_per_second must be at least 1",
			payloads:    10,
			generators: []GeneratorConfig{
				{
					Type: "logs",
					Attributes: map[string]any{
						"log_attr1": "log_val1",
						"log_attr2": "log_val2",
					},
					ResourceAttributes: map[string]any{
						"log_attr1": "log_val1",
						"log_attr2": "log_val2",
					},
					AdditionalConfig: map[string]any{
						"log_attr1": "log_val1",
						"log_attr2": "log_val2",
					},
				},
				{
					Type: "host_metrics",
					Attributes: map[string]any{
						"metric_attr1": "metric_val1",
						"metric_attr2": "metric_val2",
					},
					ResourceAttributes: map[string]any{
						"metric_attr1": "metric_val1",
						"metric_attr2": "metric_val2",
					},
					AdditionalConfig: map[string]any{
						"metric_attr1": "metric_val1",
						"metric_attr2": "metric_val2",
					},
				},
				{
					Type: "windows_events",
					Attributes: map[string]any{
						"trace_attr1": "trace_val1",
						"trace_attr2": "trace_val2",
					},
					ResourceAttributes: map[string]any{
						"trace_attr1": "trace_val1",
						"trace_attr2": "trace_val2",
					},
					AdditionalConfig: map[string]any{
						"trace_attr1": "trace_val1",
						"trace_attr2": "trace_val2",
					},
				},
			},
		},
		{
			desc:        "invalid body type",
			errExpected: true,
			errText:     "body must be a string",
			payloads:    10,
			generators: []GeneratorConfig{
				{
					Type: "logs",

					AdditionalConfig: map[string]any{
						"body": 1,
					},
				},
			},
		},
		{
			desc:     "string body type",
			payloads: 10,
			generators: []GeneratorConfig{
				{
					Type: "logs",

					AdditionalConfig: map[string]any{
						"body": `sdfsdf"dfsdf"fsd`,
					},
				},
			},
		},
		{
			desc:        "map body type",
			payloads:    10,
			errExpected: true,
			errText:     "body must be a string",
			generators: []GeneratorConfig{
				{
					Type: "logs",

					AdditionalConfig: map[string]any{
						"body": map[string]any{
							"key": "value",
						},
					},
				},
			},
		},
		{
			desc:     "valid severity",
			payloads: 10,
			generators: []GeneratorConfig{
				{
					Type: "logs",

					AdditionalConfig: map[string]any{
						"severity": 1,
					},
				},
			},
		},
		{
			desc:        "invalid severity",
			payloads:    10,
			errExpected: true,
			errText:     "severity must be an integer",
			generators: []GeneratorConfig{
				{
					Type: "logs",

					AdditionalConfig: map[string]any{
						"severity": "info",
					},
				},
			},
		},
		{
			desc:        "out of range severity",
			payloads:    10,
			errExpected: true,
			errText:     "invalid severity: 100",
			generators: []GeneratorConfig{
				{
					Type: "logs",

					AdditionalConfig: map[string]any{
						"severity": 100,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := NewFactory().CreateDefaultConfig().(*Config)
			cfg.PayloadsPerSecond = tc.payloads
			if tc.generators != nil {
				cfg.Generators = tc.generators
			}
			err := cfg.Validate()

			if tc.errExpected {
				require.EqualError(t, err, tc.errText)
				return
			}

			require.NoError(t, err)
		})
	}
}
