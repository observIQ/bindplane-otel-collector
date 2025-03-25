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

package googlecloudstoragerehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		desc      string
		cfg       *Config
		expectErr error
	}{
		{
			desc: "valid config",
			cfg: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    30,
				DeleteOnRead: false,
			},
			expectErr: nil,
		},
		{
			desc: "missing bucket name",
			cfg: &Config{
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    30,
			},
			expectErr: errors.New("bucket_name is required"),
		},
		{
			desc: "missing starting time",
			cfg: &Config{
				BucketName: "test-bucket",
				EndingTime: "2024-01-02T00:00:00Z",
				BatchSize:  30,
			},
			expectErr: errors.New("starting_time is invalid: missing value"),
		},
		{
			desc: "invalid starting time",
			cfg: &Config{
				BucketName:   "test-bucket",
				StartingTime: "invalid",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    30,
			},
			expectErr: errors.New("starting_time is invalid: invalid timestamp format must be in the form YYYY-MM-DDTHH:MM"),
		},
		{
			desc: "missing ending time",
			cfg: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				BatchSize:    30,
			},
			expectErr: errors.New("ending_time is invalid: missing value"),
		},
		{
			desc: "invalid ending time",
			cfg: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "invalid",
				BatchSize:    30,
			},
			expectErr: errors.New("ending_time is invalid: invalid timestamp format must be in the form YYYY-MM-DDTHH:MM"),
		},
		{
			desc: "ending time before starting time",
			cfg: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-02T00:00:00Z",
				EndingTime:   "2024-01-01T00:00:00Z",
				BatchSize:    30,
			},
			expectErr: errors.New("ending_time must be at least one minute after starting_time"),
		},
		{
			desc: "ending time too close to starting time",
			cfg: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-01T00:00:30Z",
				BatchSize:    30,
			},
			expectErr: errors.New("ending_time must be at least one minute after starting_time"),
		},
		{
			desc: "invalid batch size",
			cfg: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    0,
			},
			expectErr: errors.New("batch_size must be greater than 0"),
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
