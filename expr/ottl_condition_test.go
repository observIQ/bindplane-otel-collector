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

func TestLogCondition(t *testing.T) {
	testCases := []struct {
		name       string
		resource   pcommon.Resource
		scope      pcommon.InstrumentationScope
		log        plog.LogRecord
		expression string
		result     bool
	}{
		{
			name:       "Log attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `attributes["key1"] == "val1"`,
			result:     true,
		},
		{
			name:       "Converter",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `"val1-concat" == Concat([attributes["key1"], "concat"], "-")`,
			result:     true,
		},
		{
			name:       "Resource attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `resource.attributes["resource2"] != 1`,
			result:     false,
		},
		{
			name:       "Log body",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `body["body_key"] == "cool-thing"`,
			result:     true,
		},
		{
			name:       "Value does not exist",
			resource:   testResource(t),
			scope:      testScope(t),
			log:        testLogRecord(t),
			expression: `resource.attributes["dne"] == nil`,
			result:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tCtx := ottllog.NewTransformContext(tc.log, tc.scope, tc.resource)

			expr, err := NewOTTLLogRecordCondition(tc.expression, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			res, err := expr.Match(context.Background(), tCtx)
			require.NoError(t, err)
			require.Equal(t, tc.result, res)
		})
	}
}

func TestDatapointCondition(t *testing.T) {
	testCases := []struct {
		name        string
		resource    pcommon.Resource
		scope       pcommon.InstrumentationScope
		metricSlice pmetric.MetricSlice
		expression  string
		result      bool
	}{
		{
			name:        "Datapoint attribute",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `attributes["key1"] == "val1"`,
			result:      true,
		},
		{
			name:        "Converter",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `"val1-concat" == Concat([attributes["key1"], "concat"], "-")`,
			result:      true,
		},
		{
			name:        "Resource attribute",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `resource.attributes["resource2"] != 1`,
			result:      false,
		},
		{
			name:        "Metric name",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `metric.name == "test.metric"`,
			result:      true,
		},
		{
			name:        "Datapoint value",
			resource:    testResource(t),
			scope:       testScope(t),
			metricSlice: testMetricSlice(t),
			expression:  `value_int != 5`,
			result:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metric := tc.metricSlice.At(0)
			dp := metric.Gauge().DataPoints().At(0)

			tCtx := ottldatapoint.NewTransformContext(dp, metric, tc.metricSlice, tc.scope, tc.resource)

			expr, err := NewOTTLDatapointCondition(tc.expression, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			res, err := expr.Match(context.Background(), tCtx)
			require.NoError(t, err)
			require.Equal(t, tc.result, res)
		})
	}
}

func TestSpanCondition(t *testing.T) {
	testCases := []struct {
		name       string
		resource   pcommon.Resource
		scope      pcommon.InstrumentationScope
		span       ptrace.Span
		expression string
		result     bool
	}{
		{
			name:       "Span attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `attributes["key1"] == "val1"`,
			result:     true,
		},
		{
			name:       "Converter",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `"val1-concat" == Concat([attributes["key1"], "concat"], "-")`,
			result:     true,
		},
		{
			name:       "Resource attribute",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `resource.attributes["resource2"] != 1`,
			result:     false,
		},
		{
			name:       "Span name",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `name != "span-name"`,
			result:     false,
		},
		{
			name:       "Span kind",
			resource:   testResource(t),
			scope:      testScope(t),
			span:       testSpan(t),
			expression: `kind == SPAN_KIND_INTERNAL`,
			result:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tCtx := ottlspan.NewTransformContext(tc.span, tc.scope, tc.resource)

			expr, err := NewOTTLSpanCondition(tc.expression, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			res, err := expr.Match(context.Background(), tCtx)
			require.NoError(t, err)
			require.Equal(t, tc.result, res)
		})
	}
}
