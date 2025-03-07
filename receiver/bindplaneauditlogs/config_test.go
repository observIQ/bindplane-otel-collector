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

package bindplaneauditlogs

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		expectedErr error
		config      Config
	}{
		{
			desc: "pass simple",
			config: Config{
				BindplaneURLString: "https://app.bindplane.com",
				APIKey:             "testkey",
				PollInterval:       time.Second * 10,
			},
		},
		{
			desc:        "fail no api key",
			expectedErr: errors.New("api_key cannot be empty"),
			config: Config{
				BindplaneURLString: "https://localhost:3000",
				PollInterval:       time.Second * 10,
			},
		},
		{
			desc:        "fail no bindplane url",
			expectedErr: errors.New("bindplane_url_string cannot be empty"),
			config: Config{
				APIKey:       "testkey",
				PollInterval: time.Second * 10,
			},
		},
		{
			desc:        "fail invalid bindplane url no host",
			expectedErr: errors.New("bindplane_url_string must contain a host and scheme"),
			config: Config{
				BindplaneURLString: "invalid-url",
				APIKey:             "testkey",
				PollInterval:       time.Second * 10,
			},
		},
		{
			desc:        "fail invalid bindplane url no scheme",
			expectedErr: errors.New("bindplane_url_string must contain a host and scheme"),
			config: Config{
				BindplaneURLString: "localhost:3000",
				APIKey:             "testkey",
				PollInterval:       time.Second * 10,
			},
		},
		{
			desc:        "fail invalid poll interval",
			expectedErr: errors.New("poll_interval must be between 10 seconds and 24 hours"),
			config: Config{
				BindplaneURLString: "https://localhost:3000",
				APIKey:             "testkey",
				PollInterval:       time.Second,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.config.Validate()
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
