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
				CredsFilePath:             "/path/to/creds_file",
				Creds:                     "creds_example",
				LogType:                   "log_type_example",
				Compression:               noCompression,
				BatchLogCountLimitGRPC:    defaultBatchLogCountLimitGRPC,
				BatchRequestSizeLimitGRPC: defaultBatchRequestSizeLimitGRPC,
			},
			expectedErr: "can only specify creds_file_path or creds",
		},
		{
			desc: "Valid config with creds",
			config: &Config{
				Creds:                     "creds_example",
				LogType:                   "log_type_example",
				Compression:               noCompression,
				Protocol:                  protocolGRPC,
				BatchLogCountLimitGRPC:    defaultBatchLogCountLimitGRPC,
				BatchRequestSizeLimitGRPC: defaultBatchRequestSizeLimitGRPC,
			},
			expectedErr: "",
		},
		{
			desc: "Valid config with creds_file_path",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				Compression:               noCompression,
				Protocol:                  protocolGRPC,
				BatchLogCountLimitGRPC:    defaultBatchLogCountLimitGRPC,
				BatchRequestSizeLimitGRPC: defaultBatchRequestSizeLimitGRPC,
			},
			expectedErr: "",
		},
		{
			desc: "Valid config with raw log field",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				RawLogField:               `body["field"]`,
				Compression:               noCompression,
				Protocol:                  protocolGRPC,
				BatchLogCountLimitGRPC:    defaultBatchLogCountLimitGRPC,
				BatchRequestSizeLimitGRPC: defaultBatchRequestSizeLimitGRPC,
			},
			expectedErr: "",
		},
		{
			desc: "Invalid batch log count limit",
			config: &Config{
				Creds:                     "creds_example",
				LogType:                   "log_type_example",
				Compression:               noCompression,
				Protocol:                  protocolGRPC,
				BatchLogCountLimitGRPC:    0,
				BatchRequestSizeLimitGRPC: defaultBatchRequestSizeLimitGRPC,
			},
			expectedErr: "positive batch count log limit is required when protocol is grpc",
		},
		{
			desc: "Invalid batch request size limit",
			config: &Config{
				Creds:                     "creds_example",
				LogType:                   "log_type_example",
				Compression:               noCompression,
				Protocol:                  protocolGRPC,
				BatchLogCountLimitGRPC:    defaultBatchLogCountLimitGRPC,
				BatchRequestSizeLimitGRPC: 0,
			},
			expectedErr: "positive batch request size limit is required when protocol is grpc",
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
		{
			desc: "Protocol is https and location is empty",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				Protocol:                  protocolHTTPS,
				Compression:               noCompression,
				Forwarder:                 "forwarder_example",
				Project:                   "project_example",
				BatchRequestSizeLimitHTTP: defaultBatchRequestSizeLimitHTTP,
				BatchLogCountLimitHTTP:    defaultBatchLogCountLimitHTTP,
			},
			expectedErr: "location is required when protocol is https",
		},
		{
			desc: "Protocol is https and forwarder is empty",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				Protocol:                  protocolHTTPS,
				Compression:               noCompression,
				Project:                   "project_example",
				Location:                  "location_example",
				BatchRequestSizeLimitHTTP: defaultBatchRequestSizeLimitHTTP,
				BatchLogCountLimitHTTP:    defaultBatchLogCountLimitHTTP,
			},
			expectedErr: "forwarder is required when protocol is https",
		},
		{
			desc: "Protocol is https and project is empty",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				Protocol:                  protocolHTTPS,
				Compression:               noCompression,
				Location:                  "location_example",
				Forwarder:                 "forwarder_example",
				BatchRequestSizeLimitHTTP: defaultBatchRequestSizeLimitHTTP,
				BatchLogCountLimitHTTP:    defaultBatchLogCountLimitHTTP,
			},
			expectedErr: "project is required when protocol is https",
		},
		{
			desc: "Protocol is https and http batch log count limit is 0",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				Protocol:                  protocolHTTPS,
				Compression:               noCompression,
				Project:                   "project_example",
				Location:                  "location_example",
				Forwarder:                 "forwarder_example",
				BatchRequestSizeLimitHTTP: defaultBatchRequestSizeLimitHTTP,
				BatchLogCountLimitHTTP:    0,
			},
			expectedErr: "positive batch count log limit is required when protocol is https",
		},
		{
			desc: "Protocol is https and http batch request size limit is 0",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				Protocol:                  protocolHTTPS,
				Compression:               noCompression,
				Project:                   "project_example",
				Location:                  "location_example",
				Forwarder:                 "forwarder_example",
				BatchRequestSizeLimitHTTP: 0,
				BatchLogCountLimitHTTP:    defaultBatchLogCountLimitHTTP,
			},
			expectedErr: "positive batch request size limit is required when protocol is https",
		},
		{
			desc: "Valid https config",
			config: &Config{
				CredsFilePath:             "/path/to/creds_file",
				LogType:                   "log_type_example",
				Protocol:                  protocolHTTPS,
				Compression:               noCompression,
				Project:                   "project_example",
				Location:                  "location_example",
				Forwarder:                 "forwarder_example",
				BatchRequestSizeLimitHTTP: defaultBatchRequestSizeLimitHTTP,
				BatchLogCountLimitHTTP:    defaultBatchLogCountLimitHTTP,
			},
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
