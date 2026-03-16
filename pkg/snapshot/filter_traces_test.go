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
	"github.com/stretchr/testify/require"
)

var unfilteredTracesDir = filepath.Join("testdata", "traces", "before")
var filteredTracesDir = filepath.Join("testdata", "traces", "after")

func TestFilterTraces(t *testing.T) {
	testCases := []struct {
		name             string
		fileIn           string
		searchQuery      *string
		minimumTimestamp *time.Time
		expectedFileOut  string
	}{
		{
			name:            "Query matches resource attribute",
			fileIn:          filepath.Join(unfilteredTracesDir, "bindplane-traces.yaml"),
			searchQuery:     asPtr("Sams-M1-Pro.local"),
			expectedFileOut: filepath.Join(filteredTracesDir, "matches-resource.yaml"),
		},

		{
			name:            "No filters",
			fileIn:          filepath.Join(unfilteredTracesDir, "bindplane-traces.yaml"),
			expectedFileOut: filepath.Join(filteredTracesDir, "no-filters.yaml"),
		},

		{
			name:            "Filters pgstore/pgResourceInternal, no timestamp",
			fileIn:          filepath.Join(unfilteredTracesDir, "bindplane-traces.yaml"),
			searchQuery:     asPtr("pgstore/pgResourceInternal"),
			expectedFileOut: filepath.Join(filteredTracesDir, "filters-resource-internal.yaml"),
		},
		{
			name:             "Filters pgstore/pgResourceInternal before timestamp",
			fileIn:           filepath.Join(unfilteredTracesDir, "bindplane-traces.yaml"),
			searchQuery:      asPtr("pgstore/pgResourceInternal"),
			minimumTimestamp: asPtr(time.Unix(0, 1706791445370893000)),
			expectedFileOut:  filepath.Join(filteredTracesDir, "filters-resource-internal-time.yaml"),
		},
		{
			name:            "Filters parent span id",
			fileIn:          filepath.Join(unfilteredTracesDir, "bindplane-traces.yaml"),
			searchQuery:     asPtr("aa50d71d28f47370"),
			expectedFileOut: filepath.Join(filteredTracesDir, "filters-parent-span-id.yaml"),
		},
		{
			name:            "Filters span id",
			fileIn:          filepath.Join(unfilteredTracesDir, "bindplane-traces.yaml"),
			searchQuery:     asPtr("0f3367e6b090ffed"),
			expectedFileOut: filepath.Join(filteredTracesDir, "filters-span-id.yaml"),
		},
		{
			name:            "Filters kind",
			fileIn:          filepath.Join(unfilteredTracesDir, "bindplane-traces.yaml"),
			searchQuery:     asPtr("Server"),
			expectedFileOut: filepath.Join(filteredTracesDir, "filters-kind-id.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tracesIn, err := golden.ReadTraces(tc.fileIn)
			require.NoError(t, err)

			tracesOut := filterTraces(tracesIn, tc.searchQuery, tc.minimumTimestamp)

			expectedTracesOut, err := golden.ReadTraces(tc.expectedFileOut)
			require.NoError(t, err)
			require.Equal(t, expectedTracesOut, tracesOut)
		})
	}
}
