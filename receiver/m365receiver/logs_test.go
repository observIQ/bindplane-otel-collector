// Copyright observIQ, Inc.
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

package m365receiver // import "github.com/observiq/observiq-otel-collector/receiver/m365receiver"

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestStartPolling(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.TenantID = "testTenantID"
	cfg.Logs.PollInterval = 1 * time.Second

	sink := &consumertest.LogsSink{}
	l := newM365Logs(cfg, receivertest.NewNopCreateSettings(), sink)
	client := &mockLogsClient{}
	l.client = client
	file := filepath.Join("testdata", "logs", "testPollLogs", "input.json")
	client.On("GetJSON", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(client.loadTestLogs(t, file), nil)
	cancelCtx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.record = &logRecord{}

	err := l.startPolling(cancelCtx)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() > 0
	}, 5*time.Second, 1*time.Second)

	err = l.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestPollLogs(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.TenantID = "testTenantID"
	cfg.Logs.PollInterval = 1 * time.Second

	sink := &consumertest.LogsSink{}
	rcv := newM365Logs(cfg, receivertest.NewNopCreateSettings(), sink)
	client := &mockLogsClient{}
	rcv.client = client
	file := filepath.Join("testdata", "logs", "testPollLogs", "input.json")
	client.On("GetJSON", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(client.loadTestLogs(t, file), nil)
	rcv.record = &logRecord{}

	err := rcv.pollLogs(context.Background())
	require.NoError(t, err)

	logs := sink.AllLogs()

	// write logs for each service resource
	// for _, l := range logs {
	// 	audit, exist := l.ResourceLogs().At(0).Resource().Attributes().Get("m365.audit")
	// 	require.True(t, exist)

	// 	marshaler := &plog.JSONMarshaler{}
	// 	lBytes, err := marshaler.MarshalLogs(l)
	// 	require.NoError(t, err)
	// 	err = os.WriteFile(filepath.Join("testdata", "logs", "testPollLogs", fmt.Sprintf("%s.json", audit.Str())), lBytes, 0666)
	// 	require.NoError(t, err)
	// }

	// compare logs for each service resource
	for _, l := range logs {
		audit, exist := l.ResourceLogs().At(0).Resource().Attributes().Get("m365.audit")
		require.True(t, exist)

		expected, err := ReadLogs(filepath.Join("testdata", "logs", "testPollLogs", fmt.Sprintf("%s.json", audit.Str())))
		require.NoError(t, err)
		require.NoError(t, plogtest.CompareLogs(expected, l, plogtest.IgnoreObservedTimestamp()))
	}
}

func TestPollErrHandle(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.TenantID = "test"
	sink := &consumertest.LogsSink{}
	client := &mockLogsClient{}
	audit := auditMetaData{
		name:    "general",
		route:   "Audit.General",
		enabled: true,
	}
	wg := &sync.WaitGroup{}
	rcv := newM365Logs(cfg, receivertest.NewNopCreateSettings(), sink)
	rcv.client = client

	// unable to fix token
	client.On("GetJSON", mock.Anything, rcv.root+"Audit.General", mock.Anything, mock.Anything).Return([]logData{}, fmt.Errorf("authorization denied")).Once()
	client.On("GetToken", mock.Anything).Return(fmt.Errorf("err")).Once()
	wg.Add(1)
	rcv.poll(context.Background(), time.Now(), "", &audit, wg)
	require.Never(t, func() bool {
		return sink.LogRecordCount() > 0
	}, 3*time.Second, 1*time.Second)

	// GetJSON still doesn't work
	client.On("GetJSON", mock.Anything, rcv.root+"Audit.General", mock.Anything, mock.Anything).Return([]logData{}, fmt.Errorf("authorization denied")).Once()
	client.On("GetToken", mock.Anything).Return(nil).Once()
	client.On("GetJSON", mock.Anything, rcv.root+"Audit.General", mock.Anything, mock.Anything).Return([]logData{}, fmt.Errorf("err")).Once()
	wg.Add(1)
	rcv.poll(context.Background(), time.Now(), "", &audit, wg)
	require.Never(t, func() bool {
		return sink.LogRecordCount() > 0
	}, 3*time.Second, 1*time.Second)

	// GetJSON never worked
	client.On("GetJSON", mock.Anything, rcv.root+"Audit.General", mock.Anything, mock.Anything).Return([]logData{}, fmt.Errorf("err")).Once()
	wg.Add(1)
	rcv.poll(context.Background(), time.Now(), "", &audit, wg)
	require.Never(t, func() bool {
		return sink.LogRecordCount() > 0
	}, 3*time.Second, 1*time.Second)

	// regenerate token works
	file := filepath.Join("testdata", "logs", "testPollLogs", "input.json")
	client.On("GetJSON", mock.Anything, rcv.root+"Audit.General", mock.Anything, mock.Anything).Return([]logData{}, fmt.Errorf("authorization denied")).Once()
	client.On("GetJSON", mock.Anything, rcv.root+"Audit.General", mock.Anything, mock.Anything).Return(client.loadTestLogs(t, file), nil).Once()
	client.On("GetToken", mock.Anything).Return(nil).Once()
	wg.Add(1)
	rcv.poll(context.Background(), time.Now(), "", &audit, wg)
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() > 0
	}, 5*time.Second, 1*time.Second)
}

func TestParseOptionalAttributes(t *testing.T) {
	m := pcommon.NewMap()
	log := jsonLog{
		Workload:     "testWorkload",
		ResultStatus: "",
	}
	parseOptionalAttributes(&m, &log)
	w, e := m.Get("workload")
	require.True(t, e)
	require.Equal(t, "testWorkload", w.AsString())
	w, e = m.Get("result_status")
	require.False(t, e)
}

func ReadLogs(filepath string) (plog.Logs, error) {
	expectedFileBytes, err := os.ReadFile(filepath)
	if err != nil {
		return plog.Logs{}, nil
	}
	unmarshaler := &plog.JSONUnmarshaler{}
	return unmarshaler.UnmarshalLogs(expectedFileBytes)
}

type mockLogsClient struct {
	mock.Mock
}

func (mc *mockLogsClient) loadTestLogs(t *testing.T, file string) []logData {
	logBytes, err := os.ReadFile(file)
	require.NoError(t, err)

	var logs []jsonLog
	err = json.Unmarshal(logBytes, &logs)
	require.NoError(t, err)

	data := strings.Split(string(logBytes), "},{\"C")
	last := len(data) - 1
	data[0] = strings.TrimPrefix(data[0], "[{\"C")
	data[last] = strings.TrimSuffix(data[last], "}]")
	for i := 0; i < len(data); i++ {
		foo := "{\"C" + data[i] + "}"
		data[i] = foo
	}
	data[last] = strings.TrimSuffix(data[last], ",")

	var ret = []logData{}
	for _, b := range data {
		var log = jsonLog{}
		err = json.Unmarshal([]byte(b), &log)
		ret = append(ret, logData{log: log, body: b})
	}

	return ret
}

func (mc *mockLogsClient) GetJSON(ctx context.Context, endpoint string, end string, start string) ([]logData, error) {
	args := mc.Called(ctx, endpoint, end, start)
	return args.Get(0).([]logData), args.Error(1)
}

func (mc *mockLogsClient) GetToken(ctx context.Context) error {
	args := mc.Called(ctx)
	return args.Error(0)
}

func (mc *mockLogsClient) StartSubscription(_ context.Context, _ string) error {
	return nil
}

func (mc *mockLogsClient) shutdown() error {
	return nil
}
