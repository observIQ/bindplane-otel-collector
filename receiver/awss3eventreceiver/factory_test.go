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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
)

// Test that the factory creates the default configuration correctly
func TestFactoryCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	assert.Equal(t, Type, factory.Type())
	assert.Equal(t, "awss3event", Type.String())
	assert.NotNil(t, cfg)

	assert.NoError(t, componenttest.CheckConfigStruct(cfg))

	receiverCfg, ok := cfg.(*Config)
	require.True(t, ok)
	assert.Equal(t, "", receiverCfg.SQSQueueURL)
	assert.Equal(t, 20*time.Second, receiverCfg.PollInterval)
	assert.Equal(t, 300*time.Second, receiverCfg.VisibilityTimeout)
	assert.Equal(t, int32(10), receiverCfg.APIMaxMessages)
}

// Test factory receiver creation methods
func TestFactoryCreateReceivers(t *testing.T) {
	ctx := context.Background()
	factory := NewFactory()

	// Create valid config
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"

	// Create settings
	params := receiver.Settings{
		ID:                component.NewID(Type),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}

	// Test logs receiver
	logsConsumer := consumertest.NewNop()
	logsReceiver, err := factory.CreateLogs(ctx, params, cfg, logsConsumer)
	assert.NoError(t, err)
	assert.NotNil(t, logsReceiver)

	// Test stability levels
	assert.Equal(t, component.StabilityLevelDevelopment, factory.LogsStability())
}

// Test factory error cases
func TestFactoryCreateReceiverErrors(t *testing.T) {
	ctx := context.Background()
	factory := NewFactory()

	// Create invalid config
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SQSQueueURL = ""

	// Create settings
	params := receiver.Settings{
		ID:                component.NewID(Type),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}

	// Test logs receiver with invalid config
	consumerLogs := consumertest.NewNop()
	_, err := factory.CreateLogs(ctx, params, cfg, consumerLogs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'sqs_queue_url' is required")
}
