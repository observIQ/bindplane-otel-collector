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

package gcspubsubeventreceiver_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/metadata"
)

func TestValidConfig(t *testing.T) {
	t.Parallel()

	f := gcspubsubeventreceiver.NewFactory()
	cfg := f.CreateDefaultConfig().(*gcspubsubeventreceiver.Config)
	cfg.ProjectID = "test-project"
	cfg.SubscriptionID = "test-subscription"

	assert.NoError(t, cfg.Validate())
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "test_config.yaml"))
	require.NoError(t, err)

	tests := []struct {
		id          component.ID
		expected    component.Config
		expectError bool
	}{
		{
			id:          component.NewID(metadata.Type),
			expected:    gcspubsubeventreceiver.NewFactory().CreateDefaultConfig(),
			expectError: true, // Default config doesn't have required fields
		},
		{
			id: component.NewIDWithName(metadata.Type, "custom"),
			expected: &gcspubsubeventreceiver.Config{
				ProjectID:      "my-gcp-project",
				SubscriptionID: "my-gcs-events-sub",
				Workers:        10,
				MaxExtension:   2 * time.Hour,
				PollInterval:   500 * time.Millisecond,
				DedupTTL:       10 * time.Minute,
				MaxLogSize:     4096,
				MaxLogsEmitted: 500,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			factory := gcspubsubeventreceiver.NewFactory()
			cfg := factory.CreateDefaultConfig()

			// Get the receivers section from the confmap
			receiversMap, err := cm.Sub("receivers")
			require.NoError(t, err)

			// Get the specific receiver config
			sub, err := receiversMap.Sub(tt.id.String())
			require.NoError(t, err)

			require.NoError(t, sub.Unmarshal(cfg))

			if tt.expectError {
				require.Error(t, cfg.(*gcspubsubeventreceiver.Config).Validate())
				return
			}

			require.NoError(t, cfg.(*gcspubsubeventreceiver.Config).Validate())
			if cfgGCS, ok := cfg.(*gcspubsubeventreceiver.Config); ok {
				assert.Equal(t, tt.expected, cfgGCS)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		cfgMod      func(*gcspubsubeventreceiver.Config)
		expectedErr string
	}{
		{
			desc: "Valid config",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
			},
			expectedErr: "",
		},
		{
			desc:        "Missing project ID",
			cfgMod:      func(_ *gcspubsubeventreceiver.Config) {},
			expectedErr: "'project_id' is required",
		},
		{
			desc: "Missing subscription ID",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
			},
			expectedErr: "'subscription_id' is required",
		},
		{
			desc: "Invalid workers",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.Workers = -1
			},
			expectedErr: "'workers' must be greater than 0",
		},
		{
			desc: "Invalid max extension",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.MaxExtension = 0
			},
			expectedErr: "'max_extension' must be greater than 0",
		},
		{
			desc: "Invalid poll interval",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.PollInterval = 0
			},
			expectedErr: "'poll_interval' must be greater than 0",
		},
		{
			desc: "Invalid dedup TTL",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.DedupTTL = 0
			},
			expectedErr: "'dedup_ttl' must be greater than 0",
		},
		{
			desc: "Invalid max log size",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.MaxLogSize = 0
			},
			expectedErr: "'max_log_size' must be greater than 0",
		},
		{
			desc: "Invalid max logs emitted",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.MaxLogsEmitted = 0
			},
			expectedErr: "'max_logs_emitted' must be greater than 0",
		},
		{
			desc: "Invalid bucket name filter regex",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.BucketNameFilter = "[invalid"
			},
			expectedErr: "'bucket_name_filter' \"[invalid\" is invalid:",
		},
		{
			desc: "Invalid object key filter regex",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.ObjectKeyFilter = "[invalid"
			},
			expectedErr: "'object_key_filter' \"[invalid\" is invalid:",
		},
		{
			desc: "Valid config with filters",
			cfgMod: func(cfg *gcspubsubeventreceiver.Config) {
				cfg.ProjectID = "test-project"
				cfg.SubscriptionID = "test-subscription"
				cfg.BucketNameFilter = "^my-bucket-.*"
				cfg.ObjectKeyFilter = ".*\\.json$"
			},
			expectedErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			f := gcspubsubeventreceiver.NewFactory()
			cfg := f.CreateDefaultConfig().(*gcspubsubeventreceiver.Config)
			tc.cfgMod(cfg)
			err := cfg.Validate()
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
