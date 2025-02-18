// Code generated by mockery v2.52.2. DO NOT EDIT.

package chronicleexporter

import (
	context "context"

	api "github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"

	mock "github.com/stretchr/testify/mock"

	plog "go.opentelemetry.io/collector/pdata/plog"
)

// MockMarshaler is an autogenerated mock type for the logMarshaler type
type MockMarshaler struct {
	mock.Mock
}

// MarshalRawLogs provides a mock function with given fields: ctx, ld
func (_m *MockMarshaler) MarshalRawLogs(ctx context.Context, ld plog.Logs) ([]*api.BatchCreateLogsRequest, error) {
	ret := _m.Called(ctx, ld)

	if len(ret) == 0 {
		panic("no return value specified for MarshalRawLogs")
	}

	var r0 []*api.BatchCreateLogsRequest
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, plog.Logs) ([]*api.BatchCreateLogsRequest, error)); ok {
		return rf(ctx, ld)
	}
	if rf, ok := ret.Get(0).(func(context.Context, plog.Logs) []*api.BatchCreateLogsRequest); ok {
		r0 = rf(ctx, ld)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*api.BatchCreateLogsRequest)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, plog.Logs) error); ok {
		r1 = rf(ctx, ld)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MarshalRawLogsForHTTP provides a mock function with given fields: ctx, ld
func (_m *MockMarshaler) MarshalRawLogsForHTTP(ctx context.Context, ld plog.Logs) (map[string][]*api.ImportLogsRequest, error) {
	ret := _m.Called(ctx, ld)

	if len(ret) == 0 {
		panic("no return value specified for MarshalRawLogsForHTTP")
	}

	var r0 map[string][]*api.ImportLogsRequest
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, plog.Logs) (map[string][]*api.ImportLogsRequest, error)); ok {
		return rf(ctx, ld)
	}
	if rf, ok := ret.Get(0).(func(context.Context, plog.Logs) map[string][]*api.ImportLogsRequest); ok {
		r0 = rf(ctx, ld)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]*api.ImportLogsRequest)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, plog.Logs) error); ok {
		r1 = rf(ctx, ld)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
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
