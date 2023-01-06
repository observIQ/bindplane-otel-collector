// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/receiver"
)

func TestGetRequiredFactories(t *testing.T) {
	testType := component.Type("test")
	extensionFactory := extension.NewFactory(testType, nil, createExtension, component.StabilityLevelAlpha)
	receiverFactory := receiver.NewFactory(testType, nil)
	processorFactory := component.NewProcessorFactory(testType, nil)
	emitterFactory := exporter.NewFactory(testType, nil)
	host := &MockHost{}
	host.On("GetFactory", component.KindReceiver, testType).Return(receiverFactory)
	host.On("GetFactory", component.KindProcessor, testType).Return(processorFactory)
	host.On("GetFactory", component.KindExtension, testType).Return(extensionFactory)
	host.On("GetFactory", mock.Anything, mock.Anything).Return(nil)

	testCases := []struct {
		name           string
		renderedCfg    *RenderedConfig
		expectedResult *component.Factories
		expectedErr    error
	}{
		{
			name: "missing receiver factory",
			renderedCfg: &RenderedConfig{
				Receivers: map[string]any{
					"missing": nil,
				},
			},
			expectedErr: errors.New("failed to get receiver factories"),
		},
		{
			name: "missing processor factory",
			renderedCfg: &RenderedConfig{
				Processors: map[string]any{
					"missing": nil,
				},
			},
			expectedErr: errors.New("failed to get processor factories"),
		},
		{
			name: "missing extension factory",
			renderedCfg: &RenderedConfig{
				Extensions: map[string]any{
					"missing": nil,
				},
			},
			expectedErr: errors.New("failed to get extension factories"),
		},
		{
			name: "all factories exist",
			renderedCfg: &RenderedConfig{
				Receivers: map[string]any{
					"test": nil,
				},
				Processors: map[string]any{
					"test": nil,
				},
				Extensions: map[string]any{
					"test": nil,
				},
			},
			expectedResult: &component.Factories{
				Receivers: map[component.Type]receiver.Factory{
					testType: receiverFactory,
				},
				Processors: map[component.Type]component.ProcessorFactory{
					testType: processorFactory,
				},
				Exporters: map[component.Type]exporter.Factory{
					emitterFactory.Type(): emitterFactory,
				},
				Extensions: map[component.Type]extension.Factory{
					testType: extensionFactory,
				},
			},
		},
		{
			name: "duplicate receivers defined",
			renderedCfg: &RenderedConfig{
				Receivers: map[string]any{
					"test":   nil,
					"test/2": nil,
				},
				Processors: map[string]any{
					"test":   nil,
					"test/2": nil,
				},
				Extensions: map[string]any{
					"test":   nil,
					"test/2": nil,
				},
			},
			expectedResult: &component.Factories{
				Receivers: map[component.Type]receiver.Factory{
					testType: receiverFactory,
				},
				Processors: map[component.Type]component.ProcessorFactory{
					testType: processorFactory,
				},
				Exporters: map[component.Type]exporter.Factory{
					emitterFactory.Type(): emitterFactory,
				},
				Extensions: map[component.Type]extension.Factory{
					testType: extensionFactory,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factories, err := tc.renderedCfg.GetRequiredFactories(host, emitterFactory)
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

// MockHost is a mock type for the component.Host type
type MockHost struct {
	mock.Mock
}

// GetExporters provides a mock function with given fields:
func (_m *MockHost) GetExporters() map[component.Type]map[component.ID]component.Component {
	ret := _m.Called()
	var r0 map[component.Type]map[component.ID]component.Component
	if rf, ok := ret.Get(0).(func() map[component.Type]map[component.ID]component.Component); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[component.Type]map[component.ID]component.Component)
		}
	}
	return r0
}

// GetExtensions provides a mock function with given fields:
func (_m *MockHost) GetExtensions() map[component.ID]component.Extension {
	ret := _m.Called()
	var r0 map[component.ID]component.Extension
	if rf, ok := ret.Get(0).(func() map[component.ID]component.Extension); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[component.ID]component.Extension)
		}
	}
	return r0
}

// GetFactory provides a mock function with given fields: kind, componentType
func (_m *MockHost) GetFactory(kind component.Kind, componentType component.Type) component.Factory {
	ret := _m.Called(kind, componentType)
	var r0 component.Factory
	if rf, ok := ret.Get(0).(func(component.Kind, component.Type) component.Factory); ok {
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

func createExtension(
	_ context.Context,
	_ component.ExtensionCreateSettings,
	_ component.Config,
) (component.Extension, error) {
	return nil, nil
}
