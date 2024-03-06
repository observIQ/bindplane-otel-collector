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

package telemetrygeneratorreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

var expectedLogsDir = filepath.Join("testdata", "expected_logs")

func TestLogsGenerator(t *testing.T) {

	test := []struct {
		name         string
		cfg          GeneratorConfig
		expectedFile string
	}{
		{
			name: "one key value pair",
			cfg: GeneratorConfig{
				Type: generatorTypeLogs,
				ResourceAttributes: map[string]any{
					"res_key": "res_value",
				},
				Attributes: map[string]any{
					"attr_key": "attr_value",
				},
				AdditionalConfig: map[string]any{
					"body": "body_value",
				},
			},
			expectedFile: filepath.Join(expectedLogsDir, "one_key.yaml"),
		},
		{
			name: "two key value pair",
			cfg: GeneratorConfig{
				Type: generatorTypeLogs,
				ResourceAttributes: map[string]any{
					"res_key1": "res_value1",
					"res_key2": "res_value2",
				},
				Attributes: map[string]any{
					"attr_key1": "attr_value1",
					"attr_key2": "attr_value2",
				},
				AdditionalConfig: map[string]any{
					"body": "body_value",
				},
			},
			expectedFile: filepath.Join(expectedLogsDir, "two_key.yaml"),
		},
		{
			name: "non string values",
			cfg: GeneratorConfig{
				Type: generatorTypeLogs,
				ResourceAttributes: map[string]any{
					"res_key1": "res_value1",
					"res_key2": "res_value2",
				},
				Attributes: map[string]any{
					"attr_key1": 1,
					"attr_key2": 2.0,
					"attr_key3": true,
				},
				AdditionalConfig: map[string]any{
					"body":     "body_value",
					"severity": 1,
				},
			},
			expectedFile: filepath.Join(expectedLogsDir, "non_string_values.yaml"),
		},
		{
			name: "empty values",
			cfg: GeneratorConfig{
				Type: generatorTypeLogs,
				ResourceAttributes: map[string]any{
					"res_key1": "",
					"res_key2": "",
				},
				Attributes: map[string]any{
					"attr_key1": "",
					"attr_key2": "",
				},
				AdditionalConfig: map[string]any{
					"body": "",
				},
			},
			expectedFile: filepath.Join(expectedLogsDir, "empty_values.yaml"),
		},
	}

	for _, tc := range test {
		t.Run(tc.name, func(t *testing.T) {
			g := newLogsGenerator(tc.cfg, zap.NewNop())
			logs := g.generateLogs()
			clearTimeStamps(logs)
			expectedLogs, err := golden.ReadLogs(tc.expectedFile)
			require.NoError(t, err)
			clearTimeStamps(expectedLogs)
			err = plogtest.CompareLogs(expectedLogs, logs)
			require.NoError(t, err)
		})
	}
}

func clearTimeStamps(logs plog.Logs) {
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resourceLogs := logs.ResourceLogs().At(i)
		for k := 0; k < resourceLogs.ScopeLogs().Len(); k++ {
			scopeLogs := resourceLogs.ScopeLogs().At(k)
			for j := 0; j < scopeLogs.LogRecords().Len(); j++ {
				log := scopeLogs.LogRecords().At(j)
				log.SetTimestamp(pcommon.NewTimestampFromTime(time.Time{}))
			}
		}
	}
}
