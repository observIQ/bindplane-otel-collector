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

package logcountprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/receiver/routereceiver"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

var id = component.NewIDWithName(componentType, "test")

func TestProcessorCapabilities(t *testing.T) {
	p := &logCountProcessor{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, p.Capabilities())
}

func TestShutdownBeforeStart(t *testing.T) {
	logConsumer := &LogConsumer{logChan: make(chan plog.Logs, 1)}

	processorCfg := createDefaultConfig().(*Config)
	processorCfg.Interval = time.Millisecond * 100

	processorFactory := NewFactory()
	processorSettings := processor.Settings{ID: id, TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}
	processor, err := processorFactory.CreateLogs(context.Background(), processorSettings, processorCfg, logConsumer)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		processor.Shutdown(context.Background())
	})
}

func TestConsumeLogs(t *testing.T) {
	logConsumer := &LogConsumer{logChan: make(chan plog.Logs, 1)}
	metricConsumer := &MetricConsumer{metricChan: make(chan pmetric.Metrics, 1)}

	matchExpr := `body.message == "test1" and resource["service.name"] == "test2"`
	processorCfg := createDefaultConfig().(*Config)
	processorCfg.Interval = time.Millisecond * 100
	processorCfg.Match = &matchExpr
	processorCfg.Attributes = map[string]string{
		"dimension1": `body.message`,
		"dimension2": `resource["service.name"]`,
	}
	processorCfg.Route = "test"

	processorFactory := NewFactory()
	processorSettings := processor.Settings{ID: id, TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}
	processor, err := processorFactory.CreateLogs(context.Background(), processorSettings, processorCfg, logConsumer)
	require.NoError(t, err)

	receiverFactory := routereceiver.NewFactory()
	routeID := component.NewIDWithName(receiverFactory.Type(), "test")
	receiver, err := receiverFactory.CreateMetrics(context.Background(), receiver.Settings{ID: routeID}, receiverFactory.CreateDefaultConfig(), metricConsumer)
	require.NoError(t, err)

	err = processor.Start(context.Background(), nil)
	require.NoError(t, err)
	defer processor.Shutdown(context.Background())

	err = receiver.Start(context.Background(), nil)
	require.NoError(t, err)
	defer receiver.Shutdown(context.Background())

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(map[string]any{"service.name": "test2"})
	logRecord := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	logRecord.Body().SetEmptyMap().FromRaw(map[string]any{"message": "test1"})

	go func() {
		processor.ConsumeLogs(context.Background(), logs)
	}()

	consumedLogs := <-logConsumer.logChan
	require.Equal(t, logs, consumedLogs)

	consumedMetrics := <-metricConsumer.metricChan
	require.Equal(t, 1, consumedMetrics.ResourceMetrics().Len())

	resourceMetrics := consumedMetrics.ResourceMetrics().At(0)
	require.Equal(t, map[string]any{"service.name": "test2"}, resourceMetrics.Resource().Attributes().AsRaw())

	metricRecords := resourceMetrics.ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, metricRecords.Len())

	dataPoints := metricRecords.At(0).Gauge().DataPoints()
	require.Equal(t, 1, dataPoints.Len())

	metric := dataPoints.At(0)
	require.Equal(t, int64(1), metric.IntValue())
	require.Equal(t, map[string]any{"dimension1": "test1", "dimension2": "test2"}, metric.Attributes().AsRaw())
}

func TestConsumeLogsAttrsOnly(t *testing.T) {
	logConsumer := &LogConsumer{logChan: make(chan plog.Logs, 1)}
	metricConsumer := &MetricConsumer{metricChan: make(chan pmetric.Metrics, 1)}

	processorCfg := createDefaultConfig().(*Config)
	processorCfg.Interval = time.Millisecond * 100
	processorCfg.Attributes = map[string]string{
		"dimension1": `body.message`,
		"dimension2": `resource["service.name"]`,
	}
	processorCfg.Route = "test"
	processorFactory := NewFactory()
	processorSettings := processor.Settings{ID: id, TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}
	processor, err := processorFactory.CreateLogs(context.Background(), processorSettings, processorCfg, logConsumer)
	require.NoError(t, err)

	receiverFactory := routereceiver.NewFactory()
	routeID := component.NewIDWithName(receiverFactory.Type(), "test")
	receiver, err := receiverFactory.CreateMetrics(context.Background(), receiver.Settings{ID: routeID}, receiverFactory.CreateDefaultConfig(), metricConsumer)
	require.NoError(t, err)

	err = processor.Start(context.Background(), nil)
	require.NoError(t, err)
	defer processor.Shutdown(context.Background())

	err = receiver.Start(context.Background(), nil)
	require.NoError(t, err)
	defer receiver.Shutdown(context.Background())

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(map[string]any{"service.name": "test2"})
	logRecord := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	logRecord.Body().SetEmptyMap().FromRaw(map[string]any{"message": "test1"})

	go func() {
		processor.ConsumeLogs(context.Background(), logs)
	}()

	consumedLogs := <-logConsumer.logChan
	require.Equal(t, logs, consumedLogs)

	consumedMetrics := <-metricConsumer.metricChan
	require.Equal(t, 1, consumedMetrics.ResourceMetrics().Len())

	resourceMetrics := consumedMetrics.ResourceMetrics().At(0)
	require.Equal(t, map[string]any{"service.name": "test2"}, resourceMetrics.Resource().Attributes().AsRaw())

	metricRecords := resourceMetrics.ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, metricRecords.Len())

	dataPoints := metricRecords.At(0).Gauge().DataPoints()
	require.Equal(t, 1, dataPoints.Len())

	metric := dataPoints.At(0)
	require.Equal(t, int64(1), metric.IntValue())
	require.Equal(t, map[string]any{"dimension1": "test1", "dimension2": "test2"}, metric.Attributes().AsRaw())
}

