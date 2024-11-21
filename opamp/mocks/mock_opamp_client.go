// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	context "context"

	protobufs "github.com/open-telemetry/opamp-go/protobufs"
	mock "github.com/stretchr/testify/mock"

	types "github.com/open-telemetry/opamp-go/client/types"
)

// MockOpAMPClient is an autogenerated mock type for the OpAMPClient type
type MockOpAMPClient struct {
	mock.Mock
}

// AgentDescription provides a mock function with given fields:
func (_m *MockOpAMPClient) AgentDescription() *protobufs.AgentDescription {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for AgentDescription")
	}

	var r0 *protobufs.AgentDescription
	if rf, ok := ret.Get(0).(func() *protobufs.AgentDescription); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*protobufs.AgentDescription)
		}
	}

	return r0
}

// RequestConnectionSettings provides a mock function with given fields: request
func (_m *MockOpAMPClient) RequestConnectionSettings(request *protobufs.ConnectionSettingsRequest) error {
	ret := _m.Called(request)

	if len(ret) == 0 {
		panic("no return value specified for RequestConnectionSettings")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.ConnectionSettingsRequest) error); ok {
		r0 = rf(request)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendCustomMessage provides a mock function with given fields: message
func (_m *MockOpAMPClient) SendCustomMessage(message *protobufs.CustomMessage) (chan struct{}, error) {
	ret := _m.Called(message)

	if len(ret) == 0 {
		panic("no return value specified for SendCustomMessage")
	}

	var r0 chan struct{}
	var r1 error
	if rf, ok := ret.Get(0).(func(*protobufs.CustomMessage) (chan struct{}, error)); ok {
		return rf(message)
	}
	if rf, ok := ret.Get(0).(func(*protobufs.CustomMessage) chan struct{}); ok {
		r0 = rf(message)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan struct{})
		}
	}

	if rf, ok := ret.Get(1).(func(*protobufs.CustomMessage) error); ok {
		r1 = rf(message)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetAgentDescription provides a mock function with given fields: descr
func (_m *MockOpAMPClient) SetAgentDescription(descr *protobufs.AgentDescription) error {
	ret := _m.Called(descr)

	if len(ret) == 0 {
		panic("no return value specified for SetAgentDescription")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.AgentDescription) error); ok {
		r0 = rf(descr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetCustomCapabilities provides a mock function with given fields: customCapabilities
func (_m *MockOpAMPClient) SetCustomCapabilities(customCapabilities *protobufs.CustomCapabilities) error {
	ret := _m.Called(customCapabilities)

	if len(ret) == 0 {
		panic("no return value specified for SetCustomCapabilities")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.CustomCapabilities) error); ok {
		r0 = rf(customCapabilities)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetFlags provides a mock function with given fields: flags
func (_m *MockOpAMPClient) SetFlags(flags protobufs.AgentToServerFlags) {
	_m.Called(flags)
}

// SetHealth provides a mock function with given fields: health
func (_m *MockOpAMPClient) SetHealth(health *protobufs.ComponentHealth) error {
	ret := _m.Called(health)

	if len(ret) == 0 {
		panic("no return value specified for SetHealth")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.ComponentHealth) error); ok {
		r0 = rf(health)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetPackageStatuses provides a mock function with given fields: statuses
func (_m *MockOpAMPClient) SetPackageStatuses(statuses *protobufs.PackageStatuses) error {
	ret := _m.Called(statuses)

	if len(ret) == 0 {
		panic("no return value specified for SetPackageStatuses")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.PackageStatuses) error); ok {
		r0 = rf(statuses)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetRemoteConfigStatus provides a mock function with given fields: status
func (_m *MockOpAMPClient) SetRemoteConfigStatus(status *protobufs.RemoteConfigStatus) error {
	ret := _m.Called(status)

	if len(ret) == 0 {
		panic("no return value specified for SetRemoteConfigStatus")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.RemoteConfigStatus) error); ok {
		r0 = rf(status)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields: ctx, settings
func (_m *MockOpAMPClient) Start(ctx context.Context, settings types.StartSettings) error {
	ret := _m.Called(ctx, settings)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.StartSettings) error); ok {
		r0 = rf(ctx, settings)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Stop provides a mock function with given fields: ctx
func (_m *MockOpAMPClient) Stop(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Stop")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateEffectiveConfig provides a mock function with given fields: ctx
func (_m *MockOpAMPClient) UpdateEffectiveConfig(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateEffectiveConfig")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockOpAMPClient creates a new instance of MockOpAMPClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockOpAMPClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockOpAMPClient {
	mock := &MockOpAMPClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
