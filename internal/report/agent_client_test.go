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

package report

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentClient(t *testing.T) {
	agentID := "agent_id"
	secretKey := "secret_key"

	client := NewAgentClient(agentID, &secretKey, nil)

	require.Equal(t, agentID, client.agentID)
	require.Equal(t, &secretKey, client.secretKey)
	require.NotNil(t, client.client)
}

// TestAgentClientDo tests the AgentClient Do method
// Note: this unit test can't test the proxy settings since the test server has a loopback address
// (127.*.*.* or localhost) and the proxy settings are ignored for these addresses.
func TestAgentClientDo(t *testing.T) {
	secretKey := "secret_key"
	testCases := []struct {
		desc      string
		agentID   string
		secretKey *string
	}{
		{
			desc:      "No secretKey",
			agentID:   "agent_id",
			secretKey: nil,
		},
		{
			desc:      "With secretKey",
			agentID:   "agent_id",
			secretKey: &secretKey,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				headerID := r.Header.Get("Agent-ID")
				require.Equal(t, tc.agentID, headerID)

				secretKeyHeader := r.Header.Get("X-BindPlane-Secret-Key")
				if tc.secretKey != nil {
					require.Equal(t, *tc.secretKey, secretKeyHeader)
				} else {
					require.Equal(t, "", secretKeyHeader)
				}
			}))
			defer s.Close()

			// Create Client
			client := NewAgentClient(tc.agentID, tc.secretKey, nil)

			// Format small noop request
			req, err := http.NewRequest(http.MethodGet, s.URL, http.NoBody)
			require.NoError(t, err)

			_, err = client.Do(req)
			assert.NoError(t, err)
		})
	}

}
