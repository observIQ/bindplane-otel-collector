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

func TestNewLogBuffer(t *testing.T) {
	idealSize := 100
	expected := &LogBuffer{
		buffer:    make([]plog.Logs, 0),
		idealSize: idealSize,
	}

	actual := NewLogBuffer(idealSize)
	require.Equal(t, expected, actual)
}

func TestLogBufferAdd(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Insert larger than idealSize",
			testFunc: func(t *testing.T) {
				logBuffer := NewLogBuffer(1)

				// Seed buffer with one entry
				initialBufferContents := plog.NewLogs()
				initialBufferContents.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				logBuffer.buffer = append(logBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := plog.NewLogs()
				rl := toAdd.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()

				// Add to log buffer
				logBuffer.Add(toAdd)

				assert.Equal(t, 3, logBuffer.Len())
			},
		},
		{
			desc: "Insert + current size less than idealSize",
			testFunc: func(t *testing.T) {
				logBuffer := NewLogBuffer(5)

				// Seed buffer with one entry
				initialBufferContents := plog.NewLogs()
				initialBufferContents.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				logBuffer.buffer = append(logBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := plog.NewLogs()
				rl := toAdd.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()

				// Add to log buffer
				logBuffer.Add(toAdd)

				assert.Equal(t, 4, logBuffer.Len())
			},
		},
		{
			desc: "Insert + current size more than idealSize, removing oldest is ok",
			testFunc: func(t *testing.T) {
				logBuffer := NewLogBuffer(4)

				// Seed buffer with several payloads
				initialBufferContents := plog.NewLogs()
				initialBufferContents.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				logBuffer.buffer = append(logBuffer.buffer, initialBufferContents)

				secondBufferContents := plog.NewLogs()
				secondBufferContents.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				logBuffer.buffer = append(logBuffer.buffer, secondBufferContents)

				// Create payload with more than ideal size
				toAdd := plog.NewLogs()
				rl := toAdd.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()

				// Add to log buffer
				logBuffer.Add(toAdd)

				assert.Equal(t, 4, logBuffer.Len())
			},
		},
		{
			desc: "Insert + current size more than idealSize, don't remove oldest",
			testFunc: func(t *testing.T) {
				logBuffer := NewLogBuffer(4)

				// Seed buffer with several payloads
				initialBufferContents := plog.NewLogs()
				initialSl := initialBufferContents.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
				initialSl.LogRecords().AppendEmpty()
				initialSl.LogRecords().AppendEmpty()
				initialSl.LogRecords().AppendEmpty()
				logBuffer.buffer = append(logBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := plog.NewLogs()
				rl := toAdd.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()

				// Add to log buffer
				logBuffer.Add(toAdd)

				assert.Equal(t, 5, logBuffer.Len())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestLogsBufferConstructPayload(t *testing.T) {
	logBuffer := NewLogBuffer(4)

	payloadOne := plog.NewLogs()
	payloadOne.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	logBuffer.Add(payloadOne)

	payloadTwo := plog.NewLogs()
	payloadTwo.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	logBuffer.Add(payloadTwo)

	payloadThree := plog.NewLogs()
	payloadThree.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	logBuffer.Add(payloadThree)

	payload, err := logBuffer.ConstructPayload(nil, nil)
	require.NoError(t, err)

	unmarshaler := &plog.ProtoUnmarshaler{}
	actual, err := unmarshaler.UnmarshalLogs(payload)
	require.NoError(t, err)
	require.Equal(t, 3, actual.LogRecordCount())
}

func TestNewMetricBuffer(t *testing.T) {
	idealSize := 100
	expected := &MetricBuffer{
		buffer:    make([]pmetric.Metrics, 0),
		idealSize: idealSize,
	}

	actual := NewMetricBuffer(idealSize)
	require.Equal(t, expected, actual)
}

func TestMetricBufferAdd(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Insert larger than idealSize",
			testFunc: func(t *testing.T) {
				metricBuffer := NewMetricBuffer(1)

				// Seed buffer with one entry
				initialBufferContents := pmetric.NewMetrics()
				initialMetric := initialBufferContents.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				initialMetric.SetEmptyGauge()
				initialMetric.Gauge().DataPoints().AppendEmpty()
				metricBuffer.buffer = append(metricBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := pmetric.NewMetrics()
				rm := toAdd.ResourceMetrics().AppendEmpty()
				sm := rm.ScopeMetrics().AppendEmpty()
				metric := sm.Metrics().AppendEmpty()
				metric.SetEmptyGauge()
				metric.Gauge().DataPoints().AppendEmpty()
				metric.Gauge().DataPoints().AppendEmpty()
				metric.Gauge().DataPoints().AppendEmpty()

				// Add to log buffer
				metricBuffer.Add(toAdd)

				assert.Equal(t, 3, metricBuffer.Len())
			},
		},
		{
			desc: "Insert + current size less than idealSize",
			testFunc: func(t *testing.T) {
				metricBuffer := NewMetricBuffer(5)

				// Seed buffer with one entry
				initialBufferContents := pmetric.NewMetrics()
				initialBufferContents.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				initialMetric := initialBufferContents.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				initialMetric.SetEmptyGauge()
				initialMetric.Gauge().DataPoints().AppendEmpty()
				metricBuffer.buffer = append(metricBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := pmetric.NewMetrics()
				rm := toAdd.ResourceMetrics().AppendEmpty()
				sm := rm.ScopeMetrics().AppendEmpty()
				metric := sm.Metrics().AppendEmpty()
				metric.SetEmptyGauge()
				metric.Gauge().DataPoints().AppendEmpty()
				metric.Gauge().DataPoints().AppendEmpty()
				metric.Gauge().DataPoints().AppendEmpty()

				// Add to log buffer
				metricBuffer.Add(toAdd)

				assert.Equal(t, 4, metricBuffer.Len())
			},
		},
		{
			desc: "Insert + current size more than idealSize, removing oldest is ok",
			testFunc: func(t *testing.T) {
				metricBuffer := NewMetricBuffer(4)

				// Seed buffer with several payloads
				initialBufferContents := pmetric.NewMetrics()
				initialMetric := initialBufferContents.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				initialMetric.SetEmptyGauge()
				initialMetric.Gauge().DataPoints().AppendEmpty()
				metricBuffer.buffer = append(metricBuffer.buffer, initialBufferContents)

				secondBufferContents := pmetric.NewMetrics()
				secondMetric := secondBufferContents.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				secondMetric.SetEmptyGauge()
				secondMetric.Gauge().DataPoints().AppendEmpty()
				metricBuffer.buffer = append(metricBuffer.buffer, secondBufferContents)

				// Create payload with more than ideal size
				toAdd := pmetric.NewMetrics()
				rm := toAdd.ResourceMetrics().AppendEmpty()
				sm := rm.ScopeMetrics().AppendEmpty()
				metric := sm.Metrics().AppendEmpty()
				metric.SetEmptyGauge()
				metric.Gauge().DataPoints().AppendEmpty()
				metric.Gauge().DataPoints().AppendEmpty()
				metric.Gauge().DataPoints().AppendEmpty()

				// Add to log buffer
				metricBuffer.Add(toAdd)

				assert.Equal(t, 4, metricBuffer.Len())
			},
		},
		{
			desc: "Insert + current size more than idealSize, don't remove oldest",
			testFunc: func(t *testing.T) {
				metricBuffer := NewMetricBuffer(4)

				// Seed buffer with several payloads
				initialBufferContents := pmetric.NewMetrics()
				initialMetric := initialBufferContents.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				initialMetric.SetEmptyGauge()
				initialMetric.Gauge().DataPoints().AppendEmpty()
				initialMetric.Gauge().DataPoints().AppendEmpty()
				initialMetric.Gauge().DataPoints().AppendEmpty()
				metricBuffer.buffer = append(metricBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := pmetric.NewMetrics()
				rm := toAdd.ResourceMetrics().AppendEmpty()
				sm := rm.ScopeMetrics().AppendEmpty()
				metric := sm.Metrics().AppendEmpty()
				metric.SetEmptyGauge()
				metric.Gauge().DataPoints().AppendEmpty()
				metric.Gauge().DataPoints().AppendEmpty()

				// Add to log buffer
				metricBuffer.Add(toAdd)

				assert.Equal(t, 5, metricBuffer.Len())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestMetricBufferConstructPayload(t *testing.T) {
	metricBuffer := NewMetricBuffer(4)

	payloadOne := pmetric.NewMetrics()
	pOneMetric := payloadOne.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	pOneMetric.SetEmptyGauge()
	pOneMetric.Gauge().DataPoints().AppendEmpty()
	metricBuffer.Add(payloadOne)

	payloadTwo := pmetric.NewMetrics()
	pTwoMetric := payloadTwo.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	pTwoMetric.SetEmptyGauge()
	pTwoMetric.Gauge().DataPoints().AppendEmpty()
	metricBuffer.Add(payloadTwo)

	payloadThree := pmetric.NewMetrics()
	pThreeMetric := payloadThree.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	pThreeMetric.SetEmptyGauge()
	pThreeMetric.Gauge().DataPoints().AppendEmpty()
	metricBuffer.Add(payloadThree)

	payload, err := metricBuffer.ConstructPayload()
	require.NoError(t, err)

	unmarshaler := &pmetric.ProtoUnmarshaler{}
	actual, err := unmarshaler.UnmarshalMetrics(payload)
	require.NoError(t, err)
	require.Equal(t, 3, actual.DataPointCount())
}

func TestNewTraceBuffer(t *testing.T) {
	idealSize := 100
	expected := &TraceBuffer{
		buffer:    make([]ptrace.Traces, 0),
		idealSize: idealSize,
	}

	actual := NewTraceBuffer(idealSize)
	require.Equal(t, expected, actual)
}

func TestTraceBufferAdd(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Insert larger than idealSize",
			testFunc: func(t *testing.T) {
				traceBuffer := NewTraceBuffer(1)

				// Seed buffer with one entry
				initialBufferContents := ptrace.NewTraces()
				initialBufferContents.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				traceBuffer.buffer = append(traceBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := ptrace.NewTraces()
				rl := toAdd.ResourceSpans().AppendEmpty()
				sl := rl.ScopeSpans().AppendEmpty()
				sl.Spans().AppendEmpty()
				sl.Spans().AppendEmpty()
				sl.Spans().AppendEmpty()

				// Add to log buffer
				traceBuffer.Add(toAdd)

				assert.Equal(t, 3, traceBuffer.Len())
			},
		},
		{
			desc: "Insert + current size less than idealSize",
			testFunc: func(t *testing.T) {
				traceBuffer := NewTraceBuffer(5)

				// Seed buffer with one entry
				initialBufferContents := ptrace.NewTraces()
				initialBufferContents.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				traceBuffer.buffer = append(traceBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := ptrace.NewTraces()
				rl := toAdd.ResourceSpans().AppendEmpty()
				sl := rl.ScopeSpans().AppendEmpty()
				sl.Spans().AppendEmpty()
				sl.Spans().AppendEmpty()
				sl.Spans().AppendEmpty()

				// Add to log buffer
				traceBuffer.Add(toAdd)

				assert.Equal(t, 4, traceBuffer.Len())
			},
		},
		{
			desc: "Insert + current size more than idealSize, removing oldest is ok",
			testFunc: func(t *testing.T) {
				traceBuffer := NewTraceBuffer(4)

				// Seed buffer with several payloads
				initialBufferContents := ptrace.NewTraces()
				initialBufferContents.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				traceBuffer.buffer = append(traceBuffer.buffer, initialBufferContents)

				secondBufferContents := ptrace.NewTraces()
				secondBufferContents.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				traceBuffer.buffer = append(traceBuffer.buffer, secondBufferContents)

				// Create payload with more than ideal size
				toAdd := ptrace.NewTraces()
				rl := toAdd.ResourceSpans().AppendEmpty()
				sl := rl.ScopeSpans().AppendEmpty()
				sl.Spans().AppendEmpty()
				sl.Spans().AppendEmpty()
				sl.Spans().AppendEmpty()

				// Add to log buffer
				traceBuffer.Add(toAdd)

				assert.Equal(t, 4, traceBuffer.Len())
			},
		},
		{
			desc: "Insert + current size more than idealSize, don't remove oldest",
			testFunc: func(t *testing.T) {
				traceBuffer := NewTraceBuffer(4)

				// Seed buffer with several payloads
				initialBufferContents := ptrace.NewTraces()
				initialSl := initialBufferContents.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
				initialSl.Spans().AppendEmpty()
				initialSl.Spans().AppendEmpty()
				initialSl.Spans().AppendEmpty()
				traceBuffer.buffer = append(traceBuffer.buffer, initialBufferContents)

				// Create payload with more than ideal size
				toAdd := ptrace.NewTraces()
				rl := toAdd.ResourceSpans().AppendEmpty()
				sl := rl.ScopeSpans().AppendEmpty()
				sl.Spans().AppendEmpty()
				sl.Spans().AppendEmpty()

				// Add to log buffer
				traceBuffer.Add(toAdd)

				assert.Equal(t, 5, traceBuffer.Len())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestTraceBufferConstructPayload(t *testing.T) {
	traceBuffer := NewTraceBuffer(4)

	payloadOne := ptrace.NewTraces()
	payloadOne.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	traceBuffer.Add(payloadOne)

	payloadTwo := ptrace.NewTraces()
	payloadTwo.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	traceBuffer.Add(payloadTwo)

	payloadThree := ptrace.NewTraces()
	payloadThree.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	traceBuffer.Add(payloadThree)

	payload, err := traceBuffer.ConstructPayload()
	require.NoError(t, err)

	unmarshaler := &ptrace.ProtoUnmarshaler{}
	actual, err := unmarshaler.UnmarshalTraces(payload)
	require.NoError(t, err)
	require.Equal(t, 3, actual.SpanCount())
}
