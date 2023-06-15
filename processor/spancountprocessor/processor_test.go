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

package spancountprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/observiq-otel-collector/receiver/routereceiver"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestProcessorCapabilities(t *testing.T) {
	p := &spanCountProcessor{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, p.Capabilities())
}

func TestConsumeTraces(t *testing.T) {
	countMetricConsumer := &consumertest.MetricsSink{}
	nextTracesConsumer := &consumertest.TracesSink{}

	processorCfg := createDefaultConfig().(*Config)
	processorCfg.Interval = time.Millisecond * 100
	processorCfg.Match = strp(`span_duration_ms > 1000 and resource["service.name"] == "test2"`)
	processorCfg.Attributes = map[string]string{
		"dimension1": `span_duration_ms`,
		"dimension2": `resource["service.name"]`,
	}

	processorFactory := NewFactory()
	processorSettings := processor.CreateSettings{TelemetrySettings: component.TelemetrySettings{Logger: zaptest.NewLogger(t)}}
	processor, err := processorFactory.CreateTracesProcessor(context.Background(), processorSettings, processorCfg, nextTracesConsumer)
	require.NoError(t, err)

	receiverFactory := routereceiver.NewFactory()
	receiver, err := receiverFactory.CreateMetricsReceiver(context.Background(), receiver.CreateSettings{}, receiverFactory.CreateDefaultConfig(), countMetricConsumer)
	require.NoError(t, err)

	err = processor.Start(context.Background(), nil)
	require.NoError(t, err)
	defer processor.Shutdown(context.Background())

	err = receiver.Start(context.Background(), nil)
	require.NoError(t, err)
	defer receiver.Shutdown(context.Background())

	now := time.Now()
	twoSecondsAgo := now.Add(-2 * time.Second)

	traces := ptrace.NewTraces()
	resourceLogs := traces.ResourceSpans().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(map[string]any{"service.name": "test2"})
	span := resourceLogs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().FromRaw(map[string]any{"dimension1": "test1", "dimension2": "test2"})
	span.Status().SetCode(ptrace.StatusCodeOk)
	span.SetKind(ptrace.SpanKindClient)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(twoSecondsAgo))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(now))

	processor.ConsumeTraces(context.Background(), traces)

	passedTraces := nextTracesConsumer.AllTraces()[0]
	require.Equal(t, traces, passedTraces)

	require.Eventually(t, func() bool {
		return len(countMetricConsumer.AllMetrics()) > 0
	}, 5*time.Second, 200*time.Millisecond)

	countMetrics := countMetricConsumer.AllMetrics()[0]
	require.Equal(t, 1, countMetrics.ResourceMetrics().Len())

	countResourceMetrics := countMetrics.ResourceMetrics().At(0)
	require.Equal(t, map[string]any{"service.name": "test2"}, countResourceMetrics.Resource().Attributes().AsRaw())

	countMetricSlice := countResourceMetrics.ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, countMetricSlice.Len())

	countDatapoints := countMetricSlice.At(0).Gauge().DataPoints()
	require.Equal(t, 1, countDatapoints.Len())

	countDP := countDatapoints.At(0)
	require.Equal(t, int64(1), countDP.IntValue())
	require.Equal(t, map[string]any{"dimension1": time.Duration(2 * time.Second).Milliseconds(), "dimension2": "test2"}, countDP.Attributes().AsRaw())
}

func TestConsumeTracesAttrsOnly(t *testing.T) {
	countMetricConsumer := &consumertest.MetricsSink{}
	nextTracesConsumer := &consumertest.TracesSink{}

	processorCfg := createDefaultConfig().(*Config)
	processorCfg.Interval = time.Millisecond * 100
	processorCfg.Attributes = map[string]string{
		"dimension1": `span_duration_ms`,
		"dimension2": `resource["service.name"]`,
	}

	processorFactory := NewFactory()
	processorSettings := processor.CreateSettings{TelemetrySettings: component.TelemetrySettings{Logger: zaptest.NewLogger(t)}}
	processor, err := processorFactory.CreateTracesProcessor(context.Background(), processorSettings, processorCfg, nextTracesConsumer)
	require.NoError(t, err)

	receiverFactory := routereceiver.NewFactory()
	receiver, err := receiverFactory.CreateMetricsReceiver(context.Background(), receiver.CreateSettings{}, receiverFactory.CreateDefaultConfig(), countMetricConsumer)
	require.NoError(t, err)

	err = processor.Start(context.Background(), nil)
	require.NoError(t, err)
	defer processor.Shutdown(context.Background())

	err = receiver.Start(context.Background(), nil)
	require.NoError(t, err)
	defer receiver.Shutdown(context.Background())

	now := time.Now()
	twoSecondsAgo := now.Add(-2 * time.Second)

	traces := ptrace.NewTraces()
	resourceLogs := traces.ResourceSpans().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(map[string]any{"service.name": "test2"})
	span := resourceLogs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().FromRaw(map[string]any{"dimension1": "test1", "dimension2": "test2"})
	span.Status().SetCode(ptrace.StatusCodeOk)
	span.SetKind(ptrace.SpanKindClient)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(twoSecondsAgo))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(now))

	processor.ConsumeTraces(context.Background(), traces)

	passedTraces := nextTracesConsumer.AllTraces()[0]
	require.Equal(t, traces, passedTraces)

	require.Eventually(t, func() bool {
		return len(countMetricConsumer.AllMetrics()) > 0
	}, 5*time.Second, 200*time.Millisecond)

	countMetrics := countMetricConsumer.AllMetrics()[0]
	require.Equal(t, 1, countMetrics.ResourceMetrics().Len())

	countResourceMetrics := countMetrics.ResourceMetrics().At(0)
	require.Equal(t, map[string]any{"service.name": "test2"}, countResourceMetrics.Resource().Attributes().AsRaw())

	countMetricSlice := countResourceMetrics.ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, countMetricSlice.Len())

	countDatapoints := countMetricSlice.At(0).Gauge().DataPoints()
	require.Equal(t, 1, countDatapoints.Len())

	countDP := countDatapoints.At(0)
	require.Equal(t, int64(1), countDP.IntValue())
	require.Equal(t, map[string]any{"dimension1": time.Duration(2 * time.Second).Milliseconds(), "dimension2": "test2"}, countDP.Attributes().AsRaw())
}

