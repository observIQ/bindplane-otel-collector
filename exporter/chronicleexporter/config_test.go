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

package chronicleexporter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		config      *Config
		expectedErr string
	}{
		{
			desc: "Both creds_file_path and creds are set",
			config: &Config{
				CredsFilePath: "/path/to/creds_file",
				Creds:         "creds_example",
				LogType:       "log_type_example",
				Compression:   noCompression,
			},
			expectedErr: "can only specify creds_file_path or creds",
		},
		{
			desc: "LogType is empty",
			config: &Config{
				Creds:       "creds_example",
				Compression: noCompression,
			},
			expectedErr: "log_type is required",
		},

		{
			desc: "Valid config with creds",
			config: &Config{
				Creds:       "creds_example",
				LogType:     "log_type_example",
				Compression: noCompression,
			},
			expectedErr: "",
		},
		{
			desc: "Valid config with creds_file_path",
			config: &Config{
				CredsFilePath: "/path/to/creds_file",
				LogType:       "log_type_example",
				Compression:   noCompression,
			},
			expectedErr: "",
		},
		{
			desc: "Valid config with raw log field",
			config: &Config{
				CredsFilePath: "/path/to/creds_file",
				LogType:       "log_type_example",
				RawLogField:   `body["field"]`,
				Compression:   noCompression,
			},
			expectedErr: "",
		},
		{
			desc: "Invalid compression type",
			config: &Config{
				CredsFilePath: "/path/to/creds_file",
				LogType:       "log_type_example",
				Compression:   "invalid",
			},
			expectedErr: "invalid compression type",
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
