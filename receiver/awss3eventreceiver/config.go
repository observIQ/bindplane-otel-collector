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

	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
)

// Config defines the configuration for the AWS S3 Event receiver.
type Config struct {
	// SQSQueueURL is the URL of the SQS queue that receives S3 event notifications.
	SQSQueueURL string `mapstructure:"sqs_queue_url"`

	// PollInterval is the interval at which to poll the SQS queue for messages.
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// Workers is the number of workers to use to process events.
	Workers int `mapstructure:"workers"`

	// VisibilityTimeout defines how long messages received from the queue will
	// be invisible to other consumers.
	VisibilityTimeout time.Duration `mapstructure:"visibility_timeout"`

	// APIMaxMessages defines the maximum number of messages to request from
	// SQS at once.
	APIMaxMessages int32 `mapstructure:"api_max_messages"`
}

// Validate checks if all required fields are present and valid.
func (c *Config) Validate() error {
	if c.SQSQueueURL == "" {
		return errors.New("'sqs_queue_url' is required")
	}

	if c.PollInterval <= 0 {
		return errors.New("'poll_interval' must be greater than 0")
	}

	if c.VisibilityTimeout <= 0 {
		return errors.New("'visibility_timeout' must be greater than 0")
	}

	if c.APIMaxMessages <= 0 {
		return errors.New("'api_max_messages' must be greater than 0")
	}

	if c.Workers <= 0 {
		return errors.New("'workers' must be greater than 0")
	}

	if _, err := bpaws.ParseRegionFromSQSURL(c.SQSQueueURL); err != nil {
		return err
	}

	return nil
}
