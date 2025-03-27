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

package googlecloudstoragerehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver"

import (
	"errors"
	"fmt"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"go.opentelemetry.io/collector/component"
)

// Config is the configuration for the Google Cloud Storage rehydration receiver
type Config struct {
	// BatchSize is the number of objects to process entering the pipeline in a single batch. (default 30)
	// This number directly affects the number of goroutines that will be created to process the objects.
	BatchSize int `mapstructure:"batch_size"`

	// BucketName is the name of the Google Cloud Storage bucket to pull from. (no default)
	BucketName string `mapstructure:"bucket_name"`

	// FolderName is the name of the folder in the bucket to pull from.
	FolderName string `mapstructure:"folder_name"`

	// StartingTime is the UTC timestamp to start rehydration from.
	StartingTime string `mapstructure:"starting_time"`

	// EndingTime is the UTC timestamp to rehydrate up until.
	EndingTime string `mapstructure:"ending_time"`

	// DeleteOnRead indicates if a file should be deleted once it has been processed (default false)
	DeleteOnRead bool `mapstructure:"delete_on_read"`

	// Credentials is the JSON credentials for Google Cloud Storage
	Credentials string `mapstructure:"credentials"`

	// CredentialsFile is the path to the credentials file for Google Cloud Storage
	CredentialsFile string `mapstructure:"credentials_file"`

	// ProjectID is the Google Cloud project ID
	ProjectID string `mapstructure:"project_id"`

	// ID of the storage extension to use for storing progress
	StorageID *component.ID `mapstructure:"storage"`
}

// Validate validates the config
func (c *Config) Validate() error {
	if c.BatchSize < 1 {
		return errors.New("batch_size must be greater than 0")
	}

	if c.BucketName == "" {
		return errors.New("bucket_name is required")
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
