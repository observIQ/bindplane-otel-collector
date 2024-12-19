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

package topologyprocessor

import (
	"context"
	"testing"

	"github.com/observiq/bindplane-otel-collector/internal/topology"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.Equal(t, componentType, factory.Type())

	expectedCfg := &Config{
		Enabled:  false,
		Interval: defaultInterval,
	}

	cfg, ok := factory.CreateDefaultConfig().(*Config)
	require.True(t, ok)
	require.Equal(t, expectedCfg, cfg)
}

func TestCreateOrGetProcessor(t *testing.T) {
	p1Settings := processortest.NewNopSettings()
	p1Settings.ID = component.MustNewIDWithName(componentType.String(), "proc1")

	p1, err := createOrGetProcessor(p1Settings, createDefaultConfig().(*Config))
	require.NoError(t, err)
	p1Copy, err := createOrGetProcessor(p1Settings, createDefaultConfig().(*Config))
	require.NoError(t, err)

	// p1 and p1Copy should be the same pointer
	require.True(t, p1 == p1Copy, "p1 and p1Copy are not the same pointer")

	p2Settings := processortest.NewNopSettings()
	p2Settings.ID = component.MustNewIDWithName(componentType.String(), "proc2")

	p2, err := createOrGetProcessor(p2Settings, createDefaultConfig().(*Config))
	require.NoError(t, err)
	require.True(t, p2 != p1, "p2 and p1 are the same, but they should be different objects")
}

// Test that 2 instances with the same processor ID will not error when started
func TestCreateProcessorTwice_Logs(t *testing.T) {
	processorID := component.MustNewIDWithName("topology", "1")
	bindplaneExtensionID := component.MustNewID("bindplane")

	set := processortest.NewNopSettings()
	set.ID = processorID

	cfg := &Config{
		Enabled:            true,
		Interval:           defaultInterval,
		Configuration:      "myConf",
		AccountID:          "myAcct",
		OrganizationID:     "myOrg",
		BindplaneExtension: bindplaneExtensionID,
	}

	l1, err := createLogsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	l2, err := createLogsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)

	mockBindplane := mockTopologyRegistry{
		ResettableTopologyRegistry: topology.NewResettableTopologyRegistry(),
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
		Interval:           defaultInterval,
		Configuration:      "myConf",
		AccountID:          "myAcct",
		OrganizationID:     "myOrg",
		BindplaneExtension: bindplaneExtensionID,
	}

	l1, err := createMetricsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	l2, err := createMetricsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)

	mockBindplane := mockTopologyRegistry{
		ResettableTopologyRegistry: topology.NewResettableTopologyRegistry(),
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
		Interval:           defaultInterval,
		Configuration:      "myConf",
		AccountID:          "myAcct",
		OrganizationID:     "myOrg",
		BindplaneExtension: bindplaneExtensionID,
	}

	l1, err := createTracesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	l2, err := createTracesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)

	mockBindplane := mockTopologyRegistry{
		ResettableTopologyRegistry: topology.NewResettableTopologyRegistry(),
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

func (m *mockHost) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

func (m mockHost) GetExtensions() map[component.ID]component.Component {
	return m.extMap
}

type mockTopologyRegistry struct {
	*topology.ResettableTopologyRegistry
}

func (mockTopologyRegistry) Start(_ context.Context, _ component.Host) error { return nil }
func (mockTopologyRegistry) Shutdown(_ context.Context) error                { return nil }