func TestConsumeLogsOTTL(t *testing.T) {
	logConsumer := &LogConsumer{logChan: make(chan plog.Logs, 1)}
	metricConsumer := &MetricConsumer{metricChan: make(chan pmetric.Metrics, 1)}

	ottlMatchExpr := `body["message"] == "test1" and resource.attributes["service.name"] == "test2"`

	processorCfg := createDefaultConfig().(*Config)
	processorCfg.Interval = time.Millisecond * 100
	processorCfg.OTTLMatch = &ottlMatchExpr
	processorCfg.OTTLAttributes = map[string]string{
		"dimension1": `body["message"]`,
		"dimension2": `resource.attributes["service.name"]`,
	}
	processorCfg.Route = "test"
	processorFactory := NewFactory()
	processorSettings := processor.Settings{ID: id, TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}
	processor, err := processorFactory.CreateLogs(context.Background(), processorSettings, processorCfg, logConsumer)
	require.NoError(t, err)

	receiverFactory := routereceiver.NewFactory()
	routeID := component.NewIDWithName(receiverFactory.Type(), "test")
	receiver, err := receiverFactory.CreateMetrics(context.Background(), receiver.Settings{ID: routeID}, receiverFactory.CreateDefaultConfig(), metricConsumer)
	require.NoError(t, err)

	err = processor.Start(context.Background(), nil)
	require.NoError(t, err)
	defer processor.Shutdown(context.Background())

	err = receiver.Start(context.Background(), nil)
	require.NoError(t, err)
	defer receiver.Shutdown(context.Background())

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(map[string]any{"service.name": "test2"})
	logRecord := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	logRecord.Body().SetEmptyMap().FromRaw(map[string]any{"message": "test1"})

	go func() {
		processor.ConsumeLogs(context.Background(), logs)
	}()

	consumedLogs := <-logConsumer.logChan
	require.Equal(t, logs, consumedLogs)

	consumedMetrics := <-metricConsumer.metricChan
	require.Equal(t, 1, consumedMetrics.ResourceMetrics().Len())

	resourceMetrics := consumedMetrics.ResourceMetrics().At(0)
	require.Equal(t, map[string]any{"service.name": "test2"}, resourceMetrics.Resource().Attributes().AsRaw())

	metricRecords := resourceMetrics.ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, metricRecords.Len())

	dataPoints := metricRecords.At(0).Gauge().DataPoints()
	require.Equal(t, 1, dataPoints.Len())

	metric := dataPoints.At(0)
	require.Equal(t, int64(1), metric.IntValue())
	require.Equal(t, map[string]any{"dimension1": "test1", "dimension2": "test2"}, metric.Attributes().AsRaw())
}

func TestConsumeLogsWithoutReceiver(t *testing.T) {
	logger := NewTestLogger()
	processorCfg := createDefaultConfig().(*Config)
	processorFactory := NewFactory()
	processorSettings := processor.Settings{ID: id, TelemetrySettings: component.TelemetrySettings{Logger: logger.Logger}}
	p, err := processorFactory.CreateLogs(context.Background(), processorSettings, processorCfg, &LogConsumer{})
	require.NoError(t, err)

	logCountProcessor := p.(*logCountProcessor)
	logCountProcessor.counter.Add(map[string]any{"resource": "test1"}, map[string]any{"attribute": "test2"})
	logCountProcessor.sendMetrics(context.Background())
	require.Contains(t, logger.buffer.String(), "Failed to send metrics")
	require.Contains(t, logger.buffer.String(), "route not defined")
}

type LogConsumer struct {
	logChan chan plog.Logs
}

func (l *LogConsumer) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	l.logChan <- ld
	return nil
}

func (l *LogConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type MetricConsumer struct {
	metricChan chan pmetric.Metrics
}

func (m *MetricConsumer) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	m.metricChan <- md
	return nil
}

func (m *MetricConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
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
