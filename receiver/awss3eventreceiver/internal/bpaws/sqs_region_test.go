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

// Package bpaws_test provides tests for the bpaws package.
package bpaws_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
)

func TestParseRegion(t *testing.T) {
	tests := []struct {
		name        string
		sqsURL      string
		expected    string
		shouldError bool
	}{
		{
			name:        "Valid SQS URL US West 2",
			sqsURL:      "https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue",
			expected:    "us-west-2",
			shouldError: false,
		},
		{
			name:        "Valid SQS URL US East 1",
			sqsURL:      "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
			expected:    "us-east-1",
			shouldError: false,
		},
		{
			name:        "Valid SQS URL EU Central 1",
			sqsURL:      "https://sqs.eu-central-1.amazonaws.com/123456789012/MyQueue",
			expected:    "eu-central-1",
			shouldError: false,
		},
		{
			name:        "empty SQS URL",
			sqsURL:      "",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Invalid URL Format",
			sqsURL:      string([]byte{0x7f}),
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Invalid Host Format",
			sqsURL:      "https://invalid.host.com/queue",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Invalid SQS URL (missing sqs prefix)",
			sqsURL:      "https://not-sqs.us-west-2.amazonaws.com/123456789012/MyQueue",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Invalid Region Format",
			sqsURL:      "https://sqs.invalid-region.amazonaws.com/123456789012/MyQueue",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, err := bpaws.ParseRegionFromSQSURL(tt.sqsURL)
			if tt.shouldError {
				require.Error(t, err)
				assert.Empty(t, region)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, region)
			}
		})
	}
}
