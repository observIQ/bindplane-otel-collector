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

package macosunifiedloggingreceiver

import (
	"fmt"
	"os"
	"time"
)

// Config defines configuration for the macOS log command receiver
type Config struct {
	// ArchivePath points to a .logarchive directory to read from
	// If empty, reads from the live system logs
	ArchivePath string `mapstructure:"archive_path"`

	// Predicate is a filter predicate to pass to the log command
	// Example: "subsystem == 'com.apple.systempreferences'"
	Predicate string `mapstructure:"predicate"`

	// StartTime specifies when to start reading logs from
	// Format: "2006-01-02 15:04:05"
	StartTime string `mapstructure:"start_time"`

	// EndTime specifies when to stop reading logs
	// Only used with archive_path
	EndTime string `mapstructure:"end_time"`

	// PollInterval specifies how often to poll for new logs (live mode only)
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// MaxLogAge specifies the maximum age of logs to read on startup
	// Only applies to live mode. Format: "24h", "1h30m", etc.
	MaxLogAge time.Duration `mapstructure:"max_log_age"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Validate checks the Config is valid
func (cfg *Config) Validate() error {
	// Validate archive path if specified
	if cfg.ArchivePath != "" {
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
