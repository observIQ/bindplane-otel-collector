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

package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestGenerateLogPositions(t *testing.T) {
	testCases := []struct {
		desc     string
		logs     plog.Logs
		expected []logPosition
	}{
		{
			desc:     "Empty logs",
			logs:     plog.NewLogs(),
			expected: []logPosition{},
		},
		{
			desc: "Single log record",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				return logs
			}(),
			expected: []logPosition{
				{resourceIdx: 0, scopeIdx: 0, logIdx: 0},
			},
		},
		{
			desc: "Multiple resources, scopes, and logs",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				// Resource 0, Scope 0: 2 logs
				rl0 := logs.ResourceLogs().AppendEmpty()
				sl0 := rl0.ScopeLogs().AppendEmpty()
				sl0.LogRecords().AppendEmpty()
				sl0.LogRecords().AppendEmpty()
				// Resource 0, Scope 1: 1 log
				sl1 := rl0.ScopeLogs().AppendEmpty()
				sl1.LogRecords().AppendEmpty()
				// Resource 1, Scope 0: 1 log
				rl1 := logs.ResourceLogs().AppendEmpty()
				rl1.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				return logs
			}(),
			expected: []logPosition{
				{resourceIdx: 0, scopeIdx: 0, logIdx: 0},
				{resourceIdx: 0, scopeIdx: 0, logIdx: 1},
				{resourceIdx: 0, scopeIdx: 1, logIdx: 0},
				{resourceIdx: 1, scopeIdx: 0, logIdx: 0},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := generateLogPositions(tc.logs)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRandomSampleLogs(t *testing.T) {
	// Create logs with 100 log records
	createLogs := func(count int) plog.Logs {
		logs := plog.NewLogs()
		rl := logs.ResourceLogs().AppendEmpty()
		rl.Resource().Attributes().PutStr("resource", "test")
		sl := rl.ScopeLogs().AppendEmpty()
		sl.Scope().SetName("test-scope")
		for i := 0; i < count; i++ {
			lr := sl.LogRecords().AppendEmpty()
			lr.Body().SetStr("log message")
		}
		return logs
	}

	t.Run("100% retention returns original", func(t *testing.T) {
		logs := createLogs(100)
		positions := generateLogPositions(logs)
		result := randomSampleLogs(logs, positions, 100)
		// Should return original logs object
		assert.Equal(t, 100, result.LogRecordCount())
	})

	t.Run("0% retention becomes 1%", func(t *testing.T) {
		logs := createLogs(100)
		positions := generateLogPositions(logs)
		result := randomSampleLogs(logs, positions, 0)
		// 1% of 100 = 1
		assert.Equal(t, 1, result.LogRecordCount())
	})

	t.Run("50% retention", func(t *testing.T) {
		logs := createLogs(100)
		positions := generateLogPositions(logs)
		result := randomSampleLogs(logs, positions, 50)
		// 50% of 100 = 50
		assert.Equal(t, 50, result.LogRecordCount())
	})

	t.Run("25% retention", func(t *testing.T) {
		logs := createLogs(100)
		positions := generateLogPositions(logs)
		result := randomSampleLogs(logs, positions, 25)
		// 25% of 100 = 25
		assert.Equal(t, 25, result.LogRecordCount())
	})

	t.Run("Preserves resource and scope attributes", func(t *testing.T) {
		logs := createLogs(10)
		positions := generateLogPositions(logs)
		result := randomSampleLogs(logs, positions, 50)

		require.Greater(t, result.ResourceLogs().Len(), 0)
		resourceVal, ok := result.ResourceLogs().At(0).Resource().Attributes().Get("resource")
		require.True(t, ok)
		assert.Equal(t, "test", resourceVal.Str())

		require.Greater(t, result.ResourceLogs().At(0).ScopeLogs().Len(), 0)
		assert.Equal(t, "test-scope", result.ResourceLogs().At(0).ScopeLogs().At(0).Scope().Name())
	})

	t.Run("Multiple resources and scopes", func(t *testing.T) {
		logs := plog.NewLogs()
		// Resource 0 with 50 logs
		rl0 := logs.ResourceLogs().AppendEmpty()
		rl0.Resource().Attributes().PutStr("resource", "r0")
		sl0 := rl0.ScopeLogs().AppendEmpty()
		for i := 0; i < 50; i++ {
			sl0.LogRecords().AppendEmpty()
		}
		// Resource 1 with 50 logs
		rl1 := logs.ResourceLogs().AppendEmpty()
		rl1.Resource().Attributes().PutStr("resource", "r1")
		sl1 := rl1.ScopeLogs().AppendEmpty()
		for i := 0; i < 50; i++ {
			sl1.LogRecords().AppendEmpty()
		}

		positions := generateLogPositions(logs)
		result := randomSampleLogs(logs, positions, 50)
		// 50% of 100 = 50
		assert.Equal(t, 50, result.LogRecordCount())
	})
}

func TestGenerateDataPointPositions(t *testing.T) {
	testCases := []struct {
		desc     string
		metrics  pmetric.Metrics
		expected int // number of positions expected
	}{
		{
			desc:     "Empty metrics",
			metrics:  pmetric.NewMetrics(),
			expected: 0,
		},
		{
			desc: "Single gauge metric with one datapoint",
			metrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				m := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				m.SetEmptyGauge()
				m.Gauge().DataPoints().AppendEmpty()
				return metrics
			}(),
			expected: 1,
		},
		{
			desc: "Multiple metric types",
			metrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				sm := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()

				// Gauge with 2 data points
				gauge := sm.Metrics().AppendEmpty()
				gauge.SetEmptyGauge()
				gauge.Gauge().DataPoints().AppendEmpty()
				gauge.Gauge().DataPoints().AppendEmpty()

				// Sum with 1 data point
				sum := sm.Metrics().AppendEmpty()
				sum.SetEmptySum()
				sum.Sum().DataPoints().AppendEmpty()

				// Histogram with 1 data point
				histogram := sm.Metrics().AppendEmpty()
				histogram.SetEmptyHistogram()
				histogram.Histogram().DataPoints().AppendEmpty()

				return metrics
			}(),
			expected: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := generateDataPointPositions(tc.metrics)
			assert.Equal(t, tc.expected, len(result))
		})
	}
}

