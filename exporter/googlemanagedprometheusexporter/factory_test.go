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

package googlemanagedprometheusexporter

import (
	"context"
	"testing"

	gmp "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestCreateMetricExporterSuccess(t *testing.T) {
	mockExporter := &MockExporter{}

	gmpFactory = exporter.NewFactory(
		componentType,
		gmpFactory.CreateDefaultConfig,
		exporter.WithMetrics(func(_ context.Context, _ exporter.Settings, _ component.Config) (exporter.Metrics, error) {
			return mockExporter, nil
		}, stability),
	)
	defer func() {
		gmpFactory = gmp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()()
	ctx := context.Background()
	set := exportertest.NewNopSettings()

	testExporter, err := factory.CreateMetricsExporter(ctx, set, cfg)
	require.NoError(t, err)

	createdMockExporter, ok := testExporter.(*MockExporter)
	require.True(t, ok)
	require.Equal(t, createdMockExporter, mockExporter)
}

func TestCreateExporterFailure(t *testing.T) {
	gmpFactory = exporter.NewFactory(
		componentType,
		gmpFactory.CreateDefaultConfig,
	)
	defer func() {
		gmpFactory = gmp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()()
	ctx := context.Background()
	set := exportertest.NewNopSettings()

	_, err := factory.CreateMetricsExporter(ctx, set, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create metrics exporter")
}

// MockExporter is an autogenerated mock type for the Exporter type
type MockExporter struct {
	mock.Mock
}

// Capabilities provides a mock function with given fields:
func (_m *MockExporter) Capabilities() consumer.Capabilities {
	ret := _m.Called()

	var r0 consumer.Capabilities
	if rf, ok := ret.Get(0).(func() consumer.Capabilities); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(consumer.Capabilities)
	}

	return r0
}

// ConsumeMetrics provides a mock function with given fields: ctx, md
func (_m *MockExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	ret := _m.Called(ctx, md)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pmetric.Metrics) error); ok {
		r0 = rf(ctx, md)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Shutdown provides a mock function with given fields: ctx
func (_m *MockExporter) Shutdown(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields: ctx, host
func (_m *MockExporter) Start(ctx context.Context, host component.Host) error {
	ret := _m.Called(ctx, host)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, component.Host) error); ok {
		r0 = rf(ctx, host)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
