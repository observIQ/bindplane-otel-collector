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

package awss3rehydrationreceiver // import "github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver"

import (
	"errors"
	"fmt"
	"time" // timeFormat is the format for the starting and end time

	"github.com/observiq/bindplane-agent/internal/rehydration"
	"go.opentelemetry.io/collector/component"
)

// Config the configuration for the rehydration receiver
type Config struct {
	// Region AWS region
	Region string `mapstructure:"region"`

	// S3Bucket S3 Bucket to pull from
	S3Bucket string `mapstructure:"s3_bucket"`

	// S3Prefix prefix of S3 key (root directory inside bucket)
	S3Prefix string `mapstructure:"s3_prefix"`

	// StartingTime the UTC timestamp to start rehydration from.
	StartingTime string `mapstructure:"starting_time"`

	// EndingTime the UTC timestamp to rehydrate up until.
	EndingTime string `mapstructure:"ending_time"`

	// DeleteOnRead indicates if a file should be deleted once it has been processed
	// Default value of false
	DeleteOnRead bool `mapstructure:"delete_on_read"`

	// RoleArn the role ARN to be assumed
	RoleArn string `mapstructure:"role_arn"`

	// PollInterval is the interval at which the Azure API is scanned for blobs.
	// Default value of 1m
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// PollTimeout is the timeout for the Azure API to scan for blobs.
	PollTimeout time.Duration `mapstructure:"poll_timeout"`

	// ID of the storage extension to use for storing progress
	StorageID *component.ID `mapstructure:"storage"`
}

// Validate the configuration
func (c *Config) Validate() error {
	if c.Region == "" {
		return errors.New("region is required")
	}

	if c.S3Bucket == "" {
		return errors.New("s3_bucket is required")
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

	if c.PollInterval < time.Second {
		return errors.New("poll_interval must be at least one second")
	}

	if c.PollTimeout < time.Second {
		return errors.New("poll_timeout must be at least one second")
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
