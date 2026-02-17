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

package badgerextension

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
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				return cfg
			},
		},
		{
			name: "bad gc duration",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.BlobGarbageCollection = &BlobGarbageCollectionConfig{
					Interval: -1,
				}
				return cfg
			},
			expectedError: errors.New("blob garbage collection interval cannot be negative"),
		},
		{
			name: "bad gc discard ratio",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.BlobGarbageCollection = &BlobGarbageCollectionConfig{
					Interval:     1 * time.Minute,
					DiscardRatio: 1.1,
				}
				return cfg
			},
			expectedError: errors.New("blob garbage collection discard ratio must be between 0 and 1"),
		},
		{
			name: "empty file path",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: "",
				}
				return cfg
			},
			expectedError: errors.New("a file path for the directory is required"),
		},
		{
			name: "negative memory table size",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.Memory = &MemoryConfig{
					TableSize: -1,
				}
				return cfg
			},
			expectedError: errors.New("memory table size must not be negative"),
		},
		{
			name: "negative memory block cache size",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.Memory = &MemoryConfig{
					BlockCacheSize: -1,
				}
				return cfg
			},
			expectedError: errors.New("memory block cache size must not be negative"),
		},
		{
			name: "negative value log file size",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.Memory = &MemoryConfig{
					ValueLogFileSize: -1,
				}
				return cfg
			},
			expectedError: errors.New("value log file size must not be negative"),
		},
		{
			name: "negative number of compactors",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.Compaction = &CompactionConfig{
					NumCompactors: -1,
				}
				return cfg
			},
			expectedError: errors.New("number of compactors must not be negative"),
		},
		{
			name: "negative number of level zero tables",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.Compaction = &CompactionConfig{
					NumLevelZeroTables: -1,
				}
				return cfg
			},
			expectedError: errors.New("number of level zero tables must not be negative"),
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
				require.Equal(t, tt.expectedError, err)
			}
		})
	}
}
