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

package pebbleextension

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name          string
		config        func() *Config
		expectedError error
	}{
		{
			name: "valid config",
			config: func() *Config {
				return &Config{
					Directory: &DirectoryConfig{
						Path: t.TempDir(),
					},
					Compaction: &CompactionConfig{
						Interval:    1 * time.Minute,
						Concurrency: 3,
					},
				}
			},
		},
		{
			name: "valid config with sync enabled",
			config: func() *Config {
				return &Config{
					Directory: &DirectoryConfig{
						Path: t.TempDir(),
					},
					Sync: true,
					Compaction: &CompactionConfig{
						Interval:    1 * time.Minute,
						Concurrency: 3,
					},
				}
			},
		},
		{
			name: "nil directory",
			config: func() *Config {
				return &Config{
					Directory: nil,
					Compaction: &CompactionConfig{
						Interval:    1 * time.Minute,
						Concurrency: 3,
					},
				}
			},
			expectedError: errors.New("a directory path is required"),
		},
		{
			name: "empty directory path",
			config: func() *Config {
				return &Config{
					Directory: &DirectoryConfig{
						Path: "",
					},
					Compaction: &CompactionConfig{
						Interval:    1 * time.Minute,
						Concurrency: 3,
					},
				}
			},
			expectedError: errors.New("a directory path is required"),
		},
		{
			name: "negative cache size",
			config: func() *Config {
				return &Config{
					Cache: &CacheConfig{
						Size: -1,
					},
					Compaction: &CompactionConfig{
						Interval:    1 * time.Minute,
						Concurrency: 3,
					},
				}
			},
			expectedError: errors.New("cache size must be greater than or equal to 0"),
		},
		{
			name: "negative compaction interval",
			config: func() *Config {
				return &Config{
					Directory: &DirectoryConfig{
						Path: t.TempDir(),
					},
					Compaction: &CompactionConfig{
						Interval:    -1,
						Concurrency: 3,
					},
				}
			},
			expectedError: errors.New("compaction interval must be greater than or equal to 0"),
		},
		{
			name: "valid compaction config",
			config: func() *Config {
				return &Config{
					Directory: &DirectoryConfig{
						Path: t.TempDir(),
					},
					Compaction: &CompactionConfig{
						Interval:    1 * time.Minute,
						Concurrency: 3,
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config().Validate()
			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
