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

package gcspubsubeventreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver"

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config defines the configuration for the GCS Pub/Sub Event receiver.
type Config struct {
	// ProjectID is the Google Cloud project ID that contains the Pub/Sub subscription.
	ProjectID string `mapstructure:"project_id"`

	// SubscriptionID is the Pub/Sub subscription ID that receives GCS event notifications.
	SubscriptionID string `mapstructure:"subscription_id"`

	// CredentialsFile is the path to the Google Cloud credentials JSON file.
	// If empty, Application Default Credentials will be used.
	CredentialsFile string `mapstructure:"credentials_file"`

	// Workers is the number of concurrent workers to process events.
	Workers int `mapstructure:"workers"`

	// MaxExtension is the maximum duration for which Pub/Sub will extend the ack deadline
	// for a message being processed. After this duration, the message will be nacked and
	// redelivered.
	MaxExtension time.Duration `mapstructure:"max_extension"`

	// MaxLogSize defines the maximum size in bytes for a single log record.
	// Logs exceeding this size will be split into chunks.
	// Default is 1MB.
	MaxLogSize int `mapstructure:"max_log_size"`

	// MaxLogsEmitted defines the maximum number of log records to emit in a single batch.
	// A higher number will result in fewer batches, but more memory usage.
	// Default is 1000.
	MaxLogsEmitted int `mapstructure:"max_logs_emitted"`

	// StorageID is the ID of the storage extension to use for storing the offset.
	StorageID *component.ID `mapstructure:"storage"`

	// BucketNameFilter is a regex filter to apply to the GCS bucket name.
	BucketNameFilter string `mapstructure:"bucket_name_filter"`

	// ObjectKeyFilter is a regex filter to apply to the GCS object name.
	ObjectKeyFilter string `mapstructure:"object_key_filter"`
}

// Validate checks if all required fields are present and valid.
func (c *Config) Validate() error {
	if c.ProjectID == "" {
		return errors.New("'project_id' is required")
	}

	if c.SubscriptionID == "" {
		return errors.New("'subscription_id' is required")
	}

	if c.Workers <= 0 {
		return errors.New("'workers' must be greater than 0")
	}

	if c.MaxExtension <= 0 {
		return errors.New("'max_extension' must be greater than 0")
	}

	if c.MaxLogSize <= 0 {
		return errors.New("'max_log_size' must be greater than 0")
	}

	if c.MaxLogsEmitted <= 0 {
		return errors.New("'max_logs_emitted' must be greater than 0")
	}

	if strings.TrimSpace(c.BucketNameFilter) != "" {
		_, err := regexp.Compile(c.BucketNameFilter)
		if err != nil {
			return fmt.Errorf("'bucket_name_filter' %q is invalid: %w", c.BucketNameFilter, err)
		}
	}

	if strings.TrimSpace(c.ObjectKeyFilter) != "" {
		_, err := regexp.Compile(c.ObjectKeyFilter)
		if err != nil {
			return fmt.Errorf("'object_key_filter' %q is invalid: %w", c.ObjectKeyFilter, err)
		}
	}

	return nil
}
