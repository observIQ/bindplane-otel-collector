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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		config      *Config
		expectedErr error
	}{
		{
			desc: "Empty bucket name",
			config: &Config{
				BucketName:         "",
				ProjectID:          "test-project",
				Partition:          minutePartition,
				Compression:        noCompression,
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
			},
			expectedErr: errors.New("bucket_name is required"),
		},
		{
			desc: "Both credentials and credentials file provided",
			config: &Config{
				BucketName:         "test-bucket",
				ProjectID:          "test-project",
				Partition:          minutePartition,
				Compression:        noCompression,
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
				Credentials:        "{}",
				CredentialsFile:    "/path/to/creds.json",
			},
			expectedErr: errors.New("cannot specify both credentials and credentials_file"),
		},
		{
			desc: "Invalid partition",
			config: &Config{
				BucketName:         "test-bucket",
				ProjectID:          "test-project",
				Partition:          partitionType("nope"),
				Compression:        noCompression,
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
			},
			expectedErr: errors.New("unsupported partition type"),
		},
		{
			desc: "Invalid compression",
			config: &Config{
				BucketName:         "test-bucket",
				ProjectID:          "test-project",
				Partition:          minutePartition,
				Compression:        compressionType("bad"),
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
			},
			expectedErr: errors.New("unsupported compression type"),
		},
		{
			desc: "Valid minimal config",
			config: &Config{
				BucketName: "test-bucket",
				Partition:  minutePartition,
				Compression: noCompression,
			},
			expectedErr: nil,
		},
		{
			desc: "Valid config with hour partition",
			config: &Config{
				BucketName:         "test-bucket",
				ProjectID:          "test-project",
				Partition:          hourPartition,
				Compression:        noCompression,
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
			},
			expectedErr: nil,
		},
		{
			desc: "Valid config with gzip compression",
			config: &Config{
				BucketName:         "test-bucket",
				ProjectID:          "test-project",
				Partition:          minutePartition,
				Compression:        gzipCompression,
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
			},
			expectedErr: nil,
		},
		{
			desc: "Valid config with credentials JSON",
			config: &Config{
				BucketName:         "test-bucket",
				ProjectID:          "test-project",
				Partition:          minutePartition,
				Compression:        noCompression,
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
				Credentials:        "{}",
			},
			expectedErr: nil,
		},
		{
			desc: "Valid config with credentials file",
			config: &Config{
				BucketName:         "test-bucket",
				ProjectID:          "test-project",
				Partition:          minutePartition,
				Compression:        noCompression,
				BucketStorageClass: "STANDARD",
				BucketLocation:     "us-central1",
				CredentialsFile:    "/path/to/creds.json",
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		currentTC := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			err := currentTC.config.Validate()
			if currentTC.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, currentTC.expectedErr.Error())
			}
		})
	}
}
