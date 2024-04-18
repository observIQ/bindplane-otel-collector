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

package awss3rehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver"

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
			desc: "Missing region",
			cfg: &Config{
				Region:       "",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "2023-10-02T17:01",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("region is required"),
		},
		{
			desc: "Missing S3Bucket",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "2023-10-02T17:01",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("s3_bucket is required"),
		},
		{
			desc: "Missing starting_time",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "",
				EndingTime:   "2023-10-02T17:01",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("starting_time is invalid: missing value"),
		},
		{
			desc: "Missing ending_time",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("ending_time is invalid: missing value"),
		},
		{
			desc: "Invalid starting_time",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "invalid_time",
				EndingTime:   "2023-10-02T17:01",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("starting_time is invalid: invalid timestamp"),
		},
		{
			desc: "Missing ending_time",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "invalid_time",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("ending_time is invalid: invalid timestamp"),
		},
		{
			desc: "ending_time not after starting_time",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "2023-10-02T16:00",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("ending_time must be at least one minute after starting_time"),
		},
		{
			desc: "Bad poll_interval",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "2023-10-02T17:01",
				DeleteOnRead: false,
				PollInterval: time.Millisecond,
				PollTimeout:  time.Second * 10,
			},
			expectErr: errors.New("poll_interval must be at least one second"),
		},
		{
			desc: "Bad poll_timeout",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "2023-10-02T17:01",
				DeleteOnRead: false,
				PollInterval: time.Second * 2,
				PollTimeout:  time.Millisecond,
			},
			expectErr: errors.New("poll_timeout must be at least one second"),
		},
		{
			desc: "Valid config",
			cfg: &Config{
				Region:       "connection_string",
				S3Bucket:     "S3Bucket",
				S3Prefix:     "root",
				StartingTime: "2023-10-02T17:00",
				EndingTime:   "2023-10-02T17:01",
				DeleteOnRead: false,
				PollInterval: time.Second,
				PollTimeout:  time.Second * 10,
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
