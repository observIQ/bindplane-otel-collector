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

package pluginreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestCreateLogEmitterFactory(t *testing.T) {
	logs := pdata.Logs{}
	consumer := &MockConsumer{}
	consumer.On("ConsumeLogs", mock.Anything, logs).Return(nil).Once()

	factory := createLogEmitterFactory(consumer)
	ctx := context.Background()
	set := component.ExporterCreateSettings{}
	cfg := defaultEmitterConfig()

	exporter, err := factory.CreateLogsExporter(ctx, set, cfg)
	require.NoError(t, err)

	err = exporter.ConsumeLogs(ctx, logs)
	require.NoError(t, err)
	consumer.AssertExpectations(t)
}

func TestCreateMetricEmitterFactory(t *testing.T) {
	metrics := pdata.Metrics{}
	consumer := &MockConsumer{}
	consumer.On("ConsumeMetrics", mock.Anything, metrics).Return(nil).Once()

	factory := createMetricEmitterFactory(consumer)
	ctx := context.Background()
	set := component.ExporterCreateSettings{}
	cfg := defaultEmitterConfig()

	exporter, err := factory.CreateMetricsExporter(ctx, set, cfg)
	require.NoError(t, err)

	err = exporter.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)
	consumer.AssertExpectations(t)
}

func TestCreateTraceEmitterFactory(t *testing.T) {
	traces := pdata.Traces{}
	consumer := &MockConsumer{}
	consumer.On("ConsumeTraces", mock.Anything, traces).Return(nil).Once()

	factory := createTraceEmitterFactory(consumer)
	ctx := context.Background()
	set := component.ExporterCreateSettings{}
	cfg := defaultEmitterConfig()

	exporter, err := factory.CreateTracesExporter(ctx, set, cfg)
	require.NoError(t, err)

	err = exporter.ConsumeTraces(ctx, traces)
	require.NoError(t, err)
	consumer.AssertExpectations(t)
}

func TestEmitterStart(t *testing.T) {
	emitter := Emitter{}
	err := emitter.Start(context.Background(), nil)
	require.NoError(t, err)
}

func TestEmitterShutdown(t *testing.T) {
	emitter := Emitter{}
	err := emitter.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestEmitterCapabilities(t *testing.T) {
	emitter := Emitter{}
	capabilities := emitter.Capabilities()
	require.False(t, capabilities.MutatesData)
}

// MockConsumer is a mock for logs, metrics, and traces consumers
type MockConsumer struct {
	mock.Mock
}

// Capabilities provides a mock function with given fields:
func (_m *MockConsumer) Capabilities() consumer.Capabilities {
	ret := _m.Called()

	var r0 consumer.Capabilities
	if rf, ok := ret.Get(0).(func() consumer.Capabilities); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(consumer.Capabilities)
	}

	return r0
}

// ConsumeLogs provides a mock function with given fields: ctx, ld
func (_m *MockConsumer) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	ret := _m.Called(ctx, ld)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pdata.Logs) error); ok {
		r0 = rf(ctx, ld)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConsumeMetrics provides a mock function with given fields: ctx, md
func (_m *MockConsumer) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	ret := _m.Called(ctx, md)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pdata.Metrics) error); ok {
		r0 = rf(ctx, md)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConsumeTraces provides a mock function with given fields: ctx, td
func (_m *MockConsumer) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	ret := _m.Called(ctx, td)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pdata.Traces) error); ok {
		r0 = rf(ctx, td)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
