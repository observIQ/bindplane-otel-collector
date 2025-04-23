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

package awss3eventextension_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"

	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension"
	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/metadata"
)

// Test that the factory creates the default configuration correctly
func TestFactoryCreateDefaultConfig(t *testing.T) {
	factory := awss3eventextension.NewFactory()
	cfg := factory.CreateDefaultConfig()

	assert.Equal(t, metadata.Type, factory.Type())
	assert.Equal(t, "s3event", metadata.Type.String())
	assert.NotNil(t, cfg)

	assert.NoError(t, componenttest.CheckConfigStruct(cfg))

	extCfg, ok := cfg.(*awss3eventextension.Config)
	require.True(t, ok)
	assert.Equal(t, "", extCfg.SQSQueueURL)
	assert.Equal(t, 15*time.Second, extCfg.StandardPollInterval)
	assert.Equal(t, 120*time.Second, extCfg.MaxPollInterval)
	assert.Equal(t, 2.0, extCfg.PollingBackoffFactor)
	assert.Equal(t, 300*time.Second, extCfg.VisibilityTimeout)
	assert.Equal(t, 5, extCfg.Workers)
	assert.Equal(t, "aws_s3", extCfg.EventFormat)
	assert.Equal(t, "", extCfg.Directory)
}

// Test factory extension creation methods
func TestFactoryStability(t *testing.T) {
	factory := awss3eventextension.NewFactory()
	assert.Equal(t, component.StabilityLevelAlpha, factory.Stability())
}

// Test factory extension creation methods
func TestFactoryCreate(t *testing.T) {
	ctx := context.Background()
	factory := awss3eventextension.NewFactory()
	set := extensiontest.NewNopSettings(metadata.Type)

	t.Run("s3 event format", func(t *testing.T) {
		cfg := factory.CreateDefaultConfig().(*awss3eventextension.Config)
		cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
		cfg.Directory = "/tmp/s3event"

		ext, err := factory.Create(ctx, set, cfg)
		require.NoError(t, err)
		require.NotNil(t, ext)
	})

	t.Run("fdr event format", func(t *testing.T) {
		cfg := factory.CreateDefaultConfig().(*awss3eventextension.Config)
		cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
		cfg.Directory = "/tmp/s3event"
		cfg.EventFormat = "crowdstrike_fdr"

		ext, err := factory.Create(ctx, set, cfg)
		require.NoError(t, err)
		require.NotNil(t, ext)
	})

	t.Run("lack permissions to create directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping on windows")
		}
		invalidDir := filepath.Join("/dev", "null", "invalid-dir")

		cfg := factory.CreateDefaultConfig().(*awss3eventextension.Config)
		cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
		cfg.Directory = invalidDir

		_, err := factory.Create(ctx, set, cfg)
		assert.Error(t, err)
	})
}

// Test factory error cases
func TestFactoryCreateErrors(t *testing.T) {
	ctx := context.Background()
	factory := awss3eventextension.NewFactory()
	set := extensiontest.NewNopSettings(metadata.Type)

	t.Run("empty SQS queue URL", func(t *testing.T) {
		cfg := factory.CreateDefaultConfig().(*awss3eventextension.Config)
		cfg.SQSQueueURL = ""
		_, err := factory.Create(ctx, set, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'sqs_queue_url' is required")
	})

	t.Run("invalid SQS queue URL", func(t *testing.T) {
		cfg := factory.CreateDefaultConfig().(*awss3eventextension.Config)
		cfg.SQSQueueURL = "invalid"
		_, err := factory.Create(ctx, set, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract region from SQS URL")
	})

	t.Run("invalid event format", func(t *testing.T) {
		cfg := factory.CreateDefaultConfig().(*awss3eventextension.Config)
		cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
		cfg.EventFormat = "invalid"
		_, err := factory.Create(ctx, set, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported event format")
	})
}