func TestGetDataPointCount(t *testing.T) {
	testCases := []struct {
		desc     string
		setup    func() pmetric.Metric
		expected int
	}{
		{
			desc: "Gauge",
			setup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyGauge()
				m.Gauge().DataPoints().AppendEmpty()
				m.Gauge().DataPoints().AppendEmpty()
				return m
			},
			expected: 2,
		},
		{
			desc: "Sum",
			setup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptySum()
				m.Sum().DataPoints().AppendEmpty()
				return m
			},
			expected: 1,
		},
		{
			desc: "Histogram",
			setup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyHistogram()
				m.Histogram().DataPoints().AppendEmpty()
				m.Histogram().DataPoints().AppendEmpty()
				m.Histogram().DataPoints().AppendEmpty()
				return m
			},
			expected: 3,
		},
		{
			desc: "ExponentialHistogram",
			setup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyExponentialHistogram()
				m.ExponentialHistogram().DataPoints().AppendEmpty()
				return m
			},
			expected: 1,
		},
		{
			desc: "Summary",
			setup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptySummary()
				m.Summary().DataPoints().AppendEmpty()
				m.Summary().DataPoints().AppendEmpty()
				return m
			},
			expected: 2,
		},
		{
			desc: "Empty/Unknown type",
			setup: func() pmetric.Metric {
				return pmetric.NewMetric()
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			m := tc.setup()
			assert.Equal(t, tc.expected, getDataPointCount(m))
		})
	}
}

