// Code generated by mockery v2.51.0. DO NOT EDIT.

package mocks

import (
	context "context"

	azureblob "github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver/internal/azureblob"

	mock "github.com/stretchr/testify/mock"
)

// MockBlobClient is an autogenerated mock type for the BlobClient type
type MockBlobClient struct {
	mock.Mock
}

type MockBlobClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockBlobClient) EXPECT() *MockBlobClient_Expecter {
	return &MockBlobClient_Expecter{mock: &_m.Mock}
}

// DeleteBlob provides a mock function with given fields: ctx, container, blobPath
func (_m *MockBlobClient) DeleteBlob(ctx context.Context, container string, blobPath string) error {
	ret := _m.Called(ctx, container, blobPath)

	if len(ret) == 0 {
		panic("no return value specified for DeleteBlob")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, container, blobPath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockBlobClient_DeleteBlob_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteBlob'
type MockBlobClient_DeleteBlob_Call struct {
	*mock.Call
}

// DeleteBlob is a helper method to define mock.On call
//   - ctx context.Context
//   - container string
//   - blobPath string
func (_e *MockBlobClient_Expecter) DeleteBlob(ctx interface{}, container interface{}, blobPath interface{}) *MockBlobClient_DeleteBlob_Call {
	return &MockBlobClient_DeleteBlob_Call{Call: _e.mock.On("DeleteBlob", ctx, container, blobPath)}
}

func (_c *MockBlobClient_DeleteBlob_Call) Run(run func(ctx context.Context, container string, blobPath string)) *MockBlobClient_DeleteBlob_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockBlobClient_DeleteBlob_Call) Return(_a0 error) *MockBlobClient_DeleteBlob_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockBlobClient_DeleteBlob_Call) RunAndReturn(run func(context.Context, string, string) error) *MockBlobClient_DeleteBlob_Call {
	_c.Call.Return(run)
	return _c
}

// DownloadBlob provides a mock function with given fields: ctx, container, blobPath, buf
func (_m *MockBlobClient) DownloadBlob(ctx context.Context, container string, blobPath string, buf []byte) (int64, error) {
	ret := _m.Called(ctx, container, blobPath, buf)

	if len(ret) == 0 {
		panic("no return value specified for DownloadBlob")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []byte) (int64, error)); ok {
		return rf(ctx, container, blobPath, buf)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []byte) int64); ok {
		r0 = rf(ctx, container, blobPath, buf)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, []byte) error); ok {
		r1 = rf(ctx, container, blobPath, buf)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBlobClient_DownloadBlob_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DownloadBlob'
type MockBlobClient_DownloadBlob_Call struct {
	*mock.Call
}

// DownloadBlob is a helper method to define mock.On call
//   - ctx context.Context
//   - container string
//   - blobPath string
//   - buf []byte
func (_e *MockBlobClient_Expecter) DownloadBlob(ctx interface{}, container interface{}, blobPath interface{}, buf interface{}) *MockBlobClient_DownloadBlob_Call {
	return &MockBlobClient_DownloadBlob_Call{Call: _e.mock.On("DownloadBlob", ctx, container, blobPath, buf)}
}

func (_c *MockBlobClient_DownloadBlob_Call) Run(run func(ctx context.Context, container string, blobPath string, buf []byte)) *MockBlobClient_DownloadBlob_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].([]byte))
	})
	return _c
}

func (_c *MockBlobClient_DownloadBlob_Call) Return(_a0 int64, _a1 error) *MockBlobClient_DownloadBlob_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockBlobClient_DownloadBlob_Call) RunAndReturn(run func(context.Context, string, string, []byte) (int64, error)) *MockBlobClient_DownloadBlob_Call {
	_c.Call.Return(run)
	return _c
}

// StreamBlobs provides a mock function with given fields: ctx, container, prefix, errChan, blobChan, doneChan
func (_m *MockBlobClient) StreamBlobs(ctx context.Context, container string, prefix *string, errChan chan error, blobChan chan []*azureblob.BlobInfo, doneChan chan struct{}) {
	_m.Called(ctx, container, prefix, errChan, blobChan, doneChan)
}

// MockBlobClient_StreamBlobs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StreamBlobs'
type MockBlobClient_StreamBlobs_Call struct {
	*mock.Call
}

// StreamBlobs is a helper method to define mock.On call
//   - ctx context.Context
//   - container string
//   - prefix *string
//   - errChan chan error
//   - blobChan chan []*azureblob.BlobInfo
//   - doneChan chan struct{}
func (_e *MockBlobClient_Expecter) StreamBlobs(ctx interface{}, container interface{}, prefix interface{}, errChan interface{}, blobChan interface{}, doneChan interface{}) *MockBlobClient_StreamBlobs_Call {
	return &MockBlobClient_StreamBlobs_Call{Call: _e.mock.On("StreamBlobs", ctx, container, prefix, errChan, blobChan, doneChan)}
}

func (_c *MockBlobClient_StreamBlobs_Call) Run(run func(ctx context.Context, container string, prefix *string, errChan chan error, blobChan chan []*azureblob.BlobInfo, doneChan chan struct{})) *MockBlobClient_StreamBlobs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(*string), args[3].(chan error), args[4].(chan []*azureblob.BlobInfo), args[5].(chan struct{}))
	})
	return _c
}

func (_c *MockBlobClient_StreamBlobs_Call) Return() *MockBlobClient_StreamBlobs_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockBlobClient_StreamBlobs_Call) RunAndReturn(run func(context.Context, string, *string, chan error, chan []*azureblob.BlobInfo, chan struct{})) *MockBlobClient_StreamBlobs_Call {
	_c.Run(run)
	return _c
}

// NewMockBlobClient creates a new instance of MockBlobClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockBlobClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockBlobClient {
	mock := &MockBlobClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
