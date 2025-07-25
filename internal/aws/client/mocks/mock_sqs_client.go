// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	sqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	mock "github.com/stretchr/testify/mock"
)

// MockSQSClient is an autogenerated mock type for the SQSClient type
type MockSQSClient struct {
	mock.Mock
}

type MockSQSClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSQSClient) EXPECT() *MockSQSClient_Expecter {
	return &MockSQSClient_Expecter{mock: &_m.Mock}
}

// ChangeMessageVisibility provides a mock function with given fields: ctx, params, optFns
func (_m *MockSQSClient) ChangeMessageVisibility(ctx context.Context, params *sqs.ChangeMessageVisibilityInput, optFns ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error) {
	_va := make([]interface{}, len(optFns))
	for _i := range optFns {
		_va[_i] = optFns[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, params)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for ChangeMessageVisibility")
	}

	var r0 *sqs.ChangeMessageVisibilityOutput
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sqs.ChangeMessageVisibilityInput, ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error)); ok {
		return rf(ctx, params, optFns...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sqs.ChangeMessageVisibilityInput, ...func(*sqs.Options)) *sqs.ChangeMessageVisibilityOutput); ok {
		r0 = rf(ctx, params, optFns...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sqs.ChangeMessageVisibilityOutput)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sqs.ChangeMessageVisibilityInput, ...func(*sqs.Options)) error); ok {
		r1 = rf(ctx, params, optFns...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockSQSClient_ChangeMessageVisibility_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ChangeMessageVisibility'
type MockSQSClient_ChangeMessageVisibility_Call struct {
	*mock.Call
}

// ChangeMessageVisibility is a helper method to define mock.On call
//   - ctx context.Context
//   - params *sqs.ChangeMessageVisibilityInput
//   - optFns ...func(*sqs.Options)
func (_e *MockSQSClient_Expecter) ChangeMessageVisibility(ctx interface{}, params interface{}, optFns ...interface{}) *MockSQSClient_ChangeMessageVisibility_Call {
	return &MockSQSClient_ChangeMessageVisibility_Call{Call: _e.mock.On("ChangeMessageVisibility",
		append([]interface{}{ctx, params}, optFns...)...)}
}

func (_c *MockSQSClient_ChangeMessageVisibility_Call) Run(run func(ctx context.Context, params *sqs.ChangeMessageVisibilityInput, optFns ...func(*sqs.Options))) *MockSQSClient_ChangeMessageVisibility_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]func(*sqs.Options), len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(func(*sqs.Options))
			}
		}
		run(args[0].(context.Context), args[1].(*sqs.ChangeMessageVisibilityInput), variadicArgs...)
	})
	return _c
}

func (_c *MockSQSClient_ChangeMessageVisibility_Call) Return(_a0 *sqs.ChangeMessageVisibilityOutput, _a1 error) *MockSQSClient_ChangeMessageVisibility_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSQSClient_ChangeMessageVisibility_Call) RunAndReturn(run func(context.Context, *sqs.ChangeMessageVisibilityInput, ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error)) *MockSQSClient_ChangeMessageVisibility_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteMessage provides a mock function with given fields: ctx, params, optFns
func (_m *MockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	_va := make([]interface{}, len(optFns))
	for _i := range optFns {
		_va[_i] = optFns[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, params)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for DeleteMessage")
	}

	var r0 *sqs.DeleteMessageOutput
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)); ok {
		return rf(ctx, params, optFns...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) *sqs.DeleteMessageOutput); ok {
		r0 = rf(ctx, params, optFns...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sqs.DeleteMessageOutput)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) error); ok {
		r1 = rf(ctx, params, optFns...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockSQSClient_DeleteMessage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteMessage'
type MockSQSClient_DeleteMessage_Call struct {
	*mock.Call
}

// DeleteMessage is a helper method to define mock.On call
//   - ctx context.Context
//   - params *sqs.DeleteMessageInput
//   - optFns ...func(*sqs.Options)
func (_e *MockSQSClient_Expecter) DeleteMessage(ctx interface{}, params interface{}, optFns ...interface{}) *MockSQSClient_DeleteMessage_Call {
	return &MockSQSClient_DeleteMessage_Call{Call: _e.mock.On("DeleteMessage",
		append([]interface{}{ctx, params}, optFns...)...)}
}

func (_c *MockSQSClient_DeleteMessage_Call) Run(run func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options))) *MockSQSClient_DeleteMessage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]func(*sqs.Options), len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(func(*sqs.Options))
			}
		}
		run(args[0].(context.Context), args[1].(*sqs.DeleteMessageInput), variadicArgs...)
	})
	return _c
}

func (_c *MockSQSClient_DeleteMessage_Call) Return(_a0 *sqs.DeleteMessageOutput, _a1 error) *MockSQSClient_DeleteMessage_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSQSClient_DeleteMessage_Call) RunAndReturn(run func(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)) *MockSQSClient_DeleteMessage_Call {
	_c.Call.Return(run)
	return _c
}

// ReceiveMessage provides a mock function with given fields: ctx, params, optFns
func (_m *MockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	_va := make([]interface{}, len(optFns))
	for _i := range optFns {
		_va[_i] = optFns[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, params)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for ReceiveMessage")
	}

	var r0 *sqs.ReceiveMessageOutput
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)); ok {
		return rf(ctx, params, optFns...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) *sqs.ReceiveMessageOutput); ok {
		r0 = rf(ctx, params, optFns...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sqs.ReceiveMessageOutput)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) error); ok {
		r1 = rf(ctx, params, optFns...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockSQSClient_ReceiveMessage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReceiveMessage'
type MockSQSClient_ReceiveMessage_Call struct {
	*mock.Call
}

// ReceiveMessage is a helper method to define mock.On call
//   - ctx context.Context
//   - params *sqs.ReceiveMessageInput
//   - optFns ...func(*sqs.Options)
func (_e *MockSQSClient_Expecter) ReceiveMessage(ctx interface{}, params interface{}, optFns ...interface{}) *MockSQSClient_ReceiveMessage_Call {
	return &MockSQSClient_ReceiveMessage_Call{Call: _e.mock.On("ReceiveMessage",
		append([]interface{}{ctx, params}, optFns...)...)}
}

func (_c *MockSQSClient_ReceiveMessage_Call) Run(run func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options))) *MockSQSClient_ReceiveMessage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]func(*sqs.Options), len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(func(*sqs.Options))
			}
		}
		run(args[0].(context.Context), args[1].(*sqs.ReceiveMessageInput), variadicArgs...)
	})
	return _c
}

func (_c *MockSQSClient_ReceiveMessage_Call) Return(_a0 *sqs.ReceiveMessageOutput, _a1 error) *MockSQSClient_ReceiveMessage_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSQSClient_ReceiveMessage_Call) RunAndReturn(run func(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)) *MockSQSClient_ReceiveMessage_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockSQSClient creates a new instance of MockSQSClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSQSClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSQSClient {
	mock := &MockSQSClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
