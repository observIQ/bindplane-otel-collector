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

package samplingprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/observiq/bindplane-agent/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/ptracetest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func Test_processTraces(t *testing.T) {
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()

	multipleSpansInput := ptrace.NewTraces()
	multipleSpansInput.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	multipleSpansInput.ResourceSpans().At(0).ScopeSpans().At(0).Spans().AppendEmpty()
	multipleSpansInput.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutInt("ID", 1)
	multipleSpansInput.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(1).Attributes().PutInt("ID", 2)

	multipleSpansExpected := ptrace.NewTraces()
	multipleSpansExpected.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	multipleSpansExpected.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutInt("ID", 2)

	testCases := []struct {
		desc      string
		dropRatio float64
		condition string
		input     ptrace.Traces
		expected  ptrace.Traces
	}{
		{
			desc:      "Always Drop true",
			condition: "true",
			dropRatio: 1.0,
			input:     td,
			expected:  ptrace.NewTraces(),
		},
		{
			desc:      "Never Drop true",
			condition: "true",
			dropRatio: 0.0,
			input:     td,
			expected:  td,
		},
		{
			desc:      "Always Drop false",
			condition: "false",
			dropRatio: 1.0,
			input:     td,
			expected:  td,
		},
		{
			desc:      "Never Drop false",
			condition: "false",
			dropRatio: 0.0,
			input:     td,
			expected:  td,
		},
		{
			desc:      "multiple spans condition",
			condition: `(attributes["ID"] == 1)`,
			dropRatio: 1.0,
			input:     multipleSpansInput,
			expected:  multipleSpansExpected,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &Config{
				DropRatio: tc.dropRatio,
				Condition: tc.condition,
			}

			ottlCondition, err := expr.NewOTTLSpanCondition(cfg.Condition, component.TelemetrySettings{Logger: zap.NewNop()})
			require.NoError(t, err)

			processor := newTracesSamplingProcessor(zap.NewNop(), cfg, ottlCondition)
			actual, err := processor.processTraces(context.Background(), tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func Test_processLogs(t *testing.T) {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()

	multipleRecordsInput := plog.NewLogs()
	multipleRecordsInput.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	multipleRecordsInput.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty()
	multipleRecordsInput.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().SetEmptyMap()
	multipleRecordsInput.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(1).Body().SetEmptyMap()
	multipleRecordsInput.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Map().PutInt("ID", 1)
	multipleRecordsInput.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(1).Body().Map().PutInt("ID", 2)

	multipleRecordsExpected := plog.NewLogs()
	multipleRecordsExpected.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	multipleRecordsExpected.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().SetEmptyMap()
	multipleRecordsExpected.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Map().PutInt("ID", 2)

	testCases := []struct {
		desc      string
		dropRatio float64
		condition string
		input     plog.Logs
		expected  plog.Logs
	}{
		{
			desc:      "Always Drop true",
			dropRatio: 1.0,
			condition: "true",
			input:     ld,
			expected:  plog.NewLogs(),
		},
		{
			desc:      "Never Drop true",
			dropRatio: 0.0,
			condition: "true",
			input:     ld,
			expected:  ld,
		},
		{
			desc:      "Always Drop false",
			dropRatio: 1.0,
			condition: "false",
			input:     ld,
			expected:  ld,
		},
		{
			desc:      "Never Drop false",
			dropRatio: 0.0,
			condition: "false",
			input:     ld,
			expected:  ld,
		},
		{
			desc:      "Always Drop condition multiple records",
			dropRatio: 1.0,
			condition: `(body["ID"] == 1)`,
			input:     multipleRecordsInput,
			expected:  multipleRecordsExpected,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &Config{
				DropRatio: tc.dropRatio,
				Condition: tc.condition,
			}

			ottlCondition, err := expr.NewOTTLLogRecordCondition(cfg.Condition, component.TelemetrySettings{Logger: zap.NewNop()})
			require.NoError(t, err)

			processor := newLogsSamplingProcessor(zap.NewNop(), cfg, ottlCondition)
			actual, err := processor.processLogs(context.Background(), tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func Test_processMetrics(t *testing.T) {
	md := pmetric.NewMetrics()
	metric := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetEmptyGauge()
	metric.Gauge().DataPoints().AppendEmpty()

	multipleMetrics := pmetric.NewMetrics()
	m1 := multipleMetrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m2 := multipleMetrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().AppendEmpty()
	m1.SetName("m1")
	m2.SetName("m2")

	multipleMetricsResult := pmetric.NewMetrics()
	m1r := multipleMetricsResult.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m1r.SetName("m2")

	testCases := []struct {
		desc      string
		condition string
		dropRatio float64
		input     pmetric.Metrics
		expected  pmetric.Metrics
	}{
		{
			desc:      "Always Drop true",
			condition: "true",
			dropRatio: 1.0,
			input:     md,
			expected:  pmetric.NewMetrics(),
		},
		{
			desc:      "Never Drop true",
			condition: "true",
			dropRatio: 0.0,
			input:     md,
			expected:  md,
		},
		{
			desc:      "Always Drop false",
			condition: "false",
			dropRatio: 1.0,
			input:     md,
			expected:  md,
		},
		{
			desc:      "Never Drop false",
			condition: "false",
			dropRatio: 0.0,
			input:     md,
			expected:  md,
		},
		{
			desc:      "multiple metrics condition",
			condition: `(name == "m1")`,
			dropRatio: 1.0,
			input:     multipleMetrics,
			expected:  multipleMetricsResult,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &Config{
				DropRatio: tc.dropRatio,
				Condition: tc.condition,
			}

			ottlCondition, err := expr.NewOTTLMetricCondition(cfg.Condition, component.TelemetrySettings{Logger: zap.NewNop()})
			require.NoError(t, err)

			processor := newMetricsSamplingProcessor(zap.NewNop(), cfg, ottlCondition)
			actual, err := processor.processMetrics(context.Background(), tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func Test_completeResourceDropping(t *testing.T) {
	cfg := &Config{
		DropRatio: 0.5,
		Condition: "true",
	}

	t.Run("verify no empty logs", func(t *testing.T) {
		ld := plog.NewLogs()
		for i := 0; i < 2; i++ {
			rl := ld.ResourceLogs().AppendEmpty()
			for j := 0; j < 3; j++ {
				sl := rl.ScopeLogs().AppendEmpty()

				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetEmptyMap()
				lr.Body().Map().PutStr("id", fmt.Sprintf("rl-%d/sl-%d/lr", i, j))
			}
		}

		ottlCondition, err := expr.NewOTTLLogRecordCondition(cfg.Condition, component.TelemetrySettings{Logger: zap.NewNop()})
		require.NoError(t, err)
		processor := newLogsSamplingProcessor(zap.NewNop(), cfg, ottlCondition)

		actual, err := processor.processLogs(context.Background(), ld)
		require.NoError(t, err)

		// can't know for sure how many logs are removed, but at 50% we can safely assume not all logs are removed
		err = plogtest.CompareLogs(plog.NewLogs(), actual)
		require.Error(t, err)

		for i := 0; i < actual.ResourceLogs().Len(); i++ {
			rl := actual.ResourceLogs().At(i)
			require.NotEqual(t, 0, rl.ScopeLogs().Len())
			for j := 0; j < rl.ScopeLogs().Len(); j++ {
				sl := rl.ScopeLogs().At(j)
				require.NotEqual(t, 0, sl.LogRecords().Len())
			}
		}
	})

	t.Run("verify no empty traces", func(t *testing.T) {
		td := ptrace.NewTraces()
		for i := 0; i < 2; i++ {
			rt := td.ResourceSpans().AppendEmpty()
			for j := 0; j < 3; j++ {
				st := rt.ScopeSpans().AppendEmpty()

				sd := st.Spans().AppendEmpty()
				m := sd.Attributes().PutEmptyMap("test")
				m.PutStr("id", fmt.Sprintf("rt-%d/st-%d/s", i, j))
			}
		}

		ottlCondition, err := expr.NewOTTLSpanCondition(cfg.Condition, component.TelemetrySettings{Logger: zap.NewNop()})
		require.NoError(t, err)
		processor := newTracesSamplingProcessor(zap.NewNop(), cfg, ottlCondition)

		actual, err := processor.processTraces(context.Background(), td)
		require.NoError(t, err)

		// can't know for sure how many traces are removed, but at 50% we can safely assume not all traces are removed
		err = ptracetest.CompareTraces(ptrace.NewTraces(), actual)
		require.Error(t, err)

		for i := 0; i < actual.ResourceSpans().Len(); i++ {
			rt := actual.ResourceSpans().At(i)
			require.NotEqual(t, 0, rt.ScopeSpans().Len())
			for j := 0; j < rt.ScopeSpans().Len(); j++ {
				st := rt.ScopeSpans().At(j)
				require.NotEqual(t, 0, st.Spans().Len())
			}
		}
	})

	t.Run("verify no empty metrics", func(t *testing.T) {
		md := pmetric.NewMetrics()
		for i := 0; i < 2; i++ {
			rm := md.ResourceMetrics().AppendEmpty()
			for j := 0; j < 3; j++ {
				sm := rm.ScopeMetrics().AppendEmpty()

				m := sm.Metrics().AppendEmpty()
				m.SetName(fmt.Sprintf("rm-%d/sm-%d/m", i, j))
			}
		}

		ottlCondition, err := expr.NewOTTLMetricCondition(cfg.Condition, component.TelemetrySettings{Logger: zap.NewNop()})
		require.NoError(t, err)
		processor := newMetricsSamplingProcessor(zap.NewNop(), cfg, ottlCondition)

		actual, err := processor.processMetrics(context.Background(), md)
		require.NoError(t, err)

		// can't know for sure how many traces are removed, but at 50% we can safely assume not all traces are removed
		err = pmetrictest.CompareMetrics(pmetric.NewMetrics(), actual)
		require.Error(t, err)

		for i := 0; i < actual.ResourceMetrics().Len(); i++ {
			rm := actual.ResourceMetrics().At(i)
			require.NotEqual(t, 0, rm.ScopeMetrics().Len())
			for j := 0; j < rm.ScopeMetrics().Len(); j++ {
				sm := rm.ScopeMetrics().At(j)
				require.NotEqual(t, 0, sm.Metrics().Len())
			}
		}

	})
}
