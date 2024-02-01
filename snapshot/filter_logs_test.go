// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapshot

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/require"
)

var unfilteredLogsDir = filepath.Join("testdata", "logs", "before")
var filteredLogsDir = filepath.Join("testdata", "logs", "after")

func TestFilterLogs(t *testing.T) {
	testCases := []struct {
		name            string
		fileIn          string
		query           *string
		minTimestamp    *time.Time
		expectedFileOut string
	}{
		{
			name:            "Query matches resource attribute",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			query:           asPtr("Brandons-Legit-Windows-PC-Not-From-Mac-I-Swear"),
			expectedFileOut: filepath.Join(filteredLogsDir, "matches-resource.yaml"),
		},
		{
			name:            "Query matches attribute",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			query:           asPtr("unique-value"),
			expectedFileOut: filepath.Join(filteredLogsDir, "matches-attribute-value.yaml"),
		},
		{
			name:            "Query matches attribute key",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			query:           asPtr("unique-attribute"),
			expectedFileOut: filepath.Join(filteredLogsDir, "matches-attribute-key.yaml"),
		},
		{
			name:            "Query matches field on body",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			query:           asPtr("19.25.92.15"),
			expectedFileOut: filepath.Join(filteredLogsDir, "matches-body.yaml"),
		},
		{
			name:            "No filters",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			expectedFileOut: filepath.Join(filteredLogsDir, "no-filters.yaml"),
		},
		{
			name:            "Query matches string body",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			query:           asPtr("This is a string body"),
			expectedFileOut: filepath.Join(filteredLogsDir, "matches-string-body.yaml"),
		},
		{
			name:            "Filters GET, no timestamp",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			query:           asPtr("GET"),
			expectedFileOut: filepath.Join(filteredLogsDir, "filters-get.yaml"),
		},
		{
			name:            "Filters GET before timestamp",
			fileIn:          filepath.Join(unfilteredLogsDir, "w3c-logs.yaml"),
			query:           asPtr("GET"),
			minTimestamp:    asPtr(time.Unix(0, 1706632434906304000)),
			expectedFileOut: filepath.Join(filteredLogsDir, "filters-get-timestamp.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logsIn, err := golden.ReadLogs(tc.fileIn)
			require.NoError(t, err)

			logsOut := filterLogs(logsIn, tc.query, tc.minTimestamp)

			expectedLogsOut, err := golden.ReadLogs(tc.expectedFileOut)
			require.NoError(t, err)

			err = plogtest.CompareLogs(expectedLogsOut, logsOut)
			require.NoError(t, err)
		})
	}
}

func asPtr[T any](t T) *T {
	return &t
}
