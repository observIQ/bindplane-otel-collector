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

package azureblobpollingreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver"

import (
	"errors"
	"regexp"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config is the configuration for the azure blob polling receiver
type Config struct {
	// BatchSize is the number of blobs to process entering the pipeline in a single batch. (default 30)
	// This number directly affects the number of goroutines that will be created to process the blobs.
	BatchSize int `mapstructure:"batch_size"`

	// ConnectionString is the Azure Blob Storage connection key,
	// which can be found in the Azure Blob Storage resource on the Azure Portal. (no default)
	ConnectionString string `mapstructure:"connection_string"`

	// Container is the name of the storage container to pull from. (no default)
	Container string `mapstructure:"container"`

	// RootFolder is the name of the root folder in path.
	RootFolder string `mapstructure:"root_folder"`

	// PollInterval is the interval at which to poll for new blobs. (no default, required)
	// The receiver will continuously poll at this interval and dynamically adjust the time window
	// to collect only new data from each interval.
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// InitialLookback is the duration to look back on the first poll when no checkpoint exists. (default: same as poll_interval)
	// For example, if set to 1h, on first startup the receiver will look for blobs from the last hour.
	InitialLookback time.Duration `mapstructure:"initial_lookback"`

	// DeleteOnRead indicates if a file should be deleted once it has been processed
	// Default value of false
	DeleteOnRead bool `mapstructure:"delete_on_read"`

	// PageSize is the number of blobs to request from the Azure API at a time. (default 1000)
	PageSize int `mapstructure:"page_size"`

	// ID of the storage extension to use for storing progress
	StorageID *component.ID `mapstructure:"storage"`

	// UseLastModified when true, uses the blob's LastModified timestamp instead of parsing the folder structure
	// This allows collecting blobs that don't follow the year=/month=/day=/hour= naming convention
	// Default is false to maintain backward compatibility
	UseLastModified bool `mapstructure:"use_last_modified"`

	// TimePattern specifies a custom pattern for extracting timestamps from blob paths
	// Supports both named placeholders ({year}/{month}/{day}/{hour}/{minute}) and Go time format (2006/01/02/15/04)
	// Examples:
	//   "{year}/{month}/{day}/{hour}" matches "2025/12/05/14/app.log"
	//   "logs/{year}-{month}-{day}" matches "logs/2025-12-05/data.json"
	//   "2006/01/02/15/04" matches "2025/12/05/14/28/logs.json"
	// If not specified, uses the default year=YYYY/month=MM/... format
	TimePattern string `mapstructure:"time_pattern"`

	// TelemetryType explicitly sets the telemetry type ("logs", "metrics", or "traces")
	// Required when using time_pattern, as the receiver can't infer type from the path
	// If not set, falls back to the pipeline type the receiver is configured in
	TelemetryType string `mapstructure:"telemetry_type"`

	// FilenamePattern is a regex pattern to filter blobs by filename
	// Only blobs whose names match this pattern will be processed
	// Examples:
	//   "firewall\\d+_\\w+\\.json" matches "firewall43_dfreds.json"
	//   ".*\\.json" matches any file ending with .json
	//   "app-.*\\.log" matches "app-server.log", "app-client.log"
	// If not specified, all blobs matching the time pattern will be processed
	FilenamePattern string `mapstructure:"filename_pattern"`
}

// Validate validates the config
func (c *Config) Validate() error {
	if c.BatchSize < 1 {
		return errors.New("batch_size must be greater than 0")
	}

	if c.ConnectionString == "" {
		return errors.New("connection_string is required")
	}

	if c.Container == "" {
		return errors.New("container is required")
	}

	if c.PollInterval <= 0 {
		return errors.New("poll_interval must be greater than 0")
	}

	if c.PollInterval < time.Minute {
		return errors.New("poll_interval must be at least 1 minute")
	}

	if c.InitialLookback < 0 {
		return errors.New("initial_lookback must be greater than or equal to 0")
	}

	if c.PageSize < 1 {
		return errors.New("page_size must be greater than 0")
	}

	// Validate use_last_modified and time_pattern are not both set
	if c.UseLastModified && c.TimePattern != "" {
		return errors.New("use_last_modified and time_pattern cannot both be set")
	}

	// Validate telemetry_type if set
	if c.TelemetryType != "" {
		switch c.TelemetryType {
		case "logs", "metrics", "traces":
			// valid
		default:
			return errors.New("telemetry_type must be one of: logs, metrics, traces")
		}
	}

	// Validate filename_pattern is a valid regex if set
	if c.FilenamePattern != "" {
		_, err := regexp.Compile(c.FilenamePattern)
		if err != nil {
			return errors.New("filename_pattern must be a valid regex: " + err.Error())
		}
	}

	return nil
}
