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

package lookupprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.Equal(t, componentType, factory.Type())
	require.Equal(t, stability, factory.LogsStability())
	require.Equal(t, stability, factory.MetricsStability())
	require.Equal(t, stability, factory.TracesStability())
	require.Equal(t, createDefaultConfig(), factory.CreateDefaultConfig())

	cfg := Config{
		Context: "body",
		Field:   "ip",
	}
	settings := processortest.NewNopSettings(componentType)
	consumer := consumertest.NewNop()

	traceProcessor, err := factory.CreateTraces(context.Background(), settings, &cfg, consumer)
	require.NoError(t, err)
	require.NotNil(t, traceProcessor)

	metricProcessor, err := factory.CreateMetrics(context.Background(), settings, &cfg, consumer)
	require.NoError(t, err)
	require.NotNil(t, metricProcessor)

	logProcessor, err := factory.CreateLogs(context.Background(), settings, &cfg, consumer)
	require.NoError(t, err)
	require.NotNil(t, logProcessor)
}
