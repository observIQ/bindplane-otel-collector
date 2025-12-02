package badgerextension

import (
	"errors"
	"time"
)

// Config configures the badger storage extension
type Config struct {
	File                  *FileConfig                  `mapstructure:"file,omitempty"`
	BlobGarbageCollection *BlobGarbageCollectionConfig `mapstructure:"blob_garbage_collection,omitempty"`
}

// FileConfig configures the file storage
type FileConfig struct {
	Path string `mapstructure:"path"`
}

// BlobGarbageCollectionConfig configures the blob garbage collection
type BlobGarbageCollectionConfig struct {
	Interval     time.Duration `mapstructure:"interval"`
	DiscardRatio float64       `mapstructure:"discard_ratio"`
}

// Validate validate the config
func (c *Config) Validate() error {
	if c.File == nil || c.File.Path == "" {
		return errors.New("a file path is required")
	}
	return nil
}
