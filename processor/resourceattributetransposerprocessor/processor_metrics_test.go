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
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestMetricsProcessorStart(t *testing.T) {
	p := newMetricsProcessor(
		zap.NewNop(),
		consumertest.NewNop(),
		createDefaultConfig().(*Config),
	)

	err := p.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
}

func TestMetricsProcessorShutdown(t *testing.T) {
	p := newMetricsProcessor(
		zap.NewNop(),
		consumertest.NewNop(),
		createDefaultConfig().(*Config),
	)

	err := p.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestMetricsProcessorCapabilities(t *testing.T) {
	p := newMetricsProcessor(
		zap.NewNop(),
		consumertest.NewNop(),
		createDefaultConfig().(*Config),
	)
	capabilities := p.Capabilities()
	require.True(t, capabilities.MutatesData)
}

// TestConsumeMetricsNoop test that the default config is essentially a noop
func TestConsumeMetricsNoop(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value"))
	attrs.Insert("resourceattrib2", pcommon.NewValueBool(false))
	attrs.Insert("resourceattrib3", pcommon.NewValueBytes([]byte("some bytes")))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
	}).Return(nil)

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		createDefaultConfig().(*Config),
	)

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Empty(t, getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsMoveExistingAttribs(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value"))
	attrs.Insert("resourceattrib2", pcommon.NewValueBool(false))
	attrs.Insert("resourceattrib3", pcommon.NewValueBytes([]byte("some bytes")))
	attrs.Insert("resourceattrib4", pcommon.NewValueDouble(2.0))
	attrs.Insert("resourceattrib5", pcommon.NewValueInt(100))
	attrs.Insert("resourceattrib6", pcommon.NewValueEmpty())

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
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

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeGauge)
	dp := metric.Gauge().DataPoints()
	dp.AppendEmpty().SetDoubleVal(3.0)

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"resourceattrib1": "value",
		"resourceattrib2": false,
		"resourceattrib3": []byte("some bytes"),
		"resourceattrib4": float64(2.0),
		"resourceattrib5": int64(100),
		"resourceattrib6": nil,
	}, getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsMoveToMultipleMetrics(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1",
		},
	}

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)
	metricsSlice := getMetricSlice(metrics)
	metric1 := metricsSlice.At(0)
	metric1.SetDataType(pmetric.MetricDataTypeGauge)
	dp1 := metric1.Gauge().DataPoints()
	dp1.AppendEmpty().SetDoubleVal(3.0)

	metric2 := metricsSlice.AppendEmpty()
	metric2.SetDataType(pmetric.MetricDataTypeGauge)
	dp2 := metric2.Gauge().DataPoints()
	dp2.AppendEmpty().SetDoubleVal(3.0)

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"resourceattrib1": "value",
	}, getMetricAttrsFromMetric(getMetricSlice(metricsOut).At(0)))

	require.Equal(t, map[string]interface{}{
		"resourceattrib1": "value",
	}, getMetricAttrsFromMetric(getMetricSlice(metricsOut).At(1)))
}

func TestConsumeMetricsMixedExistence(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))
	attrs.Insert("resourceattrib2", pcommon.NewValueString("value2"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1out",
		},
	}

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeGauge)
	dp := metric.Gauge().DataPoints()
	dp.AppendEmpty().SetDoubleVal(3.0)

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"resourceattrib1out": "value1",
	}, getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsSum(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1out",
		},
	}

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeSum)
	dp := metric.Sum().DataPoints()
	dp.AppendEmpty().SetDoubleVal(3.0)

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"resourceattrib1out": "value1",
	}, getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsHistogram(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1out",
		},
	}

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeHistogram)
	dp := metric.Histogram().DataPoints()
	dp.AppendEmpty()

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"resourceattrib1out": "value1",
	}, getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsSummary(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1out",
		},
	}

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeSummary)
	dp := metric.Summary().DataPoints()
	dp.AppendEmpty()

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"resourceattrib1out": "value1",
	}, getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsNone(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
	}).Return(nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Operations = []CopyResourceConfig{
		{
			From: "resourceattrib1",
			To:   "resourceattrib1out",
		},
	}

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeNone)

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}(nil), getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsDoesNotOverwrite(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))
	attrs.Insert("resourceattrib2", pcommon.NewValueString("value2"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
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

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeGauge)
	dp := metric.Gauge().DataPoints()
	dp.AppendEmpty().SetDoubleVal(3.0)

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"out": "value1",
	}, getMetricAttrsFromMetrics(metricsOut))
}

func TestConsumeMetricsDoesNotOverwrite2(t *testing.T) {
	ctx := context.Background()
	metrics := createMetrics()

	attrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Insert("resourceattrib1", pcommon.NewValueString("value1"))
	attrs.Insert("resourceattrib2", pcommon.NewValueString("value2"))

	var metricsOut pmetric.Metrics

	consumer := &mockMetricsConsumer{}
	consumer.On("ConsumeMetrics", ctx, metrics).Run(func(args mock.Arguments) {
		metricsOut = args[1].(pmetric.Metrics)
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

	p := newMetricsProcessor(
		zap.NewNop(),
		consumer,
		cfg,
	)

	metric := getMetric(metrics)
	metric.SetDataType(pmetric.MetricDataTypeGauge)
	dps := metric.Gauge().DataPoints()
	dp := dps.AppendEmpty()
	dp.SetDoubleVal(3.0)
	dp.Attributes().InsertString("out", "originalvalue")

	err := p.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{
		"out": "originalvalue",
	}, getMetricAttrsFromMetrics(metricsOut))
}

type mockMetricsConsumer struct {
	mock.Mock
}

func (m *mockMetricsConsumer) Start(ctx context.Context, host component.Host) error {
	args := m.Called(ctx, host)
	return args.Error(0)
}

func (m *mockMetricsConsumer) Capabilities() consumer.Capabilities {
	args := m.Called()
	return args.Get(0).(consumer.Capabilities)
}

func (m *mockMetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	args := m.Called(ctx, md)
	return args.Error(0)
}

func (m *mockMetricsConsumer) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func getMetricSlice(m pmetric.Metrics) pmetric.MetricSlice {
	return m.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
}

func getMetric(m pmetric.Metrics) pmetric.Metric {
	return getMetricSlice(m).At(0)
}

func createMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	return metrics
}

func getMetricAttrsFromMetrics(m pmetric.Metrics) map[string]interface{} {
	return getMetricAttrsFromMetric(getMetric(m))
}

func getMetricAttrsFromMetric(m pmetric.Metric) map[string]interface{} {
	switch m.DataType() {
	case pmetric.MetricDataTypeGauge:
		return m.Gauge().DataPoints().At(0).Attributes().AsRaw()
	case pmetric.MetricDataTypeSum:
		return m.Sum().DataPoints().At(0).Attributes().AsRaw()
	case pmetric.MetricDataTypeHistogram:
		return m.Histogram().DataPoints().At(0).Attributes().AsRaw()
	case pmetric.MetricDataTypeSummary:
		return m.Summary().DataPoints().At(0).Attributes().AsRaw()
	}
	return nil
}
