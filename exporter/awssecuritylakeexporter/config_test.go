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

package awssecuritylakeexporter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
	cfg := createDefaultConfig().(*Config)
	cfg.Region = "us-east-1"
	cfg.S3Bucket = "aws-security-data-lake-us-east-1-xxxxxxxxxxxx"
	cfg.AccountID = "123456789012"
	cfg.OCSFVersion = OCSFVersion1_3_0
	cfg.CustomSources = []SecurityLakeCustomSource{
		{Name: "my-custom-source", ClassID: 1001},
	}
	return cfg
}

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		config      *Config
		expectedErr string
	}{
		{
			desc:   "Valid config",
			config: validConfig(),
		},
		{
			desc: "Missing region",
			config: func() *Config {
				cfg := validConfig()
				cfg.Region = ""
				return cfg
			}(),
			expectedErr: "region is required",
		},
		{
			desc: "Missing s3_bucket",
			config: func() *Config {
				cfg := validConfig()
				cfg.S3Bucket = ""
				return cfg
			}(),
			expectedErr: "s3_bucket is required",
		},
		{
			desc: "Missing account_id",
			config: func() *Config {
				cfg := validConfig()
				cfg.AccountID = ""
				return cfg
			}(),
			expectedErr: "account_id is required",
		},
		{
			desc: "Missing custom_sources",
			config: func() *Config {
				cfg := validConfig()
				cfg.CustomSources = nil
				return cfg
			}(),
			expectedErr: "at least one custom_source is required",
		},
		{
			desc: "Empty custom_sources",
			config: func() *Config {
				cfg := validConfig()
				cfg.CustomSources = []SecurityLakeCustomSource{}
				return cfg
			}(),
			expectedErr: "at least one custom_source is required",
		},
		{
			desc: "Custom source missing name",
			config: func() *Config {
				cfg := validConfig()
				cfg.CustomSources = []SecurityLakeCustomSource{
					{Name: "", ClassID: 1001},
				}
				return cfg
			}(),
			expectedErr: "custom_source.name is required",
		},
		{
			desc: "Custom source missing class_id",
			config: func() *Config {
				cfg := validConfig()
				cfg.CustomSources = []SecurityLakeCustomSource{
					{Name: "my-source", ClassID: 0},
				}
				return cfg
			}(),
			expectedErr: "custom_source.class_id is required",
		},
		{
			desc: "Missing ocsf_version",
			config: func() *Config {
				cfg := validConfig()
				cfg.OCSFVersion = ""
				return cfg
			}(),
			expectedErr: "ocsf_version is required",
		},
		{
			desc: "Invalid ocsf_version",
			config: func() *Config {
				cfg := validConfig()
				cfg.OCSFVersion = "9.9.9"
				return cfg
			}(),
			expectedErr: "invalid ocsf_version: 9.9.9",
		},
		{
			desc: "Multiple custom sources valid",
			config: func() *Config {
				cfg := validConfig()
				cfg.CustomSources = []SecurityLakeCustomSource{
					{Name: "source-a", ClassID: 1001},
					{Name: "source-b", ClassID: 2001},
				}
				return cfg
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}