func TestConsumeTracesOTTL(t *testing.T) {
	countMetricConsumer := &consumertest.MetricsSink{}
	nextTracesConsumer := &consumertest.TracesSink{}

	ottlMatchExpr := `end_time_unix_nano - start_time_unix_nano > 1000000000 and resource.attributes["service.name"] == "test2"`

	processorCfg := createDefaultConfig().(*Config)
	processorCfg.Interval = time.Millisecond * 100
	processorCfg.OTTLMatch = &ottlMatchExpr
	processorCfg.OTTLAttributes = map[string]string{
		"dimension1": `end_time_unix_nano - start_time_unix_nano`,
		"dimension2": `resource.attributes["service.name"]`,
	}

	processorFactory := NewFactory()
	processorSettings := processor.CreateSettings{TelemetrySettings: component.TelemetrySettings{Logger: zaptest.NewLogger(t)}}
	processor, err := processorFactory.CreateTracesProcessor(context.Background(), processorSettings, processorCfg, nextTracesConsumer)
	require.NoError(t, err)

	receiverFactory := routereceiver.NewFactory()
	receiver, err := receiverFactory.CreateMetricsReceiver(context.Background(), receiver.CreateSettings{}, receiverFactory.CreateDefaultConfig(), countMetricConsumer)
	require.NoError(t, err)

	err = processor.Start(context.Background(), nil)
	require.NoError(t, err)
	defer processor.Shutdown(context.Background())

	err = receiver.Start(context.Background(), nil)
	require.NoError(t, err)
	defer receiver.Shutdown(context.Background())

	now := time.Now()
	twoSecondsAgo := now.Add(-2 * time.Second)

	traces := ptrace.NewTraces()
	resourceLogs := traces.ResourceSpans().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(map[string]any{"service.name": "test2"})
	span := resourceLogs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().FromRaw(map[string]any{"dimension1": "test1", "dimension2": "test2"})
	span.Status().SetCode(ptrace.StatusCodeOk)
	span.SetKind(ptrace.SpanKindClient)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(twoSecondsAgo))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(now))

	processor.ConsumeTraces(context.Background(), traces)

	passedTraces := nextTracesConsumer.AllTraces()[0]
	require.Equal(t, traces, passedTraces)

	require.Eventually(t, func() bool {
		return len(countMetricConsumer.AllMetrics()) > 0
	}, 5*time.Second, 200*time.Millisecond)

	countMetrics := countMetricConsumer.AllMetrics()[0]
	require.Equal(t, 1, countMetrics.ResourceMetrics().Len())

	countResourceMetrics := countMetrics.ResourceMetrics().At(0)
	require.Equal(t, map[string]any{"service.name": "test2"}, countResourceMetrics.Resource().Attributes().AsRaw())

	countMetricSlice := countResourceMetrics.ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, countMetricSlice.Len())

	countDatapoints := countMetricSlice.At(0).Gauge().DataPoints()
	require.Equal(t, 1, countDatapoints.Len())

	countDP := countDatapoints.At(0)
	require.Equal(t, int64(1), countDP.IntValue())
	require.Equal(t, map[string]any{"dimension1": time.Duration(2 * time.Second).Nanoseconds(), "dimension2": "test2"}, countDP.Attributes().AsRaw())
}

func TestConsumeTracesWithoutReceiver(t *testing.T) {
	logger := NewTestLogger()
	processorCfg := createDefaultConfig().(*Config)
	processorFactory := NewFactory()
	processorSettings := processor.CreateSettings{TelemetrySettings: component.TelemetrySettings{Logger: logger.Logger}}
	p, err := processorFactory.CreateTracesProcessor(context.Background(), processorSettings, processorCfg, &consumertest.TracesSink{})
	require.NoError(t, err)

	spanCountProcessor := p.(*spanCountProcessor)
	spanCountProcessor.counter.Add(map[string]any{"resource": "test1"}, map[string]any{"attribute": "test2"})
	spanCountProcessor.sendMetrics(context.Background())
	require.Contains(t, logger.buffer.String(), "Failed to send metrics")
	require.Contains(t, logger.buffer.String(), "route not defined")
}

type TestLogger struct {
	buffer *zaptest.Buffer
	*zap.Logger
}

func NewTestLogger() *TestLogger {
	buffer := &zaptest.Buffer{}
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, buffer, zapcore.DebugLevel)
	logger := zap.New(core)
	return &TestLogger{buffer: buffer, Logger: logger}
}
