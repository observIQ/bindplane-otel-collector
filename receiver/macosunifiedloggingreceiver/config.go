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

//go:build darwin

package macosunifiedloggingreceiver

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Validate checks the Config is valid
func (cfg *Config) Validate() error {

	// Set default format if not specified
	if cfg.Format == "" {
		cfg.Format = "default"
	}

	// Validate format
	validFormats := map[string]bool{
		"default": true,
		"ndjson":  true,
		"json":    true,
		"syslog":  true,
		"compact": true,
	}
	if !validFormats[cfg.Format] {
		return fmt.Errorf("invalid format: %s (valid options: default, ndjson, json, syslog, compact)", cfg.Format)
	}

	// Validate archive path if specified
	if cfg.ArchivePath != "" {
		// sanitize the archive path
		cfg.ArchivePath = filepath.Clean(cfg.ArchivePath)

		info, err := os.Stat(cfg.ArchivePath)
		if err != nil {
			return fmt.Errorf("archive_path does not exist: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("archive_path must be a directory (.logarchive)")
		}
	}

	// Validate time format if specified
	if cfg.StartTime != "" {
		if _, err := time.Parse("2006-01-02 15:04:05", cfg.StartTime); err != nil {
			return fmt.Errorf("invalid start_time format (expected: 2006-01-02 15:04:05): %w", err)
		}
	}

	if cfg.EndTime != "" {
		if cfg.ArchivePath == "" {
			return fmt.Errorf("end_time can only be used with archive_path")
		}
		if _, err := time.Parse("2006-01-02 15:04:05", cfg.EndTime); err != nil {
			return fmt.Errorf("invalid end_time format (expected: 2006-01-02 15:04:05): %w", err)
		}
	}

	return nil
}
