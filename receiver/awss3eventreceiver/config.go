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
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
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

	// MaxLogSize defines the maximum size in bytes for a single log record.
	// Logs exceeding this size will be split into chunks.
	// Default is 1MB.
	MaxLogSize int `mapstructure:"max_log_size"`

	// MaxLogsEmitted defines the maximum number of log records to emit in a single batch.
	// A higher number will result in fewer batches, but more memory usage.
	// Default is 1000.
	// TODO Allow 0 to represent no limit?
	MaxLogsEmitted int `mapstructure:"max_logs_emitted"`
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

	if c.Workers <= 0 {
		return errors.New("'workers' must be greater than 0")
	}

	if c.MaxLogSize <= 0 {
		return errors.New("'max_log_size' must be greater than 0")
	}

	if _, err := client.ParseRegionFromSQSURL(c.SQSQueueURL); err != nil {
		return err
	}

	return nil
}