func TestRandomSampleMetrics(t *testing.T) {
	// Create metrics with specified number of data points
	createMetrics := func(count int) pmetric.Metrics {
		metrics := pmetric.NewMetrics()
		rm := metrics.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("resource", "test")
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test-scope")
		m := sm.Metrics().AppendEmpty()
		m.SetName("test-metric")
		m.SetDescription("test description")
		m.SetUnit("1")
		m.SetEmptyGauge()
		for i := 0; i < count; i++ {
			dp := m.Gauge().DataPoints().AppendEmpty()
			dp.SetIntValue(int64(i))
		}
		return metrics
	}

	t.Run("100% retention returns original", func(t *testing.T) {
		metrics := createMetrics(100)
		positions := generateDataPointPositions(metrics)
		result := randomSampleMetrics(metrics, positions, 100)
		assert.Equal(t, 100, result.DataPointCount())
	})

	t.Run("0% retention becomes 1%", func(t *testing.T) {
		metrics := createMetrics(100)
		positions := generateDataPointPositions(metrics)
		result := randomSampleMetrics(metrics, positions, 0)
		// 1% of 100 = 1
		assert.Equal(t, 1, result.DataPointCount())
	})

	t.Run("50% retention", func(t *testing.T) {
		metrics := createMetrics(100)
		positions := generateDataPointPositions(metrics)
		result := randomSampleMetrics(metrics, positions, 50)
		// 50% of 100 = 50
		assert.Equal(t, 50, result.DataPointCount())
	})

	t.Run("Preserves metric metadata", func(t *testing.T) {
		metrics := createMetrics(10)
		positions := generateDataPointPositions(metrics)
		result := randomSampleMetrics(metrics, positions, 50)

		require.Greater(t, result.ResourceMetrics().Len(), 0)
		require.Greater(t, result.ResourceMetrics().At(0).ScopeMetrics().Len(), 0)
		require.Greater(t, result.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len(), 0)

		m := result.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
		assert.Equal(t, "test-metric", m.Name())
		assert.Equal(t, "test description", m.Description())
		assert.Equal(t, "1", m.Unit())
	})

	t.Run("Different metric types", func(t *testing.T) {
		metrics := pmetric.NewMetrics()
		sm := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()

		// Gauge with 25 data points
		gauge := sm.Metrics().AppendEmpty()
		gauge.SetName("gauge")
		gauge.SetEmptyGauge()
		for i := 0; i < 25; i++ {
			gauge.Gauge().DataPoints().AppendEmpty()
		}

		// Sum with 25 data points
		sum := sm.Metrics().AppendEmpty()
		sum.SetName("sum")
		sum.SetEmptySum()
		sum.Sum().SetIsMonotonic(true)
		sum.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		for i := 0; i < 25; i++ {
			sum.Sum().DataPoints().AppendEmpty()
		}

		// Histogram with 25 data points
		histogram := sm.Metrics().AppendEmpty()
		histogram.SetName("histogram")
		histogram.SetEmptyHistogram()
		histogram.Histogram().SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
		for i := 0; i < 25; i++ {
			histogram.Histogram().DataPoints().AppendEmpty()
		}

		// Summary with 25 data points
		summary := sm.Metrics().AppendEmpty()
		summary.SetName("summary")
		summary.SetEmptySummary()
		for i := 0; i < 25; i++ {
			summary.Summary().DataPoints().AppendEmpty()
		}

		positions := generateDataPointPositions(metrics)
		assert.Equal(t, 100, len(positions))

		result := randomSampleMetrics(metrics, positions, 50)
		// 50% of 100 = 50
		assert.Equal(t, 50, result.DataPointCount())
	})

	t.Run("Sum metric preserves aggregation properties", func(t *testing.T) {
		metrics := pmetric.NewMetrics()
		sm := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
		sum := sm.Metrics().AppendEmpty()
		sum.SetName("sum")
		sum.SetEmptySum()
		sum.Sum().SetIsMonotonic(true)
		sum.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		for i := 0; i < 10; i++ {
			sum.Sum().DataPoints().AppendEmpty()
		}

		positions := generateDataPointPositions(metrics)
		result := randomSampleMetrics(metrics, positions, 50)

		require.Greater(t, result.ResourceMetrics().Len(), 0)
		require.Greater(t, result.ResourceMetrics().At(0).ScopeMetrics().Len(), 0)
		require.Greater(t, result.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len(), 0)

		m := result.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
		assert.Equal(t, pmetric.MetricTypeSum, m.Type())
		assert.True(t, m.Sum().IsMonotonic())
		assert.Equal(t, pmetric.AggregationTemporalityCumulative, m.Sum().AggregationTemporality())
	})
}

