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

package removeemptyvaluesprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap/zaptest"
)

func TestProcessMetrics(t *testing.T) {
	t.Run("Removes attributes", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: resourceField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputMetrics := testMetrics()

		outputLogs, err := p.processMetrics(context.Background(), inputMetrics)
		require.NoError(t, err)

		outResourceMetrics := outputLogs.ResourceMetrics().At(0)
		outMetricsSlice := outResourceMetrics.ScopeMetrics().At(0).Metrics()
		require.Equal(t, rawResourceAttributes, outResourceMetrics.Resource().Attributes().AsRaw())
		requireMetricsAttrsEqual(t, map[string]any{
			"attr_key": "attr_value",
		}, outMetricsSlice)
	})

	t.Run("Removes attributes, ignore excluded", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
					key:   "empty.key",
				},
				{
					field: resourceField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputMetrics := testMetrics()

		outputLogs, err := p.processMetrics(context.Background(), inputMetrics)
		require.NoError(t, err)

		outResourceMetrics := outputLogs.ResourceMetrics().At(0)
		outMetricsSlice := outResourceMetrics.ScopeMetrics().At(0).Metrics()
		require.Equal(t, rawResourceAttributes, outResourceMetrics.Resource().Attributes().AsRaw())
		requireMetricsAttrsEqual(t, map[string]any{
			"attr_key":  "attr_value",
			"empty.key": nil,
		}, outMetricsSlice)
	})

	t.Run("Removes resource attributes", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputMetrics := testMetrics()

		outputLogs, err := p.processMetrics(context.Background(), inputMetrics)
		require.NoError(t, err)

		outResourceMetrics := outputLogs.ResourceMetrics().At(0)
		outMetricsSlice := outResourceMetrics.ScopeMetrics().At(0).Metrics()
		require.Equal(t, map[string]any{
			"resource_key": "resource_value",
		}, outResourceMetrics.Resource().Attributes().AsRaw())
		requireMetricsAttrsEqual(t, rawAttributes, outMetricsSlice)
	})

	t.Run("Removes resource attributes, ignores excluded", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: resourceField,
					key:   "nested.map.map.some.key",
				},
				{
					field: attributesField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputMetrics := testMetrics()

		outputLogs, err := p.processMetrics(context.Background(), inputMetrics)
		require.NoError(t, err)

		outResourceMetrics := outputLogs.ResourceMetrics().At(0)
		outMetricsSlice := outResourceMetrics.ScopeMetrics().At(0).Metrics()
		require.Equal(t, map[string]any{
			"resource_key": "resource_value",
			"nested.map": map[string]any{
				"map": map[string]any{
					"some.key": "-",
				},
			},
		}, outResourceMetrics.Resource().Attributes().AsRaw())
		requireMetricsAttrsEqual(t, rawAttributes, outMetricsSlice)
	})
}

func TestProcessTraces(t *testing.T) {
	t.Run("Removes attributes", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: resourceField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputTraces := testTraces()

		outputTraces, err := p.processTraces(context.Background(), inputTraces)
		require.NoError(t, err)

		outResourceSpans := outputTraces.ResourceSpans().At(0)
		span := outResourceSpans.ScopeSpans().At(0).Spans().At(0)

		require.Equal(t, rawResourceAttributes, outResourceSpans.Resource().Attributes().AsRaw())
		require.Equal(t, map[string]any{
			"attr_key": "attr_value",
		}, span.Attributes().AsRaw())
	})

	t.Run("Removes attributes, ignores excluded", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
					key:   "empty.key",
				},
				{
					field: resourceField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputTraces := testTraces()

		outputTraces, err := p.processTraces(context.Background(), inputTraces)
		require.NoError(t, err)

		outResourceSpans := outputTraces.ResourceSpans().At(0)
		span := outResourceSpans.ScopeSpans().At(0).Spans().At(0)

		require.Equal(t, rawResourceAttributes, outResourceSpans.Resource().Attributes().AsRaw())
		require.Equal(t, map[string]any{
			"attr_key":  "attr_value",
			"empty.key": nil,
		}, span.Attributes().AsRaw())
	})

	t.Run("Removes resource attributes", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputTraces := testTraces()

		outputTraces, err := p.processTraces(context.Background(), inputTraces)
		require.NoError(t, err)

		outResourceSpans := outputTraces.ResourceSpans().At(0)
		span := outResourceSpans.ScopeSpans().At(0).Spans().At(0)

		require.Equal(t, map[string]any{
			"resource_key": "resource_value",
		}, outResourceSpans.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, span.Attributes().AsRaw())
	})

	t.Run("Removes resource attributes, ignores excluded", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: resourceField,
					key:   "nested.map.map.some.key",
				},
				{
					field: attributesField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputTraces := testTraces()

		outputTraces, err := p.processTraces(context.Background(), inputTraces)
		require.NoError(t, err)

		outResourceSpans := outputTraces.ResourceSpans().At(0)
		span := outResourceSpans.ScopeSpans().At(0).Spans().At(0)

		require.Equal(t, map[string]any{
			"resource_key": "resource_value",
			"nested.map": map[string]any{
				"map": map[string]any{
					"some.key": "-",
				},
			},
		}, outResourceSpans.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, span.Attributes().AsRaw())
	})
}

