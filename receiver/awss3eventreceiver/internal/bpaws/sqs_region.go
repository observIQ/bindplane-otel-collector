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

// Package bpaws provides a client for the SQS service.
package bpaws // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ParseRegionFromSQSURL parses the region from a SQS URL
func ParseRegionFromSQSURL(queueURL string) (string, error) {
	if queueURL == "" {
		return "", errors.New("SQS queue URL is required")
	}

	parsedURL, err := url.Parse(queueURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQS URL: %w", err)
	}

	// SQS URL format: https://sqs.{region}.amazonaws.com/{account}/{queue}
	hostParts := strings.Split(parsedURL.Host, ".")
	if len(hostParts) < 4 || hostParts[0] != "sqs" {
		return "", fmt.Errorf("invalid SQS URL format: %s", queueURL)
	}

	region := hostParts[1]

	// Validate that the region has a valid format
	validRegion := regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d+$`)
	if !validRegion.MatchString(region) {
		return "", fmt.Errorf("invalid region format in SQS URL: %s", region)
	}

	return region, nil
}
