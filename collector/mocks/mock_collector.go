// Code generated by mockery v2.52.1. DO NOT EDIT.

package mocks

import (
	context "context"

	collector "github.com/observiq/bindplane-otel-collector/collector"

	mock "github.com/stretchr/testify/mock"

	zap "go.uber.org/zap"
)

// MockCollector is an autogenerated mock type for the Collector type
type MockCollector struct {
	mock.Mock
}

// GetLoggingOpts provides a mock function with no fields
func (_m *MockCollector) GetLoggingOpts() []zap.Option {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLoggingOpts")
	}

	var r0 []zap.Option
	if rf, ok := ret.Get(0).(func() []zap.Option); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]zap.Option)
		}
	}

	return r0
}

// Restart provides a mock function with given fields: _a0
func (_m *MockCollector) Restart(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Restart")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Run provides a mock function with given fields: _a0
func (_m *MockCollector) Run(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Run")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetLoggingOpts provides a mock function with given fields: _a0
func (_m *MockCollector) SetLoggingOpts(_a0 []zap.Option) {
	_m.Called(_a0)
}

// Status provides a mock function with no fields
func (_m *MockCollector) Status() <-chan *collector.Status {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Status")
	}

	var r0 <-chan *collector.Status
	if rf, ok := ret.Get(0).(func() <-chan *collector.Status); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *collector.Status)
		}
	}

	return r0
}

// Stop provides a mock function with given fields: _a0
func (_m *MockCollector) Stop(_a0 context.Context) {
	_m.Called(_a0)
}

// NewMockCollector creates a new instance of MockCollector. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCollector(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCollector {
	mock := &MockCollector{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
