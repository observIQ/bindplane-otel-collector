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

package throughputmeasurementprocessor

import (
	"context"
	"testing"

	"github.com/observiq/bindplane-agent/internal/measurements"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.Equal(t, componentType, factory.Type())

	expectedCfg := &Config{
		Enabled:       true,
		SamplingRatio: 0.5,
	}

	cfg, ok := factory.CreateDefaultConfig().(*Config)
	require.True(t, ok)
	require.Equal(t, expectedCfg, cfg)
}

// Test that 2 instances with the same processor ID will not error when started
func TestCreateProcessorTwice_Logs(t *testing.T) {
	processorID := component.MustNewIDWithName("throughputmeasurement", "1")
	bindplaneExtensionID := component.MustNewID("bindplane")

	set := processortest.NewNopSettings()
	set.ID = processorID

	cfg := &Config{
		Enabled:            true,
		SamplingRatio:      1,
		BindplaneExtension: bindplaneExtensionID,
	}

	l1, err := createLogsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	l2, err := createLogsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)

	mockBindplane := mockThoughputRegistry{
		ResettableThroughputMeasurementsRegistry: measurements.NewResettableThroughputMeasurementsRegistry(false),
	}

	mh := mockHost{
		extMap: map[component.ID]component.Component{
			bindplaneExtensionID: mockBindplane,
		},
	}

	require.NoError(t, l1.Start(context.Background(), mh))
	require.NoError(t, l2.Start(context.Background(), mh))
	require.NoError(t, l1.Shutdown(context.Background()))
	require.NoError(t, l2.Shutdown(context.Background()))
}

// Test that 2 instances with the same processor ID will not error when started
func TestCreateProcessorTwice_Metrics(t *testing.T) {
	processorID := component.MustNewIDWithName("throughputmeasurement", "1")
	bindplaneExtensionID := component.MustNewID("bindplane")

	set := processortest.NewNopSettings()
	set.ID = processorID

	cfg := &Config{
		Enabled:            true,
		SamplingRatio:      1,
		BindplaneExtension: bindplaneExtensionID,
	}

	l1, err := createMetricsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	l2, err := createMetricsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)

	mockBindplane := mockThoughputRegistry{
		ResettableThroughputMeasurementsRegistry: measurements.NewResettableThroughputMeasurementsRegistry(false),
	}

	mh := mockHost{
		extMap: map[component.ID]component.Component{
			bindplaneExtensionID: mockBindplane,
		},
	}

	require.NoError(t, l1.Start(context.Background(), mh))
	require.NoError(t, l2.Start(context.Background(), mh))
	require.NoError(t, l1.Shutdown(context.Background()))
	require.NoError(t, l2.Shutdown(context.Background()))
}

// Test that 2 instances with the same processor ID will not error when started
func TestCreateProcessorTwice_Traces(t *testing.T) {
	processorID := component.MustNewIDWithName("throughputmeasurement", "1")
	bindplaneExtensionID := component.MustNewID("bindplane")

	set := processortest.NewNopSettings()
	set.ID = processorID

	cfg := &Config{
		Enabled:            true,
		SamplingRatio:      1,
		BindplaneExtension: bindplaneExtensionID,
	}

	l1, err := createTracesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	l2, err := createTracesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)

	mockBindplane := mockThoughputRegistry{
		ResettableThroughputMeasurementsRegistry: measurements.NewResettableThroughputMeasurementsRegistry(false),
	}

	mh := mockHost{
		extMap: map[component.ID]component.Component{
			bindplaneExtensionID: mockBindplane,
		},
	}

	require.NoError(t, l1.Start(context.Background(), mh))
	require.NoError(t, l2.Start(context.Background(), mh))
	require.NoError(t, l1.Shutdown(context.Background()))
	require.NoError(t, l2.Shutdown(context.Background()))
}

type mockHost struct {
	extMap map[component.ID]component.Component
}

func (m mockHost) GetExtensions() map[component.ID]component.Component {
	return m.extMap
}

type mockThoughputRegistry struct {
	*measurements.ResettableThroughputMeasurementsRegistry
}

func (mockThoughputRegistry) Start(_ context.Context, _ component.Host) error { return nil }
func (mockThoughputRegistry) Shutdown(_ context.Context) error                { return nil }
