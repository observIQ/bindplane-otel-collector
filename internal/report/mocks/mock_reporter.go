// Code generated by mockery v2.52.1. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// MockReporter is an autogenerated mock type for the Reporter type
type MockReporter struct {
	mock.Mock
}

// Kind provides a mock function with no fields
func (_m *MockReporter) Kind() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Kind")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Report provides a mock function with given fields: config
func (_m *MockReporter) Report(config interface{}) error {
	ret := _m.Called(config)

	if len(ret) == 0 {
		panic("no return value specified for Report")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(config)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockReporter creates a new instance of MockReporter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockReporter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockReporter {
	mock := &MockReporter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
