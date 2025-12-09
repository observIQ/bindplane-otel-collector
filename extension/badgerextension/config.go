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
	"time"
)

// Config configures the badger storage extension
type Config struct {
	Directory             *DirectoryConfig             `mapstructure:"directory,omitempty"`
	SyncWrites            bool                         `mapstructure:"sync_writes"`
	Memory                *MemoryConfig                `mapstructure:"memory,omitempty"`
	BlobGarbageCollection *BlobGarbageCollectionConfig `mapstructure:"blob_garbage_collection,omitempty"`
}

// DirectoryConfig configures the file storage parameters for the badger storage extension
type DirectoryConfig struct {
	Path string `mapstructure:"path"`
}

// BlobGarbageCollectionConfig configures the blob garbage collection
type BlobGarbageCollectionConfig struct {
	Interval     time.Duration `mapstructure:"interval"`
	DiscardRatio float64       `mapstructure:"discard_ratio"`
}

// MemoryConfig configures the memory parameters for the badger storage extension; this is intended to be used
// for fine tuning if desired and should be considered an advanced configuration option.
type MemoryConfig struct {
	TableSize      int64 `mapstructure:"table_size"`
	BlockCacheSize int64 `mapstructure:"block_cache_size"`
}

// Validate validate the config
func (c *Config) Validate() error {
	if c.Directory == nil || c.Directory.Path == "" {
		return errors.New("a file path is required")
	}

	if c.BlobGarbageCollection != nil {
		if c.BlobGarbageCollection.Interval <= 0 {
			return errors.New("blob garbage collection interval must be greater than 0")
		}
		if c.BlobGarbageCollection.DiscardRatio <= 0 || c.BlobGarbageCollection.DiscardRatio >= 1 {
			return errors.New("blob garbage collection discard ratio must be between 0 and 1")
		}
	}

	if c.Memory != nil {
		if c.Memory.TableSize <= 0 {
			return errors.New("memory table size must be greater than 0")
		}
		if c.Memory.BlockCacheSize <= 0 {
			return errors.New("memory block cache size must be greater than 0")
		}
	}

	return nil
}