func TestProcessLogs(t *testing.T) {
	t.Run("Removes attributes", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: resourceField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputLog := testLog()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, rawResourceAttributes, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, map[string]any{
			"attr_key": "attr_value",
		}, outLogRecord.Attributes().AsRaw())
		require.Equal(t, rawBody, outLogRecord.Body().Map().AsRaw())
	})

	t.Run("Removes attributes, skips excluded", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
					key:   "empty.key",
				},
				{
					field: resourceField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputLog := testLog()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, rawResourceAttributes, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, map[string]any{
			"attr_key":  "attr_value",
			"empty.key": nil,
		}, outLogRecord.Attributes().AsRaw())
		require.Equal(t, rawBody, outLogRecord.Body().Map().AsRaw())
	})

	t.Run("Removes resource attributes", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputLog := testLog()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, map[string]any{
			"resource_key": "resource_value",
		}, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, outLogRecord.Attributes().AsRaw())
		require.Equal(t, rawBody, outLogRecord.Body().Map().AsRaw())
	})

	t.Run("Removes resource attributes, ignores excluded", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: resourceField,
					key:   "nested.map.map.some.key",
				},
				{
					field: attributesField,
				},
				{
					field: bodyField,
				},
			},
		})

		inputLog := testLog()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, map[string]any{
			"resource_key": "resource_value",
			"nested.map": map[string]any{
				"map": map[string]any{
					"some.key": "-",
				},
			},
		}, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, outLogRecord.Attributes().AsRaw())
		require.Equal(t, rawBody, outLogRecord.Body().Map().AsRaw())
	})

	t.Run("Removes body keys", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
				},
				{
					field: resourceField,
				},
			},
		})

		inputLog := testLog()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, rawResourceAttributes, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, outLogRecord.Attributes().AsRaw())
		require.Equal(t, map[string]any{
			"body_key": "body_value",
		}, outLogRecord.Body().Map().AsRaw())
	})

	t.Run("Removes body keys, ignores excluded", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: bodyField,
					key:   "empty.key",
				},
				{
					field: attributesField,
				},
				{
					field: resourceField,
				},
			},
		})

		inputLog := testLog()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, rawResourceAttributes, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, outLogRecord.Attributes().AsRaw())
		require.Equal(t, map[string]any{
			"body_key":  "body_value",
			"empty.key": nil,
		}, outLogRecord.Body().Map().AsRaw())
	})

	t.Run("Removes empty slice body", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
				},
				{
					field: resourceField,
				},
			},
		})

		inputLog := testLogEmptySliceBody()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, rawResourceAttributes, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, outLogRecord.Attributes().AsRaw())
		require.Equal(t, pcommon.ValueTypeEmpty, outLogRecord.Body().Type())
	})

	t.Run("Removes string body", func(t *testing.T) {
		p := newEmptyValueProcessor(zaptest.NewLogger(t), Config{
			RemoveNulls:      true,
			RemoveEmptyLists: true,
			RemoveEmptyMaps:  true,
			EmptyStringValues: []string{
				"-",
			},
			ExcludeKeys: []MapKey{
				{
					field: attributesField,
				},
				{
					field: resourceField,
				},
			},
		})

		inputLog := testLogStringBody()

		outputLogs, err := p.processLogs(context.Background(), inputLog)
		require.NoError(t, err)

		outResourceLogs := outputLogs.ResourceLogs().At(0)
		outLogRecord := outResourceLogs.ScopeLogs().At(0).LogRecords().At(0)

		require.Equal(t, rawResourceAttributes, outResourceLogs.Resource().Attributes().AsRaw())
		require.Equal(t, rawAttributes, outLogRecord.Attributes().AsRaw())
		require.Equal(t, pcommon.ValueTypeEmpty, outLogRecord.Body().Type())
	})
}

