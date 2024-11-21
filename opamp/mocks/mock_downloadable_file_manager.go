// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	protobufs "github.com/open-telemetry/opamp-go/protobufs"
)

// MockDownloadableFileManager is an autogenerated mock type for the DownloadableFileManager type
type MockDownloadableFileManager struct {
	mock.Mock
}

// CleanupArtifacts provides a mock function with given fields:
func (_m *MockDownloadableFileManager) CleanupArtifacts() {
	_m.Called()
}

// FetchAndExtractArchive provides a mock function with given fields: _a0
func (_m *MockDownloadableFileManager) FetchAndExtractArchive(_a0 *protobufs.DownloadableFile) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for FetchAndExtractArchive")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*protobufs.DownloadableFile) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockDownloadableFileManager creates a new instance of MockDownloadableFileManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDownloadableFileManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDownloadableFileManager {
	mock := &MockDownloadableFileManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
