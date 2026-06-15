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

package ocsfstandardizationprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetOCSFSchema(t *testing.T) {
	for _, version := range OCSFVersions {
		t.Run(string(version), func(t *testing.T) {
			schema := getOCSFSchema(version)
			require.NotNil(t, schema, "getOCSFSchema should return non-nil for version %s", version)
		})
	}

	t.Run("unsupported version returns nil", func(t *testing.T) {
		schema := getOCSFSchema(OCSFVersion("99.99.99"))
		require.Nil(t, schema)
	})
}
