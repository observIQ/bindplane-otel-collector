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

// Package pebbleextension provides a Pebble extension
package pebbleextension

import "errors"

// Config is the configuration for the pebble extension
type Config struct {
	Directory *DirectoryConfig `mapstructure:"directory,omitempty"`
	Cache     *CacheConfig     `mapstructure:"cache,omitempty"`
	Sync      bool             `mapstructure:"sync"`
}

// DirectoryConfig is the configuration for the directory
type DirectoryConfig struct {
	Path       string `mapstructure:"path"`
	PathPrefix string `mapstructure:"path_prefix"`
}

// CacheConfig is the configuration for the cache
type CacheConfig struct {
	Size int64 `mapstructure:"size"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errs error
	if c.Directory == nil || c.Directory.Path == "" {
		errs = errors.Join(errs, errors.New("a directory path is required"))
	}

	if c.Cache != nil {
		if err := c.Cache.validate(); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (c *CacheConfig) validate() error {
	if c.Size < 0 {
		return errors.New("cache size must be greater than or equal to 0")
	}
	return nil
}
