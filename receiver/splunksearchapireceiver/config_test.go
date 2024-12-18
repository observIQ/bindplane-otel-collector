// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package splunksearchapireceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		endpoint    string
		username    string
		password    string
		authToken   string
		tokenType   string
		storage     string
		searches    []Search
		errExpected bool
		errText     string
	}{
		{
			desc:     "Missing endpoint",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "missing Splunk server endpoint",
		},
		{
			desc:     "Missing username, no auth token",
			endpoint: "http://localhost:8089",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "missing Splunk basic auth credentials, need username and password",
		},
		{
			desc:     "Missing password, no auth token",
			endpoint: "http://localhost:8089",
			username: "user",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "missing Splunk basic auth credentials, need username and password",
		},
		{
			desc:      "Auth token without token type",
			endpoint:  "http://localhost:8089",
			authToken: "token",
			storage:   "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "auth_token provided without a token type",
		},
		{
			desc:      "Auth token with invalid token type",
			endpoint:  "http://localhost:8089",
			authToken: "token",
			tokenType: "invalid",
			storage:   "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "auth_token provided without a correct token type, valid token types are [Bearer Splunk]",
		},
		{
			desc:      "Auth token and username/password provided",
			endpoint:  "http://localhost:8089",
			username:  "user",
			password:  "password",
			authToken: "token",
			tokenType: "Bearer",
			storage:   "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "auth_token and username/password were both provided, only one can be provided to authenticate with Splunk",
		},
		{
			desc:     "Missing storage",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "storage configuration is required for this receiver",
		},
		{
			desc:        "Missing searches",
			endpoint:    "http://localhost:8089",
			username:    "user",
			password:    "password",
			storage:     "file_storage",
			errExpected: true,
			errText:     "at least one search must be provided",
		},
		{
			desc:     "Missing query",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "missing query in search",
		},
		{
			desc:     "Missing earliest_time",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:      "search index=_internal",
					LatestTime: "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "missing earliest_time in search",
		},
		{
			desc:     "Unparsable earliest_time",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "-1hr",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "earliest_time failed to parse as RFC3339",
		},
		{
			desc:     "Missing latest_time",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "missing latest_time in search",
		},
		{
			desc:     "Unparsable latest_time",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "-1hr",
				},
			},
			errExpected: true,
			errText:     "latest_time failed to parse as RFC3339",
		},
		{
			desc:     "Invalid query chaining",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal | stats count by sourcetype",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "only standalone search commands can be used for scraping data",
		},
		{
			desc:     "Valid config",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: false,
		},
		{
			desc:      "Valid config with auth token",
			endpoint:  "http://localhost:8089",
			authToken: "token",
			tokenType: "Bearer",
			storage:   "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: false,
		},
		{
			desc:     "Valid config with multiple searches",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
				{
					Query:        "search index=_audit",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: false,
		},
		{
			desc:     "Valid config with limit",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
					Limit:        10,
				},
			},
			errExpected: false,
		},
		{
			desc:     "Query with earliest and latest time",
			endpoint: "http://localhost:8089",
			username: "user",
			password: "password",
			storage:  "file_storage",
			searches: []Search{
				{
					Query:        "search index=_internal earliest=2024-10-30T04:00:00.000Z latest=2024-10-30T14:00:00.000Z",
					EarliestTime: "2024-10-30T04:00:00.000Z",
					LatestTime:   "2024-10-30T14:00:00.000Z",
				},
			},
			errExpected: true,
			errText:     "time query parameters must be configured using only the 'earliest_time' and 'latest_time' configuration parameters",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := NewFactory().CreateDefaultConfig().(*Config)
			cfg.Endpoint = tc.endpoint
			cfg.Username = tc.username
			cfg.Password = tc.password
			cfg.AuthToken = tc.authToken
			cfg.TokenType = tc.tokenType
			cfg.Searches = tc.searches
			if tc.storage != "" {
				cfg.StorageID = &component.ID{}
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
