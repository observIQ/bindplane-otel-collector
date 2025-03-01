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
	"net/url"
	"regexp"
	"strings"
	"time"
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
		return errors.New("sqs_queue_url is required")
	}

	if c.PollInterval <= 0 {
		return errors.New("poll_interval must be greater than 0")
	}

	if c.VisibilityTimeout <= 0 {
		return errors.New("visibility_timeout must be greater than 0")
	}

	if c.APIMaxMessages <= 0 {
		return errors.New("api_max_messages must be greater than 0")
	}

	// Validate that SQS URL has a valid region
	_, err := c.GetRegion()
	if err != nil {
		return err
	}

	return nil
}

// GetRegion extracts the AWS region from the SQS queue URL.
// TODO internal/bpssqs/ParseRegion(url string) (string, error)
func (c *Config) GetRegion() (string, error) {
	if c.SQSQueueURL == "" {
		return "", errors.New("SQS queue URL is required")
	}

	parsedURL, err := url.Parse(c.SQSQueueURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQS URL: %w", err)
	}

	// SQS URL format: https://sqs.{region}.amazonaws.com/{account}/{queue}
	hostParts := strings.Split(parsedURL.Host, ".")
	if len(hostParts) < 4 || hostParts[0] != "sqs" {
		return "", fmt.Errorf("invalid SQS URL format: %s", c.SQSQueueURL)
	}

	region := hostParts[1]

	// Validate that the region has a valid format
	validRegion := regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d+$`)
	if !validRegion.MatchString(region) {
		return "", fmt.Errorf("invalid region format in SQS URL: %s", region)
	}

	return region, nil
}
