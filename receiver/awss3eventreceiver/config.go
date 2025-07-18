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

package awss3eventreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver"

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/constants"
	"go.opentelemetry.io/collector/component"
)

// Config defines the configuration for the AWS S3 Event receiver.
type Config struct {
	// SQSQueueURL is the URL of the SQS queue that receives S3 event notifications.
	SQSQueueURL string `mapstructure:"sqs_queue_url"`

	// StandardPollInterval is the default interval to poll the SQS queue for messages.
	// When messages are found, polling will occur at this interval.
	StandardPollInterval time.Duration `mapstructure:"standard_poll_interval"`

	// MaxPollInterval is the maximum interval between SQS queue polls.
	// When no messages are found, the interval will increase up to this duration.
	MaxPollInterval time.Duration `mapstructure:"max_poll_interval"`

	// PollingBackoffFactor is the multiplier used to increase the polling interval
	// when no messages are found. For example, a value of 1.5 means the interval
	// increases by 50% after each empty poll.
	PollingBackoffFactor float64 `mapstructure:"polling_backoff_factor"`

	// Workers is the number of workers to use to process events.
	Workers int `mapstructure:"workers"`

	// VisibilityTimeout defines how long messages received from the queue will
	// be invisible to other consumers.
	VisibilityTimeout time.Duration `mapstructure:"visibility_timeout"`

	// VisibilityExtensionInterval defines how often to extend the visibility timeout
	// of messages being processed. This should be less than the VisibilityTimeout
	// to ensure messages don't become visible to other consumers while being processed.
	// Default is 1 minute.
	// Minimum is 10 seconds.
	VisibilityExtensionInterval time.Duration `mapstructure:"visibility_extension_interval"`

	// MaxVisibilityWindow defines the maximum total time a message can remain invisible
	// before it becomes visible to other consumers. This prevents messages from being
	// extended indefinitely. Must be less than or equal to SQS's 12-hour limit.
	MaxVisibilityWindow time.Duration `mapstructure:"max_visibility_window"`

	// MaxLogSize defines the maximum size in bytes for a single log record.
	// Logs exceeding this size will be split into chunks.
	// Default is 1MB.
	MaxLogSize int `mapstructure:"max_log_size"`

	// MaxLogsEmitted defines the maximum number of log records to emit in a single batch.
	// A higher number will result in fewer batches, but more memory usage.
	// Default is 1000.
	// TODO Allow 0 to represent no limit?
	MaxLogsEmitted int `mapstructure:"max_logs_emitted"`

	// StorageID is the ID of the storage extension to use for storing the offset.
	StorageID *component.ID `mapstructure:"storage"`

	// BucketNameFilter is a regex filter to apply to the S3 bucket name.
	BucketNameFilter string `mapstructure:"bucket_name_filter"`

	// ObjectKeyFilter is a regex filter to apply to the S3 object key.
	ObjectKeyFilter string `mapstructure:"object_key_filter"`

	// NotificationType specifies the format of notifications received in the SQS queue.
	// Valid values: "s3" (direct S3 events), "sns" (S3 events wrapped in SNS notifications).
	// Default is "s3".
	NotificationType string `mapstructure:"notification_type"`
}

// Validate checks if all required fields are present and valid.
func (c *Config) Validate() error {
	if c.SQSQueueURL == "" {
		return errors.New("'sqs_queue_url' is required")
	}

	if c.StandardPollInterval <= 0 {
		return errors.New("'standard_poll_interval' must be greater than 0")
	}

	if c.MaxPollInterval <= c.StandardPollInterval {
		return errors.New("'max_poll_interval' must be greater than 'standard_poll_interval'")
	}

	if c.PollingBackoffFactor <= 1 {
		return errors.New("'polling_backoff_factor' must be greater than 1")
	}

	if c.VisibilityTimeout <= 0 {
		return errors.New("'visibility_timeout' must be greater than 0")
	}

	if c.VisibilityExtensionInterval <= 0 {
		return errors.New("'visibility_extension_interval' must be greater than 0")
	}

	if c.VisibilityExtensionInterval > c.VisibilityTimeout {
		return errors.New("'visibility_extension_interval' must be less than 'visibility_timeout'")
	}

	if c.VisibilityExtensionInterval < 10*time.Second {
		return errors.New("'visibility_extension_interval' must be greater than 10 seconds")
	}

	if c.MaxVisibilityWindow <= 0 {
		return errors.New("'max_visibility_window' must be greater than 0")
	}

	if c.MaxVisibilityWindow <= c.VisibilityTimeout {
		return errors.New("'max_visibility_window' must be greater than 'visibility_timeout'")
	}

	// SQS has a 12-hour limit
	maxAllowedWindow := 12 * time.Hour
	if c.MaxVisibilityWindow > maxAllowedWindow {
		return errors.New("'max_visibility_window' must be less than or equal to 12 hours")
	}

	if c.Workers <= 0 {
		return errors.New("'workers' must be greater than 0")
	}

	if c.MaxLogSize <= 0 {
		return errors.New("'max_log_size' must be greater than 0")
	}

	if _, err := client.ParseRegionFromSQSURL(c.SQSQueueURL); err != nil {
		return err
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

	return c.validateNotificationType()
}

func (c *Config) validateNotificationType() error {
	// Set default if not specified
	if c.NotificationType == "" {
		c.NotificationType = constants.NotificationTypeS3
	}

	// Validate notification type
	switch c.NotificationType {
	case constants.NotificationTypeS3, constants.NotificationTypeSNS:
		// Valid types
	default:
		return fmt.Errorf("invalid notification_type '%s': must be 's3' or 'sns'", c.NotificationType)
	}

	return nil
}
