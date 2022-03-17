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

package azure

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersistenceKey(t *testing.T) {
	type TestKey struct {
		namespace     string
		name          string
		consumerGroup string
		partitionID   string
	}

	cases := []struct {
		name     string
		input    TestKey
		expected string
	}{
		{
			"basic",
			TestKey{
				namespace:     "stanza",
				name:          "devel",
				consumerGroup: "$Default",
				partitionID:   "0",
			},
			"stanza-devel-$Default-0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := Persister{}
			out := p.persistenceKey(tc.input.namespace, tc.input.name, tc.input.consumerGroup, tc.input.partitionID)
			require.Equal(t, tc.expected, out)
		})
	}
}
