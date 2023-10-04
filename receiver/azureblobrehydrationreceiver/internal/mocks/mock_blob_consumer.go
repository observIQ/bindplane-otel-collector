// Code generated by mockery v2.31.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockBlobConsumer is an autogenerated mock type for the blobConsumer type
type MockBlobConsumer struct {
	mock.Mock
}

type MockBlobConsumer_Expecter struct {
	mock *mock.Mock
}

func (_m *MockBlobConsumer) EXPECT() *MockBlobConsumer_Expecter {
	return &MockBlobConsumer_Expecter{mock: &_m.Mock}
}

// Consume provides a mock function with given fields: ctx, blobContent
func (_m *MockBlobConsumer) Consume(ctx context.Context, blobContent []byte) error {
	ret := _m.Called(ctx, blobContent)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) error); ok {
		r0 = rf(ctx, blobContent)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockBlobConsumer_Consume_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Consume'
type MockBlobConsumer_Consume_Call struct {
	*mock.Call
}

// Consume is a helper method to define mock.On call
//   - ctx context.Context
//   - blobContent []byte
func (_e *MockBlobConsumer_Expecter) Consume(ctx interface{}, blobContent interface{}) *MockBlobConsumer_Consume_Call {
	return &MockBlobConsumer_Consume_Call{Call: _e.mock.On("Consume", ctx, blobContent)}
}

func (_c *MockBlobConsumer_Consume_Call) Run(run func(ctx context.Context, blobContent []byte)) *MockBlobConsumer_Consume_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]byte))
	})
	return _c
}

func (_c *MockBlobConsumer_Consume_Call) Return(_a0 error) *MockBlobConsumer_Consume_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockBlobConsumer_Consume_Call) RunAndReturn(run func(context.Context, []byte) error) *MockBlobConsumer_Consume_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockBlobConsumer creates a new instance of MockBlobConsumer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockBlobConsumer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockBlobConsumer {
	mock := &MockBlobConsumer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
