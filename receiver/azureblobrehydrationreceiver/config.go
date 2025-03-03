// Copyright Bindplane, Inc.
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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver"

import (
	"errors"
	"fmt"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"go.opentelemetry.io/collector/component"
)

// Config is the configuration for the azure blob rehydration receiver
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

	// StartingTime the UTC timestamp to start rehydration from.
	StartingTime string `mapstructure:"starting_time"`

	// EndingTime the UTC timestamp to rehydrate up until.
	EndingTime string `mapstructure:"ending_time"`

	// DeleteOnRead indicates if a file should be deleted once it has been processed
	// Default value of false
	DeleteOnRead bool `mapstructure:"delete_on_read"`

	// Deprecated: PollInterval is no longer required due to streaming blobs for processing.
	// if a value is provided, validation will throw an error
	// PollInterval is the interval at which the Azure API is scanned for blobs.
	// Default value of 1m
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// Deprecated: PollTimeout is no longer required due to streaming blobs for processing.
	// if a value is provided, validation will throw an error
	// PollTimeout is the timeout for the Azure API to scan for blobs.
	PollTimeout time.Duration `mapstructure:"poll_timeout"`

	// PageSize is the number of blobs to request from the Azure API at a time. (default 1000)
	PageSize int `mapstructure:"page_size"`

	// ID of the storage extension to use for storing progress
	StorageID *component.ID `mapstructure:"storage"`
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

	startingTs, err := validateTimestamp(c.StartingTime)
	if err != nil {
		return fmt.Errorf("starting_time is invalid: %w", err)
	}

	endingTs, err := validateTimestamp(c.EndingTime)
	if err != nil {
		return fmt.Errorf("ending_time is invalid: %w", err)
	}

	// Check case where ending_time is to close or before starting time
	if endingTs.Sub(*startingTs) < time.Minute {
		return errors.New("ending_time must be at least one minute after starting_time")
	}

	if c.PageSize < 1 {
		return errors.New("page_size must be greater than 0")
	}

	return nil
}

// validateTimestamp validates the passed in timestamp string
func validateTimestamp(timestamp string) (*time.Time, error) {
	if timestamp == "" {
		return nil, errors.New("missing value")
	}

	ts, err := time.Parse(rehydration.TimeFormat, timestamp)
	if err != nil {
		return nil, errors.New("invalid timestamp format must be in the form YYYY-MM-DDTHH:MM")
	}

	return &ts, nil
}
