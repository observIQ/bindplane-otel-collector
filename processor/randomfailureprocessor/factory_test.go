// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package randomfailureprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.Equal(t, componentType, factory.Type())

	expectedCfg := &Config{
		FailureRate:  defaultFailureRate,
		ErrorMessage: defaultErrorMessage,
	}

	cfg, ok := factory.CreateDefaultConfig().(*Config)
	require.True(t, ok)
	require.Equal(t, expectedCfg, cfg)
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()
	settings := processortest.NewNopSettings(componentType)
	settings.ID = component.NewIDWithName(componentType, "proc1")

	traces, err := factory.CreateTraces(context.Background(), settings, factory.CreateDefaultConfig(), nil)
	require.NoError(t, err)
	require.NotNil(t, traces)

	logs, err := factory.CreateLogs(context.Background(), settings, factory.CreateDefaultConfig(), nil)
	require.NoError(t, err)
	require.NotNil(t, logs)

	metrics, err := factory.CreateMetrics(context.Background(), settings, factory.CreateDefaultConfig(), nil)
	require.NoError(t, err)
	require.NotNil(t, metrics)
}
