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

package marshalprocessor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func Test_processLogs(t *testing.T) {
	testCases := []struct {
		desc                   string
		marshalTo              string
		kvSeparator            rune
		kvPairSeparator        rune
		mapKVSeparator         rune
		mapKVPairSeparator     rune
		inputFilePath          string
		expectedOutputFilePath string
	}{
		{
			desc:                   "Valid - Parsed body to JSON",
			marshalTo:              "JSON",
			inputFilePath:          "parsed-log.json",
			expectedOutputFilePath: `parsed-log-json.json`,
		},
		{
			desc:                   "Invalid - String body to JSON",
			marshalTo:              "JSON",
			inputFilePath:          "string-log.json",
			expectedOutputFilePath: "string-log.json",
		},
		{
			desc:                   "Invalid - String body to KV",
			marshalTo:              "KV",
			inputFilePath:          "string-log.json",
			expectedOutputFilePath: "string-log.json",
		},
		{
			desc:                   "Invalid - String body to XML",
			marshalTo:              "XML",
			inputFilePath:          "string-log.json",
			expectedOutputFilePath: "string-log.json",
		},
		{
			desc:                   "Valid - Parsed and flattened body to KV with default separators",
			marshalTo:              "KV",
			inputFilePath:          "parsed-flattened-log.json",
			expectedOutputFilePath: "parsed-flattened-log-kv.json",
		},
		{
			desc:                   "Valid - Parsed nested body to KV with default separators", // not recommended to use this unflattened format but technically valid
			marshalTo:              "KV",
			inputFilePath:          "parsed-log.json",
			expectedOutputFilePath: "parsed-log-kv.json",
		},
		{
			desc:                   "Valid - Parsed deeply nested body to KV with default separators", // not recommended to use this unflattened format but technically valid
			marshalTo:              "KV",
			inputFilePath:          "parsed-log-deeply-nested.json",
			expectedOutputFilePath: "parsed-log-deeply-nested-kv.json",
		},
		{
			desc:                   "Valid - Parsed deeply nested body to KV with custom Map separators", // not recommended to use this unflattened format but technically valid
			marshalTo:              "KV",
			mapKVSeparator:         ':',
			mapKVPairSeparator:     '|',
			inputFilePath:          "parsed-log-deeply-nested.json",
			expectedOutputFilePath: "parsed-log-deeply-nested-kv-map.json",
		},
		{
			desc:                   "Valid - Parsed deeply nested body to KV with default separators and separators present in nested map", // not recommended to use this unflattened format but technically valid
			marshalTo:              "KV",
			inputFilePath:          "parsed-log-deeply-nested-with-separators.json",
			expectedOutputFilePath: "parsed-log-deeply-nested-with-separators-kv.json",
		},
		{
			desc:                   "Valid - Parsed deeply nested body to KV with custom Map separators and map separators present in nested map", // not recommended to use this unflattened format but technically valid
			marshalTo:              "KV",
			mapKVSeparator:         ':',
			mapKVPairSeparator:     '|',
			inputFilePath:          "parsed-log-deeply-nested-with-separators-map.json",
			expectedOutputFilePath: "parsed-log-deeply-nested-with-separators-kv-map.json",
		},
		{
			desc:                   "Valid - Parsed deeply nested body to KV with normal KV separators in nested map vals but not the custom map separators in them", // not recommended to use this unflattened format but technically valid
			marshalTo:              "KV",
			mapKVSeparator:         ':',
			mapKVPairSeparator:     '|',
			inputFilePath:          "parsed-log-deeply-nested-with-separators-map-2.json",
			expectedOutputFilePath: "parsed-log-deeply-nested-with-separators-kv-map-2.json",
		},
		{
			desc:                   "Valid - Parsed and flattened body to KV with custom pair separator",
			marshalTo:              "KV",
			kvPairSeparator:        '|',
			inputFilePath:          "parsed-flattened-log.json",
			expectedOutputFilePath: "parsed-flattened-log-kv-pipe.json",
		},
		{
			desc:                   "Valid - Parsed and flattened body to KV with custom separator",
			marshalTo:              "KV",
			kvSeparator:            '+',
			inputFilePath:          "parsed-flattened-log.json",
			expectedOutputFilePath: "parsed-flattened-log-kv-plus.json",
		},
		{
			desc:                   "Valid - Parsed and flattened body to KV with default separators as part of the KV values",
			marshalTo:              "KV",
			inputFilePath:          "parsed-flattened-log-with-separators.json",
			expectedOutputFilePath: "parsed-flattened-log-kv-separators.json",
		},
		{
			desc:                   "Valid - Parsed and flattened body to KV with custom separators as part of the KV values",
			marshalTo:              "KV",
			kvPairSeparator:        ',',
			kvSeparator:            ':',
			inputFilePath:          "parsed-flattened-log-with-custom-separators.json",
			expectedOutputFilePath: "parsed-flattened-log-kv-custom-separators.json",
		},
		{
			desc:                   "Valid - Parsed and flattened body to KV with default separators as part of the KV values and quotes inside the KV values",
			marshalTo:              "KV",
			inputFilePath:          "parsed-flattened-log-with-separators-and-quotes.json",
			expectedOutputFilePath: "parsed-flattened-log-kv-separators-and-quotes.json",
		},
		{
			desc:                   "Valid - Parsed body with nested arrays and maps to KV with default separators", // not recommended to use this unflattened format but technically valid
			marshalTo:              "KV",
			inputFilePath:          "parsed-nested-log-with-slice.json",
			expectedOutputFilePath: "parsed-nested-log-with-slice-kv.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &Config{
				MarshalTo:          tc.marshalTo,
				KVSeparator:        tc.kvSeparator,
				KVPairSeparator:    tc.kvPairSeparator,
				MapKVSeparator:     tc.mapKVSeparator,
				MapKVPairSeparator: tc.mapKVPairSeparator,
			}

			if cfg.KVSeparator == 0 {
				cfg.KVSeparator = defaultKVSeparator
			}
			if cfg.KVPairSeparator == 0 {
				cfg.KVPairSeparator = defaultKVPairSeparator
			}
			if cfg.MapKVSeparator == 0 {
				cfg.MapKVSeparator = defaultMapKVSeparator
			}
			if cfg.MapKVPairSeparator == 0 {
				cfg.MapKVPairSeparator = defaultMapKVPairSeparator
			}

			processor := newMarshalProcessor(zap.NewNop(), cfg)
			inputlogs, err := golden.ReadLogs(filepath.Join("testdata", "input", tc.inputFilePath))
			require.NoError(t, err)
			actual, err := processor.processLogs(context.Background(), inputlogs)
			require.NoError(t, err)
			expectedOutput, err := golden.ReadLogs(filepath.Join("testdata", "output", tc.expectedOutputFilePath))
			require.NoError(t, err)

			require.NoError(t, plogtest.CompareLogs(expectedOutput, actual))
		})
	}
}
