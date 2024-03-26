// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
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
		{
			desc:        "invalid attributes",
			payloads:    1,
			errExpected: true,
			errText:     "error in attributes config: <Invalid value type struct {}>",
			generators: []GeneratorConfig{
				{
					Type: "logs",
					Attributes: map[string]any{
						"attr_key1": struct{}{},
					},
				},
			},
		},
		{
			desc:        "invalid resource_attributes",
			payloads:    1,
			errExpected: true,
			errText:     "error in resource_attributes config: <Invalid value type struct {}>",
			generators: []GeneratorConfig{
				{
					Type: "logs",
					ResourceAttributes: map[string]any{
						"attr_key1": struct{}{},
					},
				},
			},
		},
		{
			desc:        "otlp - no type",
			payloads:    10,
			errExpected: true,
			errText:     "telemetry_type must be set",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"otlp_json": "json",
					},
				},
			},
		},
		{
			desc:        "otlp - telemetry_type not string",
			payloads:    10,
			errExpected: true,
			errText:     "invalid telemetry type: 1",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"otlp_json":      "json",
						"telemetry_type": 1,
					},
				},
			},
		},
		{
			desc:        "otlp - bad telemetry_type ",
			payloads:    10,
			errExpected: true,
			errText:     "invalid telemetry type: bad",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"otlp_json":      "json",
						"telemetry_type": "bad",
					},
				},
			},
		},
		{
			desc:        "otlp - no otlp_json",
			payloads:    10,
			errExpected: true,
			errText:     "otlp_json must be set",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "logs",
					},
				},
			},
		},
		{
			desc:        "otlp - otlp_json not string",
			payloads:    10,
			errExpected: true,
			errText:     "otlp_json must be a string, got: 1",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "logs",
						"otlp_json":      1,
					},
				},
			},
		},
		{
			desc:        "otlp - malformed otlp_json logs",
			payloads:    10,
			errExpected: true,
			errText:     "error unmarshalling logs from otlp_json: skipThreeBytes: expect ull, error found in #2 byte of ...|not json|..., bigger context ...|not json|...",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "logs",
						"otlp_json":      "not json",
					},
				},
			},
		},
		{
			desc:        "otlp - malformed otlp_json metrics",
			payloads:    10,
			errExpected: true,
			errText:     "error unmarshalling metrics from otlp_json: ReadObjectCB: expect \" after {, but found n, error found in #2 byte of ...|{not json}|..., bigger context ...|{not json}|...",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "metrics",
						"otlp_json":      "{not json}",
					},
				},
			},
		},
		{
			desc:        "otlp - malformed otlp_json traces",
			payloads:    10,
			errExpected: true,
			errText:     "error unmarshalling traces from otlp_json: ReadObjectCB: expect { or n, but found ?, error found in #1 byte of ...|?(not json)|..., bigger context ...|?(not json)|...",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "traces",
						"otlp_json":      "?(not json)",
					},
				},
			},
		},
		{
			desc:        "otlp - otlp_json logs, telemetry_type traces",
			payloads:    10,
			errExpected: true,
			errText:     "no trace spans found in otlp_json",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "traces",
						"otlp_json":      `{"resourceLogs":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"Sams-M1-Pro.local"}},{"key":"os.type","value":{"stringValue":"darwin"}}]},"scopeLogs":[{"scope":{},"logRecords":[{"timeUnixNano":"1709677536097000000","observedTimeUnixNano":"1709677536223996000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.097 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.097 EST"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536111000000","observedTimeUnixNano":"1709677536224110000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.111 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.111 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536113000000","observedTimeUnixNano":"1709677536224164000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.113 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.113 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536126000000","observedTimeUnixNano":"1709677536224300000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.126 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"role","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.126 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536149000000","observedTimeUnixNano":"1709677536224359000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.149 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.149 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536151000000","observedTimeUnixNano":"1709677536224466000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.151 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.151 EST"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536154000000","observedTimeUnixNano":"1709677536224517000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.154 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.154 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536157000000","observedTimeUnixNano":"1709677536224635000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.157 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"duration","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.157 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536159000000","observedTimeUnixNano":"1709677536224688000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.159 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.159 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""}]}]},{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"Sams-M1-Pro.local"}},{"key":"os.type","value":{"stringValue":"darwin"}}]},"scopeLogs":[{"scope":{},"logRecords":[{"timeUnixNano":"1709677536097000000","observedTimeUnixNano":"1709677536223996000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.097 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.097 EST"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536111000000","observedTimeUnixNano":"1709677536224110000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.111 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.111 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536113000000","observedTimeUnixNano":"1709677536224164000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.113 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.113 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536126000000","observedTimeUnixNano":"1709677536224300000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.126 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"role","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.126 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536149000000","observedTimeUnixNano":"1709677536224359000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.149 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.149 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536151000000","observedTimeUnixNano":"1709677536224466000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.151 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.151 EST"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536154000000","observedTimeUnixNano":"1709677536224517000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.154 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.154 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536157000000","observedTimeUnixNano":"1709677536224635000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.157 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"duration","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.157 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536159000000","observedTimeUnixNano":"1709677536224688000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.159 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.159 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""}]}]}]}`,
					},
				},
			},
		},
		{
			desc:        "otlp - telemetry_type metrics, otlp_json logs",
			payloads:    10,
			errExpected: true,
			errText:     "no metric data points found in otlp_json",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "metrics",
						"otlp_json":      `{"resourceLogs":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"Sams-M1-Pro.local"}},{"key":"os.type","value":{"stringValue":"darwin"}}]},"scopeLogs":[{"scope":{},"logRecords":[{"timeUnixNano":"1709677536097000000","observedTimeUnixNano":"1709677536223996000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.097 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.097 EST"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536111000000","observedTimeUnixNano":"1709677536224110000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.111 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.111 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536113000000","observedTimeUnixNano":"1709677536224164000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.113 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.113 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536126000000","observedTimeUnixNano":"1709677536224300000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.126 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"role","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.126 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536149000000","observedTimeUnixNano":"1709677536224359000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.149 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.149 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536151000000","observedTimeUnixNano":"1709677536224466000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.151 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.151 EST"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536154000000","observedTimeUnixNano":"1709677536224517000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.154 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.154 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536157000000","observedTimeUnixNano":"1709677536224635000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.157 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"duration","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.157 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536159000000","observedTimeUnixNano":"1709677536224688000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.159 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.159 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""}]}]},{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"Sams-M1-Pro.local"}},{"key":"os.type","value":{"stringValue":"darwin"}}]},"scopeLogs":[{"scope":{},"logRecords":[{"timeUnixNano":"1709677536097000000","observedTimeUnixNano":"1709677536223996000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.097 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.097 EST"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536111000000","observedTimeUnixNano":"1709677536224110000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.111 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.111 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536113000000","observedTimeUnixNano":"1709677536224164000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.113 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.113 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536126000000","observedTimeUnixNano":"1709677536224300000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.126 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"role","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.126 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536149000000","observedTimeUnixNano":"1709677536224359000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.149 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.149 EST"}},{"key":"role","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"tid","value":{"stringValue":"8334"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536151000000","observedTimeUnixNano":"1709677536224466000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.151 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"duration","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.151 EST"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536154000000","observedTimeUnixNano":"1709677536224517000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.154 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"role","value":{"stringValue":""}},{"key":"duration","value":{"stringValue":""}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.154 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"user","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536157000000","observedTimeUnixNano":"1709677536224635000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.157 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"log_type","value":{"stringValue":"postgresql.general"}},{"key":"duration","value":{"stringValue":""}},{"key":"user","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.157 EST"}},{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}}],"traceId":"","spanId":""},{"timeUnixNano":"1709677536159000000","observedTimeUnixNano":"1709677536224688000","severityNumber":9,"severityText":"LOG","body":{"stringValue":"2024-03-05 17:25:36.159 EST [8334] LOG:  statement: COMMIT"},"attributes":[{"key":"tid","value":{"stringValue":"8334"}},{"key":"role","value":{"stringValue":""}},{"key":"level","value":{"stringValue":"LOG"}},{"key":"message","value":{"stringValue":"statement: COMMIT"}},{"key":"duration","value":{"stringValue":""}},{"key":"statement","value":{"stringValue":"COMMIT"}},{"key":"sql_command","value":{"stringValue":"COMMIT"}},{"key":"timestamp","value":{"stringValue":"2024-03-05 17:25:36.159 EST"}},{"key":"log.file.name","value":{"stringValue":"postgresql-2024-03-05_172300.log"}},{"key":"user","value":{"stringValue":""}},{"key":"log_type","value":{"stringValue":"postgresql.general"}}],"traceId":"","spanId":""}]}]}]}`,
					},
				},
			},
		},

		{
			desc:        "otlp - telemetry_type traces, otlp_json metrics",
			payloads:    10,
			errExpected: true,
			errText:     "no trace spans found in otlp_json",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "traces",
						"otlp_json":      `{"resourceMetrics":[{"resource":{},"scopeMetrics":[{"scope":{},"metrics":[{"summary":{"dataPoints":[{"attributes":[{"key":"prod-machine","value":{"stringValue":"prod-1"}}],"count":"4","sum":111}]}}]}]}]}`,
					},
				},
			},
		},
		{
			desc:        "otlp - telemetry_type logs, otlp_json metrics",
			payloads:    10,
			errExpected: true,
			errText:     "no log records found in otlp_json",
			generators: []GeneratorConfig{
				{
					Type: "otlp",
					AdditionalConfig: map[string]any{
						"telemetry_type": "logs",
						"otlp_json":      `{"resourceMetrics":[{"resource":{},"scopeMetrics":[{"scope":{},"metrics":[{"summary":{"dataPoints":[{"attributes":[{"key":"prod-machine","value":{"stringValue":"prod-1"}}],"count":"4","sum":111}]}}]}]}]}`,
					},
				},
			},
		},
		{
			desc:        "metrics - valid config",
			payloads:    1,
			errExpected: false,
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": 1000000000,
								"type":      "Sum",
								"unit":      "By",
								"attributes": map[string]any{
									"state": "buffered",
								},
							},
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": 1000000000,
								"type":      "Sum",
								"unit":      "By",
								"attributes": map[string]any{
									"state": "slab_reclaimed",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - invalid attributes",
			payloads:    1,
			errExpected: true,
			errText:     "error in attributes config: <Invalid value type struct {}>",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					Attributes: map[string]any{
						"attr_key1": struct{}{},
					},
				},
			},
		},
		{
			desc:        "metrics - invalid resource attributes",
			payloads:    1,
			errExpected: true,
			errText:     "error in resource_attributes config: <Invalid value type struct {}>",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					ResourceAttributes: map[string]any{
						"attr_key1": struct{}{},
					},
				},
			},
		},
		{
			desc:        "metrics - no metrics",
			payloads:    1,
			errExpected: true,
			errText:     "metrics must be set",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
				},
			},
		},
		{
			desc:        "metrics - metrics not array",
			payloads:    1,
			errExpected: true,
			errText:     "metrics must be an array of maps",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": map[string]any{
							"name":      "system.memory.usage",
							"value_min": 100000,
							"value_max": 1000000000,
							"type":      "Sum",
							"unit":      "By",
							"attributes": map[string]any{
								"state": "slab_reclaimed",
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - metric not map",
			payloads:    1,
			errExpected: true,
			errText:     "each metric must be a map",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							1,
						},
					},
				},
			},
		},
		{
			desc:        "metrics - missing name",
			payloads:    1,
			errExpected: true,
			errText:     "each metric must have a name",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"value_min": 100000,
								"value_max": 1000000000,
								"type":      "Sum",
								"unit":      "By",
								"attributes": map[string]any{
									"state": "buffered",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - missing type",
			payloads:    1,
			errExpected: true,
			errText:     "metric system.memory.usage missing type",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": 1000000000,
								"unit":      "By",
								"attributes": map[string]any{
									"state": "buffered",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - invalid type",
			payloads:    1,
			errExpected: true,
			errText:     "error decoding metric: 1 error(s) decoding:\n\n* 'type' expected type 'string', got unconvertible type 'int', value: '1'",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": 1000000000,
								"type":      1,
								"unit":      "By",
								"attributes": map[string]any{
									"state": "slab_reclaimed",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - unknown type",
			payloads:    1,
			errExpected: true,
			errText:     "metric system.memory.usage has invalid metric type: Foo",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": 1000000000,
								"type":      "Foo",
								"unit":      "By",
								"attributes": map[string]any{
									"state": "slab_reclaimed",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - invalid value_min",
			payloads:    1,
			errExpected: true,
			errText:     "error decoding metric: 1 error(s) decoding:\n\n* 'value_min' expected type 'int64', got unconvertible type 'string', value: 'foo'",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_max": 1000000000,
								"value_min": "foo",
								"type":      "Sum",
								"unit":      "By",
								"attributes": map[string]any{
									"state": "slab_reclaimed",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - invalid value_max",
			payloads:    1,
			errExpected: true,
			errText:     "error decoding metric: 1 error(s) decoding:\n\n* 'value_max' expected type 'int64', got unconvertible type 'string', value: 'foo'",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": "foo",
								"type":      "Sum",
								"unit":      "By",
								"attributes": map[string]any{
									"state": "slab_reclaimed",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - missing unit",
			payloads:    1,
			errExpected: true,
			errText:     "metric system.memory.usage missing unit",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": 1000000000,
								"type":      "Sum",
								"attributes": map[string]any{
									"state": "slab_reclaimed",
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "metrics - invalid attributes",
			payloads:    1,
			errExpected: true,
			errText:     "error in attributes config for metric system.memory.usage: <Invalid value type struct {}>",
			generators: []GeneratorConfig{
				{
					Type: "metrics",
					AdditionalConfig: map[string]any{
						"metrics": []any{
							map[string]any{
								"name":      "system.memory.usage",
								"value_min": 100000,
								"value_max": 1000000000,
								"type":      "Sum",
								"unit":      "By",
								"attributes": map[string]any{
									"state": struct{}{},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "host metrics - invalid attributes",
			payloads:    1,
			errExpected: true,
			errText:     "error in resource_attributes config: <Invalid value type struct {}>",
			generators: []GeneratorConfig{
				{
					Type: "host_metrics",
					ResourceAttributes: map[string]any{
						"state": struct{}{},
					},
				},
			},
		},
		{
			desc:        "host metrics - valid config",
			payloads:    1,
			errExpected: false,
			generators: []GeneratorConfig{
				{
					Type: "host_metrics",
					ResourceAttributes: map[string]any{
						"host.name": "2ed77de7e4c1",
						"os.type":   "linux",
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
