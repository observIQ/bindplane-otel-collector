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

package awss3eventextension

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/metadata"
)

func TestValidConfig(t *testing.T) {
	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	cfg.SQSQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
	cfg.Directory = "/tmp/s3event"

	assert.NoError(t, cfg.Validate())
}

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "test_config.yaml"))
	require.NoError(t, err)

	tests := []struct {
		id          component.ID
		expected    component.Config
		expectError bool
	}{
		{
			id:          component.NewID(metadata.Type),
			expected:    createDefaultConfig(),
			expectError: true, // Default config doesn't have required fields
		},
		{
			id: component.NewIDWithName(metadata.Type, "s3"),
			expected: &Config{
				SQSQueueURL:          "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
				StandardPollInterval: 30 * time.Second,
				MaxPollInterval:      60 * time.Second,
				PollingBackoffFactor: 2,
				VisibilityTimeout:    600 * time.Second,
				Workers:              5,
				EventFormat:          "aws_s3",
				Directory:            "/tmp/s3event",
			},
			expectError: false,
		},
		{
			id: component.NewIDWithName(metadata.Type, "fdr"),
			expected: &Config{
				SQSQueueURL:          "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
				StandardPollInterval: 30 * time.Second,
				MaxPollInterval:      60 * time.Second,
				PollingBackoffFactor: 2,
				VisibilityTimeout:    600 * time.Second,
				Workers:              5,
				EventFormat:          "fdr",
				Directory:            "/tmp/s3event",
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
		cfgMod      func(*Config)
		expectedErr string
	}{
		{
			desc: "Valid config",
			cfgMod: func(cfg *Config) {
				cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
				cfg.Directory = "/tmp/s3event"
			},
			expectedErr: "",
		},
		{
			desc:        "Missing SQS queue URL",
			cfgMod:      func(_ *Config) {},
			expectedErr: "'sqs_queue_url' is required",
		},
		{
			desc: "Invalid poll interval",
			cfgMod: func(cfg *Config) {
				cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
				cfg.StandardPollInterval = 0
			},
			expectedErr: "'standard_poll_interval' must be greater than 0",
		},
		{
			desc: "Invalid max poll interval",
			cfgMod: func(cfg *Config) {
				cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
				cfg.StandardPollInterval = 15 * time.Second
				cfg.MaxPollInterval = 10 * time.Second
			},
			expectedErr: "'max_poll_interval' must be greater than 'standard_poll_interval'",
		},
		{
			desc: "Invalid polling backoff factor",
			cfgMod: func(cfg *Config) {
				cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
				cfg.StandardPollInterval = 15 * time.Second
				cfg.MaxPollInterval = 60 * time.Second
				cfg.PollingBackoffFactor = 1
			},
			expectedErr: "'polling_backoff_factor' must be greater than 1",
		},
		{
			desc: "Invalid visibility timeout",
			cfgMod: func(cfg *Config) {
				cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
				cfg.VisibilityTimeout = 0
			},
			expectedErr: "'visibility_timeout' must be greater than 0",
		},
		{
			desc: "Invalid workers",
			cfgMod: func(cfg *Config) {
				cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
				cfg.Workers = -1
			},
			expectedErr: "'workers' must be greater than 0",
		},
		{
			desc: "Invalid directory",
			cfgMod: func(cfg *Config) {
				cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
				cfg.Directory = ""
			},
			expectedErr: "'directory' is required",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			f := NewFactory()
			cfg := f.CreateDefaultConfig().(*Config)
			tc.cfgMod(cfg)
			err := cfg.Validate()
			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
