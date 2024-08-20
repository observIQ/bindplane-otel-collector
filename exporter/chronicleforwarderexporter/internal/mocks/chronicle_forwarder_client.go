// Code generated by mockery v2.44.2. DO NOT EDIT.

package mocks

import (
	net "net"

	mock "github.com/stretchr/testify/mock"

	os "os"

	tls "crypto/tls"
)

// MockForwarderClient is an autogenerated mock type for the chronicleForwarderClient type
type MockForwarderClient struct {
	mock.Mock
}

type MockForwarderClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockForwarderClient) EXPECT() *MockForwarderClient_Expecter {
	return &MockForwarderClient_Expecter{mock: &_m.Mock}
}

// Dial provides a mock function with given fields: network, address
func (_m *MockForwarderClient) Dial(network string, address string) (net.Conn, error) {
	ret := _m.Called(network, address)

	if len(ret) == 0 {
		panic("no return value specified for Dial")
	}

	var r0 net.Conn
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (net.Conn, error)); ok {
		return rf(network, address)
	}
	if rf, ok := ret.Get(0).(func(string, string) net.Conn); ok {
		r0 = rf(network, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(net.Conn)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(network, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockForwarderClient_Dial_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Dial'
type MockForwarderClient_Dial_Call struct {
	*mock.Call
}

// Dial is a helper method to define mock.On call
//   - network string
//   - address string
func (_e *MockForwarderClient_Expecter) Dial(network interface{}, address interface{}) *MockForwarderClient_Dial_Call {
	return &MockForwarderClient_Dial_Call{Call: _e.mock.On("Dial", network, address)}
}

func (_c *MockForwarderClient_Dial_Call) Run(run func(network string, address string)) *MockForwarderClient_Dial_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string))
	})
	return _c
}

func (_c *MockForwarderClient_Dial_Call) Return(_a0 net.Conn, _a1 error) *MockForwarderClient_Dial_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockForwarderClient_Dial_Call) RunAndReturn(run func(string, string) (net.Conn, error)) *MockForwarderClient_Dial_Call {
	_c.Call.Return(run)
	return _c
}

// DialWithTLS provides a mock function with given fields: network, addr, config
func (_m *MockForwarderClient) DialWithTLS(network string, addr string, config *tls.Config) (*tls.Conn, error) {
	ret := _m.Called(network, addr, config)

	if len(ret) == 0 {
		panic("no return value specified for DialWithTLS")
	}

	var r0 *tls.Conn
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string, *tls.Config) (*tls.Conn, error)); ok {
		return rf(network, addr, config)
	}
	if rf, ok := ret.Get(0).(func(string, string, *tls.Config) *tls.Conn); ok {
		r0 = rf(network, addr, config)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tls.Conn)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string, *tls.Config) error); ok {
		r1 = rf(network, addr, config)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockForwarderClient_DialWithTLS_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DialWithTLS'
type MockForwarderClient_DialWithTLS_Call struct {
	*mock.Call
}

// DialWithTLS is a helper method to define mock.On call
//   - network string
//   - addr string
//   - config *tls.Config
func (_e *MockForwarderClient_Expecter) DialWithTLS(network interface{}, addr interface{}, config interface{}) *MockForwarderClient_DialWithTLS_Call {
	return &MockForwarderClient_DialWithTLS_Call{Call: _e.mock.On("DialWithTLS", network, addr, config)}
}

func (_c *MockForwarderClient_DialWithTLS_Call) Run(run func(network string, addr string, config *tls.Config)) *MockForwarderClient_DialWithTLS_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string), args[2].(*tls.Config))
	})
	return _c
}

func (_c *MockForwarderClient_DialWithTLS_Call) Return(_a0 *tls.Conn, _a1 error) *MockForwarderClient_DialWithTLS_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockForwarderClient_DialWithTLS_Call) RunAndReturn(run func(string, string, *tls.Config) (*tls.Conn, error)) *MockForwarderClient_DialWithTLS_Call {
	_c.Call.Return(run)
	return _c
}

// OpenFile provides a mock function with given fields: name
func (_m *MockForwarderClient) OpenFile(name string) (*os.File, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for OpenFile")
	}

	var r0 *os.File
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*os.File, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *os.File); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*os.File)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockForwarderClient_OpenFile_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OpenFile'
type MockForwarderClient_OpenFile_Call struct {
	*mock.Call
}

// OpenFile is a helper method to define mock.On call
//   - name string
func (_e *MockForwarderClient_Expecter) OpenFile(name interface{}) *MockForwarderClient_OpenFile_Call {
	return &MockForwarderClient_OpenFile_Call{Call: _e.mock.On("OpenFile", name)}
}

func (_c *MockForwarderClient_OpenFile_Call) Run(run func(name string)) *MockForwarderClient_OpenFile_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockForwarderClient_OpenFile_Call) Return(_a0 *os.File, _a1 error) *MockForwarderClient_OpenFile_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockForwarderClient_OpenFile_Call) RunAndReturn(run func(string) (*os.File, error)) *MockForwarderClient_OpenFile_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockForwarderClient creates a new instance of MockForwarderClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockForwarderClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockForwarderClient {
	mock := &MockForwarderClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
