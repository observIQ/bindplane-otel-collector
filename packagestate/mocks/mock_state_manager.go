// Code generated by mockery v2.12.2. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	protobufs "github.com/open-telemetry/opamp-go/protobufs"

	testing "testing"
)

// MockStateManager is an autogenerated mock type for the StateManager type
type MockStateManager struct {
	mock.Mock
}

// LoadStatuses provides a mock function with given fields:
func (_m *MockStateManager) LoadStatuses() (*protobufs.PackageStatuses, error) {
	ret := _m.Called()

	var r0 *protobufs.PackageStatuses
	if rf, ok := ret.Get(0).(func() *protobufs.PackageStatuses); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*protobufs.PackageStatuses)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveStatuses provides a mock function with given fields: statuses
func (_m *MockStateManager) SaveStatuses(statuses *protobufs.PackageStatuses) error {
	ret := _m.Called(statuses)

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.PackageStatuses) error); ok {
		r0 = rf(statuses)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockStateManager creates a new instance of MockStateManager. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockStateManager(t testing.TB) *MockStateManager {
	mock := &MockStateManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
