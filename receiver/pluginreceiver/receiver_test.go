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
	"go.opentelemetry.io/collector/service"
	"go.uber.org/zap"
)

func TestReceiverGetFactoryFailure(t *testing.T) {
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nil)

	components := ComponentMap{
		Receivers: map[string]interface{}{
			"missing": nil,
		},
	}
	configProvider := createConfigProvider(&components)
	emitterFactory := createLogEmitterFactory(nil)

	receiver := Receiver{
		plugin:         &Plugin{},
		configProvider: configProvider,
		emitterFactory: emitterFactory,
		logger:         zap.NewNop(),
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get factories")
}

func TestReceiverCreateServiceFailure(t *testing.T) {
	nopFactory := component.NewReceiverFactory("nop", nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	components := ComponentMap{
		Receivers: map[string]interface{}{
			"nop": nil,
		},
	}
	configProvider := createConfigProvider(&components)
	emitterFactory := createLogEmitterFactory(nil)

	receiver := Receiver{
		plugin:         &Plugin{},
		configProvider: configProvider,
		emitterFactory: emitterFactory,
		logger:         zap.NewNop(),
		createService: func(factories component.Factories, configProvider service.ConfigProvider, logger *zap.Logger) (Service, error) {
			return nil, errors.New("failure")
		},
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create internal service")
}

func TestReceiverStartServiceFailure(t *testing.T) {
	nopFactory := component.NewReceiverFactory("nop", nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	components := ComponentMap{
		Receivers: map[string]interface{}{
			"nop": nil,
		},
	}
	configProvider := createConfigProvider(&components)
	emitterFactory := createLogEmitterFactory(nil)

	svc := &MockService{}
	svc.On("Run", mock.Anything).Return(errors.New("failure"))
	svc.On("GetState").Return(service.Starting)

	receiver := Receiver{
		plugin:         &Plugin{},
		configProvider: configProvider,
		emitterFactory: emitterFactory,
		logger:         zap.NewNop(),
		createService: func(factories component.Factories, configProvider service.ConfigProvider, logger *zap.Logger) (Service, error) {
			return svc, nil
		},
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to start internal service")
}

func TestReceiverStartServiceContext(t *testing.T) {
	nopFactory := component.NewReceiverFactory("nop", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	components := ComponentMap{
		Receivers: map[string]interface{}{
			"nop": nil,
		},
	}
	configProvider := createConfigProvider(&components)
	emitterFactory := createLogEmitterFactory(nil)

	svc := &MockService{}
	svc.On("Run", mock.Anything).Return(nil)
	svc.On("GetState").Return(service.Starting)

	receiver := Receiver{
		plugin:         &Plugin{},
		configProvider: configProvider,
		emitterFactory: emitterFactory,
		logger:         zap.NewNop(),
		createService: func(factories component.Factories, configProvider service.ConfigProvider, logger *zap.Logger) (Service, error) {
			return svc, nil
		},
	}

	err := receiver.Start(ctx, host)
	require.Error(t, err)
	require.Contains(t, err.Error(), context.Canceled.Error())
}

func TestReceiverStartSuccess(t *testing.T) {
	nopFactory := component.NewReceiverFactory("nop", nil)
	ctx := context.Background()
	host := &MockHost{}
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nopFactory)

	components := ComponentMap{
		Receivers: map[string]interface{}{
			"nop": nil,
		},
	}
	configProvider := createConfigProvider(&components)
	emitterFactory := createLogEmitterFactory(nil)

	svc := &MockService{}
	svc.On("Run", mock.Anything).WaitUntil(time.After(time.Second)).Return(errors.New("unexpected timeout"))
	svc.On("GetState").Return(service.Running)

	receiver := Receiver{
		plugin:         &Plugin{},
		configProvider: configProvider,
		emitterFactory: emitterFactory,
		logger:         zap.NewNop(),
		createService: func(factories component.Factories, configProvider service.ConfigProvider, logger *zap.Logger) (Service, error) {
			return svc, nil
		},
	}

	err := receiver.Start(ctx, host)
	require.NoError(t, err)
}

func TestReceiverShutdown(t *testing.T) {
	ctx := context.Background()
	receiver := Receiver{}
	err := receiver.Shutdown(ctx)
	require.NoError(t, err)

	service := &MockService{}
	service.On("Shutdown").Return().Once()
	receiver.service = service
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)
	service.AssertExpectations(t)
}

// MockService is a mock type for the Service type
type MockService struct {
	mock.Mock
}

// GetState provides a mock function with given fields:
func (_m *MockService) GetState() service.State {
	ret := _m.Called()

	var r0 service.State
	if rf, ok := ret.Get(0).(func() service.State); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(service.State)
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
