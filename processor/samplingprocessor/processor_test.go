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
	"testing"

	"github.com/observiq/bindplane-agent/expr"
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