func TestGenerateSpanPositions(t *testing.T) {
	testCases := []struct {
		desc     string
		traces   ptrace.Traces
		expected []spanPosition
	}{
		{
			desc:     "Empty traces",
			traces:   ptrace.NewTraces(),
			expected: []spanPosition{},
		},
		{
			desc: "Single span",
			traces: func() ptrace.Traces {
				traces := ptrace.NewTraces()
				traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				return traces
			}(),
			expected: []spanPosition{
				{resourceIdx: 0, scopeIdx: 0, spanIdx: 0},
			},
		},
		{
			desc: "Multiple resources, scopes, and spans",
			traces: func() ptrace.Traces {
				traces := ptrace.NewTraces()
				// Resource 0, Scope 0: 2 spans
				rs0 := traces.ResourceSpans().AppendEmpty()
				ss0 := rs0.ScopeSpans().AppendEmpty()
				ss0.Spans().AppendEmpty()
				ss0.Spans().AppendEmpty()
				// Resource 0, Scope 1: 1 span
				ss1 := rs0.ScopeSpans().AppendEmpty()
				ss1.Spans().AppendEmpty()
				// Resource 1, Scope 0: 1 span
				rs1 := traces.ResourceSpans().AppendEmpty()
				rs1.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				return traces
			}(),
			expected: []spanPosition{
				{resourceIdx: 0, scopeIdx: 0, spanIdx: 0},
				{resourceIdx: 0, scopeIdx: 0, spanIdx: 1},
				{resourceIdx: 0, scopeIdx: 1, spanIdx: 0},
				{resourceIdx: 1, scopeIdx: 0, spanIdx: 0},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := generateSpanPositions(tc.traces)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRandomSampleTraces(t *testing.T) {
	// Create traces with specified number of spans
	createTraces := func(count int) ptrace.Traces {
		traces := ptrace.NewTraces()
		rs := traces.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("resource", "test")
		ss := rs.ScopeSpans().AppendEmpty()
		ss.Scope().SetName("test-scope")
		for i := 0; i < count; i++ {
			span := ss.Spans().AppendEmpty()
			span.SetName("test-span")
		}
		return traces
	}

	t.Run("100% retention returns original", func(t *testing.T) {
		traces := createTraces(100)
		positions := generateSpanPositions(traces)
		result := randomSampleTraces(traces, positions, 100)
		assert.Equal(t, 100, result.SpanCount())
	})

	t.Run("0% retention becomes 1%", func(t *testing.T) {
		traces := createTraces(100)
		positions := generateSpanPositions(traces)
		result := randomSampleTraces(traces, positions, 0)
		// 1% of 100 = 1
		assert.Equal(t, 1, result.SpanCount())
	})

	t.Run("50% retention", func(t *testing.T) {
		traces := createTraces(100)
		positions := generateSpanPositions(traces)
		result := randomSampleTraces(traces, positions, 50)
		// 50% of 100 = 50
		assert.Equal(t, 50, result.SpanCount())
	})

	t.Run("25% retention", func(t *testing.T) {
		traces := createTraces(100)
		positions := generateSpanPositions(traces)
		result := randomSampleTraces(traces, positions, 25)
		// 25% of 100 = 25
		assert.Equal(t, 25, result.SpanCount())
	})

	t.Run("Preserves resource and scope attributes", func(t *testing.T) {
		traces := createTraces(10)
		positions := generateSpanPositions(traces)
		result := randomSampleTraces(traces, positions, 50)

		require.Greater(t, result.ResourceSpans().Len(), 0)
		resourceVal, ok := result.ResourceSpans().At(0).Resource().Attributes().Get("resource")
		require.True(t, ok)
		assert.Equal(t, "test", resourceVal.Str())

		require.Greater(t, result.ResourceSpans().At(0).ScopeSpans().Len(), 0)
		assert.Equal(t, "test-scope", result.ResourceSpans().At(0).ScopeSpans().At(0).Scope().Name())
	})

	t.Run("Multiple resources and scopes", func(t *testing.T) {
		traces := ptrace.NewTraces()
		// Resource 0 with 50 spans
		rs0 := traces.ResourceSpans().AppendEmpty()
		rs0.Resource().Attributes().PutStr("resource", "r0")
		ss0 := rs0.ScopeSpans().AppendEmpty()
		for i := 0; i < 50; i++ {
			ss0.Spans().AppendEmpty()
		}
		// Resource 1 with 50 spans
		rs1 := traces.ResourceSpans().AppendEmpty()
		rs1.Resource().Attributes().PutStr("resource", "r1")
		ss1 := rs1.ScopeSpans().AppendEmpty()
		for i := 0; i < 50; i++ {
			ss1.Spans().AppendEmpty()
		}

		positions := generateSpanPositions(traces)
		result := randomSampleTraces(traces, positions, 50)
		// 50% of 100 = 50
		assert.Equal(t, 50, result.SpanCount())
	})
}

func TestInitMetricDataPoints(t *testing.T) {
	testCases := []struct {
		desc      string
		srcSetup  func() pmetric.Metric
		checkDest func(t *testing.T, dst pmetric.Metric)
	}{
		{
			desc: "Gauge",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyGauge()
				return m
			},
			checkDest: func(t *testing.T, dst pmetric.Metric) {
				assert.Equal(t, pmetric.MetricTypeGauge, dst.Type())
			},
		},
		{
			desc: "Sum with properties",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptySum()
				m.Sum().SetIsMonotonic(true)
				m.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				return m
			},
			checkDest: func(t *testing.T, dst pmetric.Metric) {
				assert.Equal(t, pmetric.MetricTypeSum, dst.Type())
				assert.True(t, dst.Sum().IsMonotonic())
				assert.Equal(t, pmetric.AggregationTemporalityCumulative, dst.Sum().AggregationTemporality())
			},
		},
		{
			desc: "Histogram",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyHistogram()
				m.Histogram().SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
				return m
			},
			checkDest: func(t *testing.T, dst pmetric.Metric) {
				assert.Equal(t, pmetric.MetricTypeHistogram, dst.Type())
				assert.Equal(t, pmetric.AggregationTemporalityDelta, dst.Histogram().AggregationTemporality())
			},
		},
		{
			desc: "ExponentialHistogram",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyExponentialHistogram()
				m.ExponentialHistogram().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				return m
			},
			checkDest: func(t *testing.T, dst pmetric.Metric) {
				assert.Equal(t, pmetric.MetricTypeExponentialHistogram, dst.Type())
				assert.Equal(t, pmetric.AggregationTemporalityCumulative, dst.ExponentialHistogram().AggregationTemporality())
			},
		},
		{
			desc: "Summary",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptySummary()
				return m
			},
			checkDest: func(t *testing.T, dst pmetric.Metric) {
				assert.Equal(t, pmetric.MetricTypeSummary, dst.Type())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			src := tc.srcSetup()
			dst := pmetric.NewMetric()
			initMetricDataPoints(src, dst)
			tc.checkDest(t, dst)
		})
	}
}

