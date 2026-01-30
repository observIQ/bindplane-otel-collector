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

package azureblobpollingreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver"

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc      string
		cfg       *Config
		expectErr error
	}{
		{
			desc: "Missing connection string",
			cfg: &Config{
				ConnectionString: "",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     10 * time.Minute,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         1000,
			},
			expectErr: errors.New("connection_string is required"),
		},
		{
			desc: "Missing container",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "",
				RootFolder:       "root",
				PollInterval:     10 * time.Minute,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         1000,
			},
			expectErr: errors.New("container is required"),
		},
		{
			desc: "Missing poll_interval",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     0,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         1000,
			},
			expectErr: errors.New("poll_interval must be greater than 0"),
		},
		{
			desc: "poll_interval less than 1 minute",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     30 * time.Second,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         1000,
			},
			expectErr: errors.New("poll_interval must be at least 1 minute"),
		},
		{
			desc: "Negative initial_lookback",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     10 * time.Minute,
				InitialLookback:  -5 * time.Minute,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         1000,
			},
			expectErr: errors.New("initial_lookback must be greater than or equal to 0"),
		},
		{
			desc: "Bad batch_size",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     10 * time.Minute,
				DeleteOnRead:     false,
				BatchSize:        0,
				PageSize:         1000,
			},
			expectErr: errors.New("batch_size must be greater than 0"),
		},
		{
			desc: "Bad page_size",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     10 * time.Minute,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         0,
			},
			expectErr: errors.New("page_size must be greater than 0"),
		},
		{
			desc: "Valid config",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     10 * time.Minute,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         1000,
			},
			expectErr: nil,
		},
		{
			desc: "Valid config with initial_lookback",
			cfg: &Config{
				ConnectionString: "connection_string",
				Container:        "container",
				RootFolder:       "root",
				PollInterval:     5 * time.Minute,
				InitialLookback:  1 * time.Hour,
				DeleteOnRead:     false,
				BatchSize:        30,
				PageSize:         1000,
			},
			expectErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.expectErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectErr.Error())
			}
		})
	}
}
