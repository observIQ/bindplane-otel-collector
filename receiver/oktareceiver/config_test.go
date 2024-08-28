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

package oktareceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
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
				Domain:       "oktadomain.com",
				APIToken:     "dummyAPIToken",
				PollInterval: time.Second,
			},
		},
		{
			desc:        "fail no domain",
			expectedErr: errNoDomain,
			config: Config{
				APIToken:     "dummyAPIToken",
				PollInterval: time.Second,
			},
		},
		{
			desc:        "fail invalid domain https",
			expectedErr: errInvalidDomain,
			config: Config{
				APIToken:     "dummyAPIToken",
				Domain:       "https://test.okta.com",
				PollInterval: time.Second,
			},
		},
		{
			desc:        "fail invalid domain http",
			expectedErr: errInvalidDomain,
			config: Config{
				APIToken:     "dummyAPIToken",
				Domain:       "http://test.okta.com",
				PollInterval: time.Second,
			},
		},
		{
			desc:        "fail no api token",
			expectedErr: errNoAPIToken,
			config: Config{
				Domain:       "oktadomain.com",
				PollInterval: time.Second,
			},
		},
		{
			desc:        "fail api token empty string",
			expectedErr: errNoAPIToken,
			config: Config{
				Domain:       "oktadomain.com",
				APIToken:     "",
				PollInterval: time.Second,
			},
		},
		{
			desc:        "fail invalid poll interval short",
			expectedErr: errInvalidPollInterval,
			config: Config{
				Domain:       "oktadomain.com",
				APIToken:     "dummyAPIToken",
				PollInterval: 500 * time.Millisecond,
			},
		},
		{
			desc:        "fail invalid poll interval long",
			expectedErr: errInvalidPollInterval,
			config: Config{
				Domain:       "oktadomain.com",
				APIToken:     "dummyAPIToken",
				PollInterval: time.Hour * 25,
			},
		},
		{
			desc:        "fail with no poll interval",
			expectedErr: errInvalidPollInterval,
			config: Config{
				Domain:   "oktadomain.com",
				APIToken: "dummyAPIToken",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := component.ValidateConfig(tc.config)
			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