func TestCopyDataPoint(t *testing.T) {
	testCases := []struct {
		desc     string
		srcSetup func() pmetric.Metric
		idx      int
		validate func(t *testing.T, dst pmetric.Metric)
	}{
		{
			desc: "Gauge data point",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyGauge()
				dp := m.Gauge().DataPoints().AppendEmpty()
				dp.SetIntValue(42)
				dp.Attributes().PutStr("key", "value")
				return m
			},
			idx: 0,
			validate: func(t *testing.T, dst pmetric.Metric) {
				require.Equal(t, 1, dst.Gauge().DataPoints().Len())
				dp := dst.Gauge().DataPoints().At(0)
				assert.Equal(t, int64(42), dp.IntValue())
				val, ok := dp.Attributes().Get("key")
				require.True(t, ok)
				assert.Equal(t, "value", val.Str())
			},
		},
		{
			desc: "Sum data point",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptySum()
				dp := m.Sum().DataPoints().AppendEmpty()
				dp.SetDoubleValue(3.14)
				return m
			},
			idx: 0,
			validate: func(t *testing.T, dst pmetric.Metric) {
				require.Equal(t, 1, dst.Sum().DataPoints().Len())
				assert.Equal(t, 3.14, dst.Sum().DataPoints().At(0).DoubleValue())
			},
		},
		{
			desc: "Histogram data point",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyHistogram()
				dp := m.Histogram().DataPoints().AppendEmpty()
				dp.SetCount(100)
				dp.SetSum(500.0)
				return m
			},
			idx: 0,
			validate: func(t *testing.T, dst pmetric.Metric) {
				require.Equal(t, 1, dst.Histogram().DataPoints().Len())
				dp := dst.Histogram().DataPoints().At(0)
				assert.Equal(t, uint64(100), dp.Count())
				assert.Equal(t, 500.0, dp.Sum())
			},
		},
		{
			desc: "ExponentialHistogram data point",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyExponentialHistogram()
				dp := m.ExponentialHistogram().DataPoints().AppendEmpty()
				dp.SetCount(50)
				dp.SetScale(2)
				return m
			},
			idx: 0,
			validate: func(t *testing.T, dst pmetric.Metric) {
				require.Equal(t, 1, dst.ExponentialHistogram().DataPoints().Len())
				dp := dst.ExponentialHistogram().DataPoints().At(0)
				assert.Equal(t, uint64(50), dp.Count())
				assert.Equal(t, int32(2), dp.Scale())
			},
		},
		{
			desc: "Summary data point",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptySummary()
				dp := m.Summary().DataPoints().AppendEmpty()
				dp.SetCount(200)
				dp.SetSum(1000.0)
				return m
			},
			idx: 0,
			validate: func(t *testing.T, dst pmetric.Metric) {
				require.Equal(t, 1, dst.Summary().DataPoints().Len())
				dp := dst.Summary().DataPoints().At(0)
				assert.Equal(t, uint64(200), dp.Count())
				assert.Equal(t, 1000.0, dp.Sum())
			},
		},
		{
			desc: "Copy specific index",
			srcSetup: func() pmetric.Metric {
				m := pmetric.NewMetric()
				m.SetEmptyGauge()
				m.Gauge().DataPoints().AppendEmpty().SetIntValue(1)
				m.Gauge().DataPoints().AppendEmpty().SetIntValue(2)
				m.Gauge().DataPoints().AppendEmpty().SetIntValue(3)
				return m
			},
			idx: 1,
			validate: func(t *testing.T, dst pmetric.Metric) {
				require.Equal(t, 1, dst.Gauge().DataPoints().Len())
				assert.Equal(t, int64(2), dst.Gauge().DataPoints().At(0).IntValue())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			src := tc.srcSetup()
			dst := pmetric.NewMetric()
			initMetricDataPoints(src, dst)
			copyDataPoint(src, dst, tc.idx)
			tc.validate(t, dst)
		})
	}
}
