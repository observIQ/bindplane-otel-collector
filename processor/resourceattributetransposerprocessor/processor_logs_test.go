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

package resourceattributetransposerprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestLogsProcessorStart(t *testing.T) {
	p := newLogsProcessor(
		zap.NewNop(),
		consumertest.NewNop(),
		createDefaultConfig().(*Config),
	)

	err := p.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
}

func TestLogsProcessorShutdown(t *testing.T) {
	p := newLogsProcessor(
		zap.NewNop(),
		consumertest.NewNop(),
		createDefaultConfig().(*Config),
	)

	err := p.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestLogsProcessorCapabilities(t *testing.T) {
	p := newLogsProcessor(
		zap.NewNop(),
		consumertest.NewNop(),
		createDefaultConfig().(*Config),
	)
	capabilities := p.Capabilities()
	require.True(t, capabilities.MutatesData)
}

func TestConsumeLogs(t *testing.T) {
	ctx := context.Background()
	logs := createLogs()

	attrs := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value"))
	attrs.Insert("resourceattrib2", pcommon.NewValueBool(false))
	attrs.Insert("resourceattrib3", pcommon.NewValueBytes(pcommon.NewImmutableByteSlice([]byte("some bytes"))))
	attrs.Insert("resourceattrib4", pcommon.NewValueDouble(2.0))
	attrs.Insert("resourceattrib5", pcommon.NewValueInt(100))
	attrs.Insert("resourceattrib6", pcommon.NewValueEmpty())

	var logsOut plog.Logs

	consumer := &mockLogsConsumer{}
	consumer.On("ConsumeLogs", ctx, logs).Run(func(args mock.Arguments) {
		logsOut = args[1].(plog.Logs)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1",
		},
		{
			From: "resourceattrib2",
			To:   "resourceattrib2",
		},
		{
			From: "resourceattrib3",
			To:   "resourceattrib3",
		},
		{
			From: "resourceattrib4",
			To:   "resourceattrib4",
		},
		{
			From: "resourceattrib5",
			To:   "resourceattrib5",
		},
		{
			From: "resourceattrib6",
			To:   "resourceattrib6",
		},
		{
			From: "resourceattrib7",
			To:   "resourceattrib7",
		},
	}

	p := newLogsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	err := p.ConsumeLogs(ctx, logs)
	require.NoError(t, err)

	require.Equal(t, map[string]any{
		"resourceattrib1": "value",
		"resourceattrib2": false,
		"resourceattrib3": []byte("some bytes"),
		"resourceattrib4": float64(2.0),
		"resourceattrib5": int64(100),
		"resourceattrib6": nil,
	}, logsOut.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().AsRaw())
}

func TestConsumeLogsMoveToMultipleMetrics(t *testing.T) {
	ctx := context.Background()
	logs := createLogs()

	attrs := logs.ResourceLogs().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value"))

	var logsOut plog.Logs

	consumer := &mockLogsConsumer{}
	consumer.On("ConsumeLogs", ctx, logs).Run(func(args mock.Arguments) {
		logsOut = args[1].(plog.Logs)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1",
		},
	}

	p := newLogsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty()

	err := p.ConsumeLogs(ctx, logs)
	require.NoError(t, err)

	require.Equal(t, map[string]any{
		"resourceattrib1": "value",
	}, logsOut.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().AsRaw())

	require.Equal(t, map[string]any{
		"resourceattrib1": "value",
	}, logsOut.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(1).Attributes().AsRaw())
}

func TestConsumeLogsDoesNotOverwrite(t *testing.T) {
	// Tests that subsequent operations do not overwrite values written
	// by previous options (list order is respected)
	ctx := context.Background()
	logs := createLogs()

	attrs := logs.ResourceLogs().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))
	attrs.Insert("resourceattrib2", pcommon.NewValueString("value2"))

	var logsOut plog.Logs

	consumer := &mockLogsConsumer{}
	consumer.On("ConsumeLogs", ctx, logs).Run(func(args mock.Arguments) {
		logsOut = args[1].(plog.Logs)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "out",
		},
		{
			From: "resourceattrib2",
			To:   "out",
		},
	}

	p := newLogsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	err := p.ConsumeLogs(ctx, logs)
	require.NoError(t, err)

	require.Equal(t, map[string]any{
		"out": "value1",
	}, logsOut.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().AsRaw())
}

func TestConsumeLogsDoesNotOverwrite2(t *testing.T) {
	// Tests that operations will not overwrite previously filled in attributes.
	ctx := context.Background()
	logs := createLogs()

	attrs := logs.ResourceLogs().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))
	attrs.Insert("resourceattrib2", pcommon.NewValueString("value2"))

	var logsOut plog.Logs

	consumer := &mockLogsConsumer{}
	consumer.On("ConsumeLogs", ctx, logs).Run(func(args mock.Arguments) {
		logsOut = args[1].(plog.Logs)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "out",
		},
		{
			From: "resourceattrib2",
			To:   "out",
		},
	}

	p := newLogsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().InsertString("out", "originalvalue")

	err := p.ConsumeLogs(ctx, logs)
	require.NoError(t, err)

	require.Equal(t, map[string]any{
		"out": "originalvalue",
	}, logsOut.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().AsRaw())
}

func createLogs() plog.Logs {
	logs := plog.NewLogs()
	logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	return logs
}

type mockLogsConsumer struct {
	mock.Mock
}

func (m *mockLogsConsumer) Start(ctx context.Context, host component.Host) error {
	args := m.Called(ctx, host)
	return args.Error(0)
}

func (m *mockLogsConsumer) Capabilities() consumer.Capabilities {
	args := m.Called()
	return args.Get(0).(consumer.Capabilities)
}

func (m *mockLogsConsumer) ConsumeLogs(ctx context.Context, md plog.Logs) error {
	args := m.Called(ctx, md)
	return args.Error(0)
}

func (m *mockLogsConsumer) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
