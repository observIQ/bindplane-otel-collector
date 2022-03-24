// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package varnishreceiver // import "github.com/observiq/observiq-otel-collector/receiver/varnishreceiver"

import (
	"errors"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap"
)

func TestNewVarnishClient(t *testing.T) {
	client := newVarnishClient(
		createDefaultConfig().(*Config),
		componenttest.NewNopHost(),
		componenttest.NewNopTelemetrySettings())
	require.NotNil(t, client)
}

func TestBuildCommand(t *testing.T) {
	testCases := []struct {
		desc    string
		config  Config
		command string
		argList []string
	}{
		{
			desc: "without exec dir and with default host",
			config: Config{
				CacheDir: "defaultHostName",
			},
			command: "varnishstat",
			argList: []string{"-j", "-n", "defaultHostName"},
		},
		{
			desc: "without exec dir and with cache dir",
			config: Config{
				CacheDir: "/path/varnishinstance",
			},
			command: "varnishstat",
			argList: []string{"-j", "-n", "/path/varnishinstance"},
		},
		{
			desc: "with exec dir and cache dir",
			config: Config{
				CacheDir: "/path/varnishinstance",
				ExecDir:  "/exec/dir/varnishstat",
			},
			command: "/exec/dir/varnishstat",
			argList: []string{"-j", "-n", "/path/varnishinstance"},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			client := varnishClient{
				cfg: &tC.config,
			}
			command, argList := client.BuildCommand()
			require.EqualValues(t, tC.command, command)
			require.EqualValues(t, tC.argList, argList)
		})
	}
}

func getBytes(t *testing.T, filename string) ([]byte, error) {
	t.Helper()
	if filename == "" {
		return nil, errors.New("bad response")
	}

	body, err := os.ReadFile(path.Join("testdata", "scraper", filename))
	if err != nil {
		return nil, err
	}

	return body, nil
}

func TestGetStats(t *testing.T) {
	mockExec := new(mockExecuter)
	mockExec.On("Execute", "varnishstat", []string{"-j", "-n", "/path/varnishinstance"}).Return(getBytes(t, "mock_response6_0.json"))
	myclient := varnishClient{
		exec:   mockExec,
		cfg:    createDefaultConfig().(*Config),
		logger: zap.NewNop(),
	}
	myclient.cfg.CacheDir = "/path/varnishinstance"
	stats, err := myclient.GetStats()
	require.NoError(t, err)
	require.NotNil(t, stats)

	mockExecuter6_5 := new(mockExecuter)
	mockExecuter6_5.On("Execute", "varnishstat", []string{"-j", "-n", "/path/varnishinstance"}).Return(getBytes(t, "mock_response6_5.json"))
	myclient6_5 := varnishClient{
		exec:   mockExecuter6_5,
		cfg:    createDefaultConfig().(*Config),
		logger: zap.NewNop(),
	}
	myclient6_5.cfg.CacheDir = "/path/varnishinstance"
	stats6_5, err := myclient6_5.GetStats()
	require.NoError(t, err)
	require.NotNil(t, stats)
	require.EqualValues(t, stats, stats6_5)
}

type mockExecuter struct {
	mock.Mock
}

// Execute provides a mock function with given fields: command, args
func (_m *mockExecuter) Execute(command string, args []string) ([]byte, error) {
	ret := _m.Called(command, args)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(string, []string) []byte); ok {
		r0 = rf(command, args)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, []string) error); ok {
		r1 = rf(command, args)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
