// Code generated by mockery v2.37.1. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	plog "go.opentelemetry.io/collector/pdata/plog"

	pmetric "go.opentelemetry.io/collector/pdata/pmetric"

	ptrace "go.opentelemetry.io/collector/pdata/ptrace"
)

// MockMarshaler is an autogenerated mock type for the marshaler type
type MockMarshaler struct {
	mock.Mock
}

type MockMarshaler_Expecter struct {
	mock *mock.Mock
}

func (_m *MockMarshaler) EXPECT() *MockMarshaler_Expecter {
	return &MockMarshaler_Expecter{mock: &_m.Mock}
}

// Format provides a mock function with given fields:
func (_m *MockMarshaler) Format() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// MockMarshaler_Format_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Format'
type MockMarshaler_Format_Call struct {
	*mock.Call
}

// Format is a helper method to define mock.On call
func (_e *MockMarshaler_Expecter) Format() *MockMarshaler_Format_Call {
	return &MockMarshaler_Format_Call{Call: _e.mock.On("Format")}
}

func (_c *MockMarshaler_Format_Call) Run(run func()) *MockMarshaler_Format_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockMarshaler_Format_Call) Return(_a0 string) *MockMarshaler_Format_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockMarshaler_Format_Call) RunAndReturn(run func() string) *MockMarshaler_Format_Call {
	_c.Call.Return(run)
	return _c
}

// MarshalLogs provides a mock function with given fields: ld
func (_m *MockMarshaler) MarshalLogs(ld plog.Logs) ([]byte, error) {
	ret := _m.Called(ld)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(plog.Logs) ([]byte, error)); ok {
		return rf(ld)
	}
	if rf, ok := ret.Get(0).(func(plog.Logs) []byte); ok {
		r0 = rf(ld)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(plog.Logs) error); ok {
		r1 = rf(ld)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMarshaler_MarshalLogs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MarshalLogs'
type MockMarshaler_MarshalLogs_Call struct {
	*mock.Call
}

// MarshalLogs is a helper method to define mock.On call
//   - ld plog.Logs
func (_e *MockMarshaler_Expecter) MarshalLogs(ld interface{}) *MockMarshaler_MarshalLogs_Call {
	return &MockMarshaler_MarshalLogs_Call{Call: _e.mock.On("MarshalLogs", ld)}
}

func (_c *MockMarshaler_MarshalLogs_Call) Run(run func(ld plog.Logs)) *MockMarshaler_MarshalLogs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(plog.Logs))
	})
	return _c
}

func (_c *MockMarshaler_MarshalLogs_Call) Return(_a0 []byte, _a1 error) *MockMarshaler_MarshalLogs_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMarshaler_MarshalLogs_Call) RunAndReturn(run func(plog.Logs) ([]byte, error)) *MockMarshaler_MarshalLogs_Call {
	_c.Call.Return(run)
	return _c
}

// MarshalMetrics provides a mock function with given fields: md
func (_m *MockMarshaler) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	ret := _m.Called(md)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(pmetric.Metrics) ([]byte, error)); ok {
		return rf(md)
	}
	if rf, ok := ret.Get(0).(func(pmetric.Metrics) []byte); ok {
		r0 = rf(md)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(pmetric.Metrics) error); ok {
		r1 = rf(md)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMarshaler_MarshalMetrics_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MarshalMetrics'
type MockMarshaler_MarshalMetrics_Call struct {
	*mock.Call
}

// MarshalMetrics is a helper method to define mock.On call
//   - md pmetric.Metrics
func (_e *MockMarshaler_Expecter) MarshalMetrics(md interface{}) *MockMarshaler_MarshalMetrics_Call {
	return &MockMarshaler_MarshalMetrics_Call{Call: _e.mock.On("MarshalMetrics", md)}
}

func (_c *MockMarshaler_MarshalMetrics_Call) Run(run func(md pmetric.Metrics)) *MockMarshaler_MarshalMetrics_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(pmetric.Metrics))
	})
	return _c
}

func (_c *MockMarshaler_MarshalMetrics_Call) Return(_a0 []byte, _a1 error) *MockMarshaler_MarshalMetrics_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMarshaler_MarshalMetrics_Call) RunAndReturn(run func(pmetric.Metrics) ([]byte, error)) *MockMarshaler_MarshalMetrics_Call {
	_c.Call.Return(run)
	return _c
}

// MarshalTraces provides a mock function with given fields: td
func (_m *MockMarshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	ret := _m.Called(td)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(ptrace.Traces) ([]byte, error)); ok {
		return rf(td)
	}
	if rf, ok := ret.Get(0).(func(ptrace.Traces) []byte); ok {
		r0 = rf(td)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(ptrace.Traces) error); ok {
		r1 = rf(td)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMarshaler_MarshalTraces_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MarshalTraces'
type MockMarshaler_MarshalTraces_Call struct {
	*mock.Call
}

// MarshalTraces is a helper method to define mock.On call
//   - td ptrace.Traces
func (_e *MockMarshaler_Expecter) MarshalTraces(td interface{}) *MockMarshaler_MarshalTraces_Call {
	return &MockMarshaler_MarshalTraces_Call{Call: _e.mock.On("MarshalTraces", td)}
}

func (_c *MockMarshaler_MarshalTraces_Call) Run(run func(td ptrace.Traces)) *MockMarshaler_MarshalTraces_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(ptrace.Traces))
	})
	return _c
}

func (_c *MockMarshaler_MarshalTraces_Call) Return(_a0 []byte, _a1 error) *MockMarshaler_MarshalTraces_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMarshaler_MarshalTraces_Call) RunAndReturn(run func(ptrace.Traces) ([]byte, error)) *MockMarshaler_MarshalTraces_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockMarshaler creates a new instance of MockMarshaler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockMarshaler(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockMarshaler {
	mock := &MockMarshaler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
