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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				BucketName:    "test-bucket",
				StartingTime:  "2024-01-01T00:00:00Z",
				EndingTime:    "2024-01-02T00:00:00Z",
				BatchSize:     30,
				PageSize:      1000,
				DeleteOnRead:  false,
			},
			wantErr: false,
		},
		{
			name: "missing bucket name",
			config: &Config{
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    30,
				PageSize:     1000,
			},
			wantErr: true,
		},
		{
			name: "invalid starting time",
			config: &Config{
				BucketName:   "test-bucket",
				StartingTime: "invalid",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    30,
				PageSize:     1000,
			},
			wantErr: true,
		},
		{
			name: "invalid ending time",
			config: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "invalid",
				BatchSize:    30,
				PageSize:     1000,
			},
			wantErr: true,
		},
		{
			name: "ending time before starting time",
			config: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-02T00:00:00Z",
				EndingTime:   "2024-01-01T00:00:00Z",
				BatchSize:    30,
				PageSize:     1000,
			},
			wantErr: true,
		},
		{
			name: "ending time too close to starting time",
			config: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-01T00:00:30Z",
				BatchSize:    30,
				PageSize:     1000,
			},
			wantErr: true,
		},
		{
			name: "invalid batch size",
			config: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    0,
				PageSize:     1000,
			},
			wantErr: true,
		},
		{
			name: "invalid page size",
			config: &Config{
				BucketName:   "test-bucket",
				StartingTime: "2024-01-01T00:00:00Z",
				EndingTime:   "2024-01-02T00:00:00Z",
				BatchSize:    30,
				PageSize:     0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestValidateTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid timestamp",
			input:   "2024-01-01T00:00:00Z",
			wantErr: false,
		},
		{
			name:    "empty timestamp",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "2024-01-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := validateTimestamp(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ts)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, ts)

			// Verify the timestamp was parsed correctly
			expected, err := time.Parse(time.RFC3339, tt.input)
			require.NoError(t, err)
			assert.Equal(t, expected, *ts)
		})
	}
} 