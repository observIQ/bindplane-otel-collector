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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/extension/extensionhelper"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

func TestConfigProviderWatch(t *testing.T) {
	provider := createConfigProvider(nil)
	err := errors.New("config err")
	go func() {
		provider.errChan <- err
	}()

	receivedErr := <-provider.Watch()
	require.Equal(t, err, receivedErr)
}

func TestConfigProviderShutdown(t *testing.T) {
	provider := createConfigProvider(nil)
	err := provider.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestConfigProviderGet(t *testing.T) {
	ctx := context.Background()
	components := &ComponentMap{}
	provider := createConfigProvider(components)
	factories := component.Factories{}

	unmarshaller := &MockUnmarshaller{}
	unmarshaller.On("Unmarshal", mock.Anything, mock.Anything).Return(nil, errors.New("failure")).Once()
	unmarshaller.On("Unmarshal", mock.Anything, mock.Anything).Return(&config.Config{}, nil).Once()
	provider.unmarshaller = unmarshaller

	cfg, err := provider.Get(ctx, factories)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal config")
	require.Nil(t, cfg)

	cfg, err = provider.Get(ctx, factories)
	require.NoError(t, err)
	require.Equal(t, &config.Config{}, cfg)
}

func TestGetRequiredFactories(t *testing.T) {
	testType := config.Type("test")
	emitterFactory := exporterhelper.NewFactory(testType, nil)
	receiverFactory := receiverhelper.NewFactory(testType, nil)
	processorFactory := processorhelper.NewFactory(testType, nil)
	extensionFactory := extensionhelper.NewFactory(testType, nil, nil)

	host := &MockHost{}
	host.On("GetFactory", component.KindReceiver, testType).Return(receiverFactory)
	host.On("GetFactory", component.KindProcessor, testType).Return(processorFactory)
	host.On("GetFactory", component.KindExtension, testType).Return(extensionFactory)
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nil)

	testCases := []struct {
		name           string
		components     *ComponentMap
		expectedResult *component.Factories
		expectedErr    error
	}{
		{
			name: "missing receiver factory",
			components: &ComponentMap{
				Receivers: map[string]interface{}{
					"missing": nil,
				},
			},
			expectedErr: errors.New("failed to get receiver factories"),
		},
		{
			name: "missing processor factory",
			components: &ComponentMap{
				Processors: map[string]interface{}{
					"missing": nil,
				},
			},
			expectedErr: errors.New("failed to get processor factories"),
		},
		{
			name: "missing extension factory",
			components: &ComponentMap{
				Extensions: map[string]interface{}{
					"missing": nil,
				},
			},
			expectedErr: errors.New("failed to get extension factories"),
		},
		{
			name: "all factories exist",
			components: &ComponentMap{
				Receivers: map[string]interface{}{
					"test": nil,
				},
				Processors: map[string]interface{}{
					"test": nil,
				},
				Extensions: map[string]interface{}{
					"test": nil,
				},
			},
			expectedResult: &component.Factories{
				Receivers: map[config.Type]component.ReceiverFactory{
					testType: receiverFactory,
				},
				Processors: map[config.Type]component.ProcessorFactory{
					testType: processorFactory,
				},
				Exporters: map[config.Type]component.ExporterFactory{
					emitterFactory.Type(): emitterFactory,
				},
				Extensions: map[config.Type]component.ExtensionFactory{
					testType: extensionFactory,
				},
			},
		},
		{
			name: "duplicate receivers defined",
			components: &ComponentMap{
				Receivers: map[string]interface{}{
					"test":   nil,
					"test/2": nil,
				},
				Processors: map[string]interface{}{
					"test":   nil,
					"test/2": nil,
				},
				Extensions: map[string]interface{}{
					"test":   nil,
					"test/2": nil,
				},
			},
			expectedResult: &component.Factories{
				Receivers: map[config.Type]component.ReceiverFactory{
					testType: receiverFactory,
				},
				Processors: map[config.Type]component.ProcessorFactory{
					testType: processorFactory,
				},
				Exporters: map[config.Type]component.ExporterFactory{
					emitterFactory.Type(): emitterFactory,
				},
				Extensions: map[config.Type]component.ExtensionFactory{
					testType: extensionFactory,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := createConfigProvider(tc.components)
			factories, err := provider.GetRequiredFactories(host, emitterFactory)
			switch tc.expectedErr {
			case nil:
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, factories)
			default:
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestComponentsToConfigMap(t *testing.T) {
	components := ComponentMap{
		Receivers: map[string]interface{}{
			"receiver": nil,
		},
		Processors: map[string]interface{}{
			"processor": nil,
		},
		Exporters: map[string]interface{}{
			"exporter": nil,
		},
		Extensions: map[string]interface{}{
			"extension": nil,
		},
		Service: ServiceMap{
			Extensions: []string{"extension"},
			Pipelines: map[string]PipelineMap{
				"metrics": {
					Receivers:  []string{"receiver"},
					Processors: []string{"processor"},
					Exporters:  []string{"exporter"},
				},
			},
		},
	}

	stringMap := map[string]interface{}{
		"receivers": map[string]interface{}{
			"receiver": nil,
		},
		"processors": map[string]interface{}{
			"processor": nil,
		},
		"exporters": map[string]interface{}{
			"exporter": nil,
		},
		"extensions": map[string]interface{}{
			"extension": nil,
		},
		"service": map[string]interface{}{
			"extensions": []string{"extension"},
			"pipelines": map[string]interface{}{
				"metrics": map[string]interface{}{
					"receivers":  []string{"receiver"},
					"processors": []string{"processor"},
					"exporters":  []string{"exporter"},
				},
			},
		},
	}

	require.Equal(t, stringMap, components.ToConfigMap().ToStringMap())
}

// MockUnmarshaller is a mock type for the configunmarshaler.Unmarshaller type
type MockUnmarshaller struct {
	mock.Mock
}

// Unmarshal provides a mock function with given fields: v, factories
func (_m *MockUnmarshaller) Unmarshal(v *config.Map, factories component.Factories) (*config.Config, error) {
	ret := _m.Called(v, factories)

	var r0 *config.Config
	if rf, ok := ret.Get(0).(func(*config.Map, component.Factories) *config.Config); ok {
		r0 = rf(v, factories)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*config.Config)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*config.Map, component.Factories) error); ok {
		r1 = rf(v, factories)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockHost is a mock type for the component.Host type
type MockHost struct {
	mock.Mock
}

// GetExporters provides a mock function with given fields:
func (_m *MockHost) GetExporters() map[config.Type]map[config.ComponentID]component.Exporter {
	ret := _m.Called()

	var r0 map[config.Type]map[config.ComponentID]component.Exporter
	if rf, ok := ret.Get(0).(func() map[config.Type]map[config.ComponentID]component.Exporter); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[config.Type]map[config.ComponentID]component.Exporter)
		}
	}

	return r0
}

// GetExtensions provides a mock function with given fields:
func (_m *MockHost) GetExtensions() map[config.ComponentID]component.Extension {
	ret := _m.Called()

	var r0 map[config.ComponentID]component.Extension
	if rf, ok := ret.Get(0).(func() map[config.ComponentID]component.Extension); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[config.ComponentID]component.Extension)
		}
	}

	return r0
}

// GetFactory provides a mock function with given fields: kind, componentType
func (_m *MockHost) GetFactory(kind component.Kind, componentType config.Type) component.Factory {
	ret := _m.Called(kind, componentType)

	var r0 component.Factory
	if rf, ok := ret.Get(0).(func(component.Kind, config.Type) component.Factory); ok {
		r0 = rf(kind, componentType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(component.Factory)
		}
	}

	return r0
}

// ReportFatalError provides a mock function with given fields: err
func (_m *MockHost) ReportFatalError(err error) {
	_m.Called(err)
}
