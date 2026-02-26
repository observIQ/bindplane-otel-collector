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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"

	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/metadata"
)

// Test that the factory creates the default configuration correctly
func TestFactoryCreateDefaultConfig(t *testing.T) {
	factory := gcspubsubeventreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig()

	assert.Equal(t, metadata.Type, factory.Type())
	assert.Equal(t, "gcsevent", metadata.Type.String())
	assert.NotNil(t, cfg)

	assert.NoError(t, componenttest.CheckConfigStruct(cfg))

	receiverCfg, ok := cfg.(*gcspubsubeventreceiver.Config)
	require.True(t, ok)
	assert.Equal(t, "", receiverCfg.ProjectID)
	assert.Equal(t, "", receiverCfg.SubscriptionID)
	assert.Equal(t, "", receiverCfg.CredentialsFile)
	assert.Equal(t, 5, receiverCfg.Workers)
	assert.Equal(t, 1*time.Hour, receiverCfg.MaxExtension)
	assert.Equal(t, 1024*1024, receiverCfg.MaxLogSize)
	assert.Equal(t, 1000, receiverCfg.MaxLogsEmitted)
}

// Test factory receiver creation methods
func TestFactoryCreateReceivers(t *testing.T) {
	ctx := context.Background()
	factory := gcspubsubeventreceiver.NewFactory()

	// Create valid config
	cfg := factory.CreateDefaultConfig().(*gcspubsubeventreceiver.Config)
	cfg.ProjectID = "test-project"
	cfg.SubscriptionID = "test-subscription"

	// Create settings
	params := receiver.Settings{
		ID:                component.NewID(metadata.Type),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}

	// Test logs receiver
	logsConsumer := consumertest.NewNop()
	logsReceiver, err := factory.CreateLogs(ctx, params, cfg, logsConsumer)
	assert.NoError(t, err)
	assert.NotNil(t, logsReceiver)

	// Test stability levels
	assert.Equal(t, component.StabilityLevelAlpha, factory.LogsStability())
}

// Test factory error cases
func TestFactoryCreateReceiverErrors(t *testing.T) {
	ctx := context.Background()
	factory := gcspubsubeventreceiver.NewFactory()

	// Create invalid config (missing project_id)
	cfg := factory.CreateDefaultConfig().(*gcspubsubeventreceiver.Config)
	cfg.ProjectID = ""

	// Create settings
	params := receiver.Settings{
		ID:                component.NewID(metadata.Type),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}

	// Test logs receiver with invalid config
	consumerLogs := consumertest.NewNop()
	_, err := factory.CreateLogs(ctx, params, cfg, consumerLogs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'project_id' is required")
}
