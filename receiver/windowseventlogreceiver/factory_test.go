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

package windowseventlogreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/windowseventlogreceiver"

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventlogreceiver/internal/metadata"
)

func TestNewFactory(t *testing.T) {
	t.Run("NewFactoryCorrectType", func(t *testing.T) {
		factory := NewFactory()
		require.Equal(t, metadata.Type, factory.Type())
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NotNil(t, cfg, "failed to create default config")
}

func TestCreateAndShutdown(t *testing.T) {
	factory := NewFactory()
	defaultConfig := factory.CreateDefaultConfig()
	cfg := defaultConfig.(*WindowsLogConfig) // This cast should work on all platforms.
	cfg.InputConfig.Channel = "Application"  // Must be explicitly set to a valid channel.

	ctx := t.Context()
	settings := receivertest.NewNopSettings(metadata.Type)
	sink := new(consumertest.LogsSink)
	receiver, err := factory.CreateLogs(ctx, settings, cfg, sink)

	if runtime.GOOS != "windows" {
		assert.Error(t, err)
		assert.IsType(t, pipeline.ErrSignalNotSupported, err)
		assert.Nil(t, receiver)
	} else {
		assert.NoError(t, err)
		require.NotNil(t, receiver)
		require.NoError(t, receiver.Shutdown(ctx))
	}
}
