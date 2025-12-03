package pebbleextension

import "errors"

type Config struct {
	File *FileConfig `mapstructure:"file,omitempty"`
	Sync bool        `mapstructure:"sync"`
}

type FileConfig struct {
	Path string `mapstructure:"path"`
}

func (c *Config) Validate() error {
	if c.File == nil || c.File.Path == "" {
		return errors.New("a file path is required")
	}
	return nil
}
