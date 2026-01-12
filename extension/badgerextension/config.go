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
	Compaction            *CompactionConfig            `mapstructure:"compaction,omitempty"`
	BlobGarbageCollection *BlobGarbageCollectionConfig `mapstructure:"blob_garbage_collection,omitempty"`
	Telemetry             *TelemetryConfig             `mapstructure:"telemetry,omitempty"`

	_ struct{} // prevent unkeyed literal initialization
}

// DirectoryConfig configures the file storage parameters for the badger storage extension
type DirectoryConfig struct {
	Path       string   `mapstructure:"path"`
	PathPrefix string   `mapstructure:"path_prefix"`
	_          struct{} // prevent unkeyed literal initialization
}

// BlobGarbageCollectionConfig configures the blob garbage collection
type BlobGarbageCollectionConfig struct {
	Interval     time.Duration `mapstructure:"interval"`
	DiscardRatio float64       `mapstructure:"discard_ratio"`
	_            struct{}      // prevent unkeyed literal initialization
}

// MemoryConfig configures the memory parameters for the badger storage extension; this is intended to be used
// for fine tuning if desired and should be considered an advanced configuration option.
type MemoryConfig struct {
	TableSize        int64    `mapstructure:"table_size"`
	BlockCacheSize   int64    `mapstructure:"block_cache_size"`
	ValueLogFileSize int64    `mapstructure:"value_log_file_size"`
	_                struct{} // prevent unkeyed literal initialization
}

// CompactionConfig configures LSM tree compaction behavior for more aggressive cleanup of deleted data.
type CompactionConfig struct {
	// NumCompactors is the number of background compaction workers. Higher values = more aggressive compaction.
	NumCompactors int `mapstructure:"num_compactors"`
	// NumLevelZeroTables is the number of L0 tables that triggers compaction. Lower values = earlier compaction.
	NumLevelZeroTables int `mapstructure:"num_level_zero_tables"`
	// NumLevelZeroTablesStall is the number of L0 tables that stalls writes until compaction catches up.
	NumLevelZeroTablesStall int      `mapstructure:"num_level_zero_tables_stall"`
	_                       struct{} // prevent unkeyed literal initialization
}

// TelemetryConfig configures the telemetry parameters for the badger storage extension
type TelemetryConfig struct {
	// whether or not to enable telemetry
	Enabled bool `mapstructure:"enabled"`

	// the interval at which to update the telemetry
	UpdateInterval time.Duration `mapstructure:"update_interval"`
	_              struct{}      // prevent unkeyed literal initialization
}

// Validate validate the config
func (c *Config) Validate() error {
	if c.Directory == nil || c.Directory.Path == "" {
		return errors.New("a file path for the directory is required")
	}

	if c.BlobGarbageCollection != nil {
		if c.BlobGarbageCollection.Interval < 0 {
			return errors.New("blob garbage collection interval cannot be negative")
		}
		if c.BlobGarbageCollection.DiscardRatio <= 0 || c.BlobGarbageCollection.DiscardRatio >= 1 {
			return errors.New("blob garbage collection discard ratio must be between 0 and 1")
		}
	}

	if c.Memory != nil {
		if c.Memory.TableSize < 0 {
			return errors.New("memory table size must not be negative")
		}
		if c.Memory.BlockCacheSize < 0 {
			return errors.New("memory block cache size must not be negative")
		}
		if c.Memory.ValueLogFileSize < 0 {
			return errors.New("value log file size must not be negative")
		}
	}

	if c.Compaction != nil {
		if c.Compaction.NumCompactors < 0 {
			return errors.New("number of compactors must not be negative")
		}
		if c.Compaction.NumLevelZeroTables < 0 {
			return errors.New("number of level zero tables must not be negative")
		}
		if c.Compaction.NumLevelZeroTablesStall < 0 {
			return errors.New("number of level zero tables stall must not be negative")
		}
	}

	return nil
}
