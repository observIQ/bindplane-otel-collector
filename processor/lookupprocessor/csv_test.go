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

package lookupprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookup(t *testing.T) {
	csvContents := map[string]any{
		"ip":     "0.0.0.0",
		"env":    "prod",
		"region": "us-west",
	}
	csvPath := createTestCSVFile(t, csvContents)
	csvFile := NewCSVFile(csvPath, "ip")
	err := csvFile.Load()
	require.NoError(t, err)

	results, err := csvFile.Lookup("0.0.0.0")
	require.NoError(t, err)
	require.Equal(t, "prod", results["env"])
	require.Equal(t, "us-west", results["region"])

	_, err = csvFile.Lookup("1.1.1.1")
	require.Error(t, err)
	require.ErrorIs(t, err, errKeyNotFound)
}
