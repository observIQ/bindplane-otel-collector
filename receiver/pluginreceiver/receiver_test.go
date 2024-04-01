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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

func TestReceiverGetFactoryFailure(t *testing.T) {
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nil)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"missing": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	receiver := Receiver{
		plugin:         &Plugin{},
		renderedCfg:    renderedCfg,
		emitterFactory: emitterFactory,
		logger:         zap.NewNop(),
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get factories")
}

func TestReceiverCreateServiceFailure(t *testing.T) {
	nopType := component.MustNewType("nop")
	nopFactory := receiver.NewFactory(nopType, nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"nop": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	receiver := NewReceiver(&Plugin{}, renderedCfg, emitterFactory, zap.NewNop())
	receiver.createService = func(_ otelcol.Factories, _ otelcol.ConfigProvider, _ *zap.Logger) (Service, error) {
		return nil, errors.New("failure")
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create internal service")
}

func TestReceiverStartServiceFailure(t *testing.T) {
	nopType := component.MustNewType("nop")
	nopFactory := receiver.NewFactory(nopType, nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"nop": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	svc := &MockService{}
	svc.On("Run", mock.Anything).Return(errors.New("failure"))
	svc.On("GetState").Return(otelcol.StateStarting)
	receiver := NewReceiver(&Plugin{}, renderedCfg, emitterFactory, zap.NewNop())
	receiver.createService = func(_ otelcol.Factories, _ otelcol.ConfigProvider, _ *zap.Logger) (Service, error) {
		return svc, nil
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to start internal service")
}

func TestReceiverStartServiceContext(t *testing.T) {
	nopType := component.MustNewType("nop")
	nopFactory := receiver.NewFactory(nopType, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"nop": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	svc := &MockService{}
	svc.On("Run", mock.Anything).Return(nil)
	svc.On("GetState").Return(otelcol.StateStarting)
	receiver := NewReceiver(&Plugin{}, renderedCfg, emitterFactory, zap.NewNop())
	receiver.createService = func(_ otelcol.Factories, _ otelcol.ConfigProvider, _ *zap.Logger) (Service, error) {
		return svc, nil
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), context.Canceled.Error())
}

func TestReceiverStartSuccess(t *testing.T) {
	nopType := component.MustNewType("nop")
	nopFactory := receiver.NewFactory(nopType, nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"nop": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	svc := &MockService{}
	svc.On("Run", mock.Anything).WaitUntil(time.After(time.Second)).Return(errors.New("unexpected timeout"))
	svc.On("GetState").Return(otelcol.StateRunning)

	receiver := NewReceiver(&Plugin{}, renderedCfg, emitterFactory, zap.NewNop())
	receiver.createService = func(_ otelcol.Factories, _ otelcol.ConfigProvider, _ *zap.Logger) (Service, error) {
		return svc, nil
	}

	err := receiver.Start(ctx, host)
	require.NoError(t, err)
}

func TestReceiverShutdown(t *testing.T) {
	nopType := component.MustNewType("nop")
	nopFactory := receiver.NewFactory(nopType, nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"nop": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	blockChan := make(chan struct{})

	svc := &MockService{}
	svc.On("Run", mock.Anything).Run(func(_ mock.Arguments) {
		<-blockChan
	}).Return(nil)
	svc.On("GetState").Return(otelcol.StateRunning)
	svc.On("Shutdown").Run(func(_ mock.Arguments) {
		close(blockChan)
	}).Return()

	receiver := NewReceiver(&Plugin{}, renderedCfg, emitterFactory, zap.NewNop())
	receiver.createService = func(_ otelcol.Factories, _ otelcol.ConfigProvider, _ *zap.Logger) (Service, error) {
		return svc, nil
	}

	err := receiver.Start(ctx, host)
	require.NoError(t, err)

	err = receiver.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestReceiverShutdownCancelledContext(t *testing.T) {
	nopType := component.MustNewType("nop")
	nopFactory := receiver.NewFactory(nopType, nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"nop": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	blockChan := make(chan struct{})
	t.Cleanup(func() {
		close(blockChan)
	})

	svc := &MockService{}
	svc.On("Run", mock.Anything).Run(func(_ mock.Arguments) {
		<-blockChan
	}).Return(nil)
	svc.On("GetState").Return(otelcol.StateRunning)
	svc.On("Shutdown").Return()

	receiver := NewReceiver(&Plugin{}, renderedCfg, emitterFactory, zap.NewNop())
	receiver.createService = func(_ otelcol.Factories, _ otelcol.ConfigProvider, _ *zap.Logger) (Service, error) {
		return svc, nil
	}

	err := receiver.Start(ctx, host)
	require.NoError(t, err)

	// Create a context that is already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)
}

func TestReceiverShutdownWithError(t *testing.T) {
	nopType := component.MustNewType("nop")
	nopFactory := receiver.NewFactory(nopType, nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	renderedCfg := &RenderedConfig{
		Receivers: map[string]any{
			"nop": nil,
		},
	}

	emitterFactory := createLogEmitterFactory(nil)

	blockChan := make(chan struct{})

	svc := &MockService{}
	svc.On("Run", mock.Anything).Run(func(_ mock.Arguments) {
		<-blockChan
	}).Return(errors.New("an error occurred"))
	svc.On("GetState").Return(otelcol.StateRunning)
	svc.On("Shutdown").Run(func(_ mock.Arguments) {
		close(blockChan)
	}).Return()

	receiver := NewReceiver(&Plugin{}, renderedCfg, emitterFactory, zap.NewNop())
	receiver.createService = func(_ otelcol.Factories, _ otelcol.ConfigProvider, _ *zap.Logger) (Service, error) {
		return svc, nil
	}

	err := receiver.Start(ctx, host)
	require.NoError(t, err)

	err = receiver.Shutdown(context.Background())
	require.ErrorContains(t, err, "an error occurred")
}

// MockService is a mock type for the Service type
type MockService struct {
	mock.Mock
}

// GetState provides a mock function with given fields:
func (_m *MockService) GetState() otelcol.State {
	ret := _m.Called()

	var r0 otelcol.State
	if rf, ok := ret.Get(0).(func() otelcol.State); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(otelcol.State)
	}

	return r0
}

// Run provides a mock function with given fields: ctx
func (_m *MockService) Run(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Shutdown provides a mock function with given fields:
func (_m *MockService) Shutdown() {
	_m.Called()
}
