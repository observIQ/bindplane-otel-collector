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

package awss3eventreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestValidConfig(t *testing.T) {
	t.Parallel()

	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	cfg.SQSQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"

	assert.NoError(t, cfg.Validate())
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	tests := []struct {
		id          component.ID
		expected    component.Config
		expectError bool
	}{
		{
			id:          component.NewID(Type),
			expected:    createDefaultConfig(),
			expectError: true, // Default config doesn't have required fields
		},
		{
			id: component.NewIDWithName(Type, "custom"),
			expected: &Config{
				SQSQueueURL:       "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
				PollInterval:      30 * time.Second,
				VisibilityTimeout: 600 * time.Second,
				APIMaxMessages:    20,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()

			// Get the receivers section from the confmap
			receiversMap, err := cm.Sub("receivers")
			require.NoError(t, err)

			// Get the specific receiver config
			sub, err := receiversMap.Sub(tt.id.String())
			require.NoError(t, err)

			require.NoError(t, sub.Unmarshal(cfg))

			if tt.expectError {
				require.Error(t, cfg.(*Config).Validate())
				return
			}

			require.NoError(t, cfg.(*Config).Validate())
			if cfgS3, ok := cfg.(*Config); ok {
				assert.Equal(t, tt.expected, cfgS3)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		cfg         *Config
		expectedErr string
	}{
		{
			desc: "Valid config",
			cfg: &Config{
				SQSQueueURL:       "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue",
				PollInterval:      30 * time.Second,
				VisibilityTimeout: 600 * time.Second,
				APIMaxMessages:    10,
			},
			expectedErr: "",
		},
		{
			desc: "Missing SQS queue URL",
			cfg: &Config{
				PollInterval:      30 * time.Second,
				VisibilityTimeout: 600 * time.Second,
				APIMaxMessages:    10,
			},
			expectedErr: "sqs_queue_url is required",
		},
		{
			desc: "Invalid poll interval",
			cfg: &Config{
				SQSQueueURL:       "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue",
				PollInterval:      0,
				VisibilityTimeout: 600 * time.Second,
				APIMaxMessages:    10,
			},
			expectedErr: "poll_interval must be greater than 0",
		},
		{
			desc: "Invalid visibility timeout",
			cfg: &Config{
				SQSQueueURL:       "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue",
				PollInterval:      30 * time.Second,
				VisibilityTimeout: 0,
				APIMaxMessages:    10,
			},
			expectedErr: "visibility_timeout must be greater than 0",
		},
		{
			desc: "Invalid API max messages",
			cfg: &Config{
				SQSQueueURL:       "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue",
				PollInterval:      30 * time.Second,
				VisibilityTimeout: 600 * time.Second,
				APIMaxMessages:    0,
			},
			expectedErr: "api_max_messages must be greater than 0",
		},
		{
			desc: "Invalid max workers",
			cfg: &Config{
				SQSQueueURL:       "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue",
				PollInterval:      30 * time.Second,
				VisibilityTimeout: 600 * time.Second,
				APIMaxMessages:    10,
			},
			expectedErr: "max_workers must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractRegionFromSQSURL(t *testing.T) {
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
			name:        "Invalid URL Format",
			sqsURL:      "invalid-url",
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
			// Create a config with the test SQS URL
			cfg := Config{
				SQSQueueURL: tt.sqsURL,
			}

			// Use the GetRegion method
			region, err := cfg.GetRegion()

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

func TestConfigValidateRegionExtraction(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
	}{
		{
			name: "Valid Config with Valid SQS URL",
			config: Config{
				SQSQueueURL:       "https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue",
				PollInterval:      20,
				VisibilityTimeout: 300,
				APIMaxMessages:    10,
			},
			shouldError: false,
		},
		{
			name: "Valid Config with Different Region",
			config: Config{
				SQSQueueURL:       "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
				PollInterval:      20,
				VisibilityTimeout: 300,
				APIMaxMessages:    10,
			},
			shouldError: false,
		},
		{
			name: "Invalid SQS URL for Region Extraction",
			config: Config{
				SQSQueueURL:       "https://invalid-url",
				PollInterval:      20,
				VisibilityTimeout: 300,
				APIMaxMessages:    10,
			},
			shouldError: true,
		},
		{
			name: "Missing SQS URL",
			config: Config{
				SQSQueueURL:       "", // Missing URL
				PollInterval:      20,
				VisibilityTimeout: 300,
				APIMaxMessages:    10,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Check if we can extract the region from the SQS URL
				if tt.config.SQSQueueURL != "" {
					region, regErr := tt.config.GetRegion()
					if !tt.shouldError {
						assert.NoError(t, regErr)
						assert.NotEmpty(t, region)
					}
				}
			}
		})
	}
}
