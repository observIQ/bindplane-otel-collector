// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	mock "github.com/stretchr/testify/mock"
)

// MockS3Client is an autogenerated mock type for the S3Client type
type MockS3Client struct {
	mock.Mock
}

type MockS3Client_Expecter struct {
	mock *mock.Mock
}

func (_m *MockS3Client) EXPECT() *MockS3Client_Expecter {
	return &MockS3Client_Expecter{mock: &_m.Mock}
}

// GetObject provides a mock function with given fields: ctx, params, optFns
func (_m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	_va := make([]interface{}, len(optFns))
	for _i := range optFns {
		_va[_i] = optFns[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, params)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetObject")
	}

	var r0 *s3.GetObjectOutput
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)); ok {
		return rf(ctx, params, optFns...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) *s3.GetObjectOutput); ok {
		r0 = rf(ctx, params, optFns...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*s3.GetObjectOutput)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) error); ok {
		r1 = rf(ctx, params, optFns...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockS3Client_GetObject_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetObject'
type MockS3Client_GetObject_Call struct {
	*mock.Call
}

// GetObject is a helper method to define mock.On call
//   - ctx context.Context
//   - params *s3.GetObjectInput
//   - optFns ...func(*s3.Options)
func (_e *MockS3Client_Expecter) GetObject(ctx interface{}, params interface{}, optFns ...interface{}) *MockS3Client_GetObject_Call {
	return &MockS3Client_GetObject_Call{Call: _e.mock.On("GetObject",
		append([]interface{}{ctx, params}, optFns...)...)}
}

func (_c *MockS3Client_GetObject_Call) Run(run func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options))) *MockS3Client_GetObject_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]func(*s3.Options), len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(func(*s3.Options))
			}
		}
		run(args[0].(context.Context), args[1].(*s3.GetObjectInput), variadicArgs...)
	})
	return _c
}

func (_c *MockS3Client_GetObject_Call) Return(_a0 *s3.GetObjectOutput, _a1 error) *MockS3Client_GetObject_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockS3Client_GetObject_Call) RunAndReturn(run func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)) *MockS3Client_GetObject_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockS3Client creates a new instance of MockS3Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockS3Client(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockS3Client {
	mock := &MockS3Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
