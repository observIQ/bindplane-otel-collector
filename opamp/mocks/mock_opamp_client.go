// Code generated by mockery v2.31.1. DO NOT EDIT.

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

// SetAgentDescription provides a mock function with given fields: descr
func (_m *MockOpAMPClient) SetAgentDescription(descr *protobufs.AgentDescription) error {
	ret := _m.Called(descr)

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.AgentDescription) error); ok {
		r0 = rf(descr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetHealth provides a mock function with given fields: health
func (_m *MockOpAMPClient) SetHealth(health *protobufs.AgentHealth) error {
	ret := _m.Called(health)

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.AgentHealth) error); ok {
		r0 = rf(health)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetPackageStatuses provides a mock function with given fields: statuses
func (_m *MockOpAMPClient) SetPackageStatuses(statuses *protobufs.PackageStatuses) error {
	ret := _m.Called(statuses)

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
