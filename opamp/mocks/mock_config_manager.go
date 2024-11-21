// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	opamp "github.com/observiq/bindplane-agent/opamp"
	mock "github.com/stretchr/testify/mock"

	protobufs "github.com/open-telemetry/opamp-go/protobufs"
)

// MockConfigManager is an autogenerated mock type for the ConfigManager type
type MockConfigManager struct {
	mock.Mock
}

// AddConfig provides a mock function with given fields: configName, reloader
func (_m *MockConfigManager) AddConfig(configName string, reloader *opamp.ManagedConfig) {
	_m.Called(configName, reloader)
}

// ApplyConfigChanges provides a mock function with given fields: remoteConfig
func (_m *MockConfigManager) ApplyConfigChanges(remoteConfig *protobufs.AgentRemoteConfig) (bool, error) {
	ret := _m.Called(remoteConfig)

	if len(ret) == 0 {
		panic("no return value specified for ApplyConfigChanges")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(*protobufs.AgentRemoteConfig) (bool, error)); ok {
		return rf(remoteConfig)
	}
	if rf, ok := ret.Get(0).(func(*protobufs.AgentRemoteConfig) bool); ok {
		r0 = rf(remoteConfig)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(*protobufs.AgentRemoteConfig) error); ok {
		r1 = rf(remoteConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ComposeEffectiveConfig provides a mock function with given fields:
func (_m *MockConfigManager) ComposeEffectiveConfig() (*protobufs.EffectiveConfig, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ComposeEffectiveConfig")
	}

	var r0 *protobufs.EffectiveConfig
	var r1 error
	if rf, ok := ret.Get(0).(func() (*protobufs.EffectiveConfig, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *protobufs.EffectiveConfig); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*protobufs.EffectiveConfig)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockConfigManager creates a new instance of MockConfigManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockConfigManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockConfigManager {
	mock := &MockConfigManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
