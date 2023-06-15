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

package expr

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestLogExpression(t *testing.T) {
	testCases := []struct {
		name       string
		resource   pcommon.Resource
		scope      pcommon.InstrumentationScope
		log        plog.LogRecord
		expression string
		result     any
	}{
		{
			name:       "Log attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `attributes["key1"]`,
			result:     "val1",
		},
		{
			name:       "Converter",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `Concat([attributes["key1"], "concat"], "-")`,
			result:     "val1-concat",
		},
		{
			name:       "Resource attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `resource.attributes["resource2"]`,
			result:     int64(1),
		},
		{
			name:       "Log body",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `body["body_key"]`,
			result:     "cool-thing",
		},
		{
			name:       "Value does not exist",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `resource.attributes["dne"]`,
			result:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tCtx := ottllog.NewTransformContext(tc.log, tc.scope, tc.resource)

			expr, err := NewOTTLLogRecordExpression(tc.expression, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			res, err := expr.Execute(context.Background(), tCtx)
			require.NoError(t, err)
			require.Equal(t, tc.result, res)
		})
	}
}

func TestDatapointExpression(t *testing.T) {
	testCases := []struct {
		name        string
		resource    pcommon.Resource
		scope       pcommon.InstrumentationScope
		metricSlice pmetric.MetricSlice
		expression  string
		result      any
	}{
		{
			name:        "Datapoint attribute",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `attributes["key1"]`,
			result:      "val1",
		},
		{
			name:        "Converter",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `Concat([attributes["key1"], "concat"], "-")`,
			result:      "val1-concat",
		},
		{
			name:        "Resource attribute",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `resource.attributes["resource2"]`,
			result:      int64(1),
		},
		{
			name:        "Metric name",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `metric.name`,
			result:      "test.metric",
		},
		{
			name:        "Datapoint value",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `value_int`,
			result:      int64(5),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metric := tc.metricSlice.At(0)
			dp := metric.Gauge().DataPoints().At(0)

			tCtx := ottldatapoint.NewTransformContext(dp, metric, tc.metricSlice, tc.scope, tc.resource)

			expr, err := NewOTTLDatapointExpression(tc.expression, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			res, err := expr.Execute(context.Background(), tCtx)
			require.NoError(t, err)
			require.Equal(t, tc.result, res)
		})
	}
}

func TestSpanExpression(t *testing.T) {
	testCases := []struct {
		name       string
		resource   pcommon.Resource
		scope      pcommon.InstrumentationScope
		span       ptrace.Span
		expression string
		result     any
	}{
		{
			name:       "Span attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `attributes["key1"]`,
			result:     "val1",
		},
		{
			name:       "Converter",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `Concat([attributes["key1"], "concat"], "-")`,
			result:     "val1-concat",
		},
		{
			name:       "Resource attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `resource.attributes["resource2"]`,
			result:     int64(1),
		},
		{
			name:       "Span name",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `name`,
			result:     "span-name",
		},
		{
			name:       "Span kind",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `kind`,
			result:     int64(ptrace.SpanKindInternal),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tCtx := ottlspan.NewTransformContext(tc.span, tc.scope, tc.resource)

			expr, err := NewOTTLSpanExpression(tc.expression, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			res, err := expr.Execute(context.Background(), tCtx)
			require.NoError(t, err)
			require.Equal(t, tc.result, res)
		})
	}
}

func testResource(t *testing.T) pcommon.Resource {
	res := pcommon.NewResource()
	err := res.Attributes().FromRaw(
		map[string]any{
			"key":       "value",
			"resource2": int64(1),
		},
	)
	require.NoError(t, err)

	return res
}

func testScope(t *testing.T) pcommon.InstrumentationScope {
	res := pcommon.NewInstrumentationScope()
	err := res.Attributes().FromRaw(
		map[string]any{
			"key":       "value",
			"resource2": 1,
		},
	)
	require.NoError(t, err)

	res.SetName("otel/test")

	return res
}

var testAttributes = map[string]any{
	"key1":      "val1",
	"errorCode": 17,
}

func testLogRecord(t *testing.T) plog.LogRecord {
	lr := plog.NewLogRecord()

	err := lr.Attributes().FromRaw(testAttributes)
	require.NoError(t, err)

	err = lr.Body().FromRaw(map[string]any{
		"body_key": "cool-thing",
	})
	require.NoError(t, err)

	return lr
}

func testMetricSlice(t *testing.T) pmetric.MetricSlice {
	ms := pmetric.NewMetricSlice()
	m := ms.AppendEmpty()
	m.SetName("test.metric")
	m.SetDescription("A test metric")

	dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetIntValue(5)
	err := dp.Attributes().FromRaw(testAttributes)
	require.NoError(t, err)

	return ms
}

func testSpan(t *testing.T) ptrace.Span {
	span := ptrace.NewSpan()

	err := span.Attributes().FromRaw(testAttributes)
	require.NoError(t, err)

	span.SetName("span-name")
	span.SetKind(ptrace.SpanKindInternal)

	return span
}