var rawResourceAttributes = map[string]any{
	"empty.key":        nil,
	"removable.string": "-",
	"resource_key":     "resource_value",
	"empty.map":        map[string]any{},
	"empty.slice":      []any{},
	"nested.map": map[string]any{
		"map": map[string]any{
			"some.key":    "-",
			"another.key": "-",
		},
	},
}

var rawAttributes = map[string]any{
	"empty.key":        nil,
	"removable.string": "-",
	"attr_key":         "attr_value",
	"empty.map":        map[string]any{},
	"empty.slice":      []any{},
}

var rawBody = map[string]any{
	"empty.key":        nil,
	"removable.string": "-",
	"body_key":         "body_value",
	"empty.map":        map[string]any{},
	"empty.slice":      []any{},
}

func testLog() plog.Logs {
	ld := plog.NewLogs()
	resourceLog := ld.ResourceLogs().AppendEmpty()
	resourceLog.Resource().Attributes().FromRaw(rawResourceAttributes)

	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
	logRecord := scopeLog.LogRecords().AppendEmpty()

	attrs := logRecord.Attributes()
	attrs.FromRaw(rawAttributes)

	mapBody := logRecord.Body().SetEmptyMap()
	mapBody.FromRaw(rawBody)

	return ld
}

func testLogEmptySliceBody() plog.Logs {
	ld := plog.NewLogs()
	resourceLog := ld.ResourceLogs().AppendEmpty()
	resourceLog.Resource().Attributes().FromRaw(rawResourceAttributes)

	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
	logRecord := scopeLog.LogRecords().AppendEmpty()

	attrs := logRecord.Attributes()
	attrs.FromRaw(rawAttributes)

	logRecord.Body().SetEmptySlice()

	return ld
}

func testLogStringBody() plog.Logs {
	ld := plog.NewLogs()
	resourceLog := ld.ResourceLogs().AppendEmpty()
	resourceLog.Resource().Attributes().FromRaw(rawResourceAttributes)

	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
	logRecord := scopeLog.LogRecords().AppendEmpty()

	attrs := logRecord.Attributes()
	attrs.FromRaw(rawAttributes)

	logRecord.Body().SetStr("-")

	return ld
}

func testMetrics() pmetric.Metrics {
	ms := pmetric.NewMetrics()
	resourceMetrics := ms.ResourceMetrics().AppendEmpty()
	resourceMetrics.Resource().Attributes().FromRaw(rawResourceAttributes)

	metricsSlice := resourceMetrics.ScopeMetrics().AppendEmpty().Metrics()

	gaugeMetric := metricsSlice.AppendEmpty()
	gaugeDp := gaugeMetric.SetEmptyGauge().DataPoints().AppendEmpty()
	gaugeDp.Attributes().FromRaw(rawAttributes)

	sumMetric := metricsSlice.AppendEmpty()
	sumDp := sumMetric.SetEmptySum().DataPoints().AppendEmpty()
	sumDp.Attributes().FromRaw(rawAttributes)

	summaryMetric := metricsSlice.AppendEmpty()
	summaryDp := summaryMetric.SetEmptySummary().DataPoints().AppendEmpty()
	summaryDp.Attributes().FromRaw(rawAttributes)

	histogramMetric := metricsSlice.AppendEmpty()
	histogramDp := histogramMetric.SetEmptyHistogram().DataPoints().AppendEmpty()
	histogramDp.Attributes().FromRaw(rawAttributes)

	expHistogramMetric := metricsSlice.AppendEmpty()
	expHistogramDp := expHistogramMetric.SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
	expHistogramDp.Attributes().FromRaw(rawAttributes)

	return ms
}

func requireMetricsAttrsEqual(t *testing.T, rawAttrs map[string]any, ms pmetric.MetricSlice) {
	require.Equal(t, rawAttrs, ms.At(0).Gauge().DataPoints().At(0).Attributes().AsRaw())
	require.Equal(t, rawAttrs, ms.At(1).Sum().DataPoints().At(0).Attributes().AsRaw())
	require.Equal(t, rawAttrs, ms.At(2).Summary().DataPoints().At(0).Attributes().AsRaw())
	require.Equal(t, rawAttrs, ms.At(3).Histogram().DataPoints().At(0).Attributes().AsRaw())
	require.Equal(t, rawAttrs, ms.At(4).ExponentialHistogram().DataPoints().At(0).Attributes().AsRaw())
}

func testTraces() ptrace.Traces {
	td := ptrace.NewTraces()

	resourceSpans := td.ResourceSpans().AppendEmpty()
	resourceSpans.Resource().Attributes().FromRaw(rawResourceAttributes)

	span := resourceSpans.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().FromRaw(rawAttributes)

	return td
}
