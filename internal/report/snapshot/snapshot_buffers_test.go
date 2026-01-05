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
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// decompress gzip decompresses the data
func decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

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

	compressedPayload, err := logBuffer.ConstructPayload(&plog.ProtoMarshaler{}, nil, nil, 1000)
	require.NoError(t, err)

	payload, err := decompress(compressedPayload)
	require.NoError(t, err)

	unmarshaler := &plog.ProtoUnmarshaler{}
	actual, err := unmarshaler.UnmarshalLogs(payload)
	require.NoError(t, err)
	require.Equal(t, 3, actual.LogRecordCount())
}

func TestLogsBufferConstructPayloadSampling(t *testing.T) {
	t.Run("Samples when payload exceeds max size", func(t *testing.T) {
		logBuffer := NewLogBuffer(2000)

		// Add many log records with unique content to prevent effective compression
		payload := plog.NewLogs()
		rl := payload.ResourceLogs().AppendEmpty()
		sl := rl.ScopeLogs().AppendEmpty()
		for i := 0; i < 1000; i++ {
			lr := sl.LogRecords().AppendEmpty()
			// Use unique content per log to reduce compression effectiveness
			lr.Body().SetStr(fmt.Sprintf("Log message %d with unique identifier %x", i, i*12345))
			lr.Attributes().PutStr("key", fmt.Sprintf("value-%d", i))
			lr.Attributes().PutInt("index", int64(i))
		}
		logBuffer.Add(payload)

		// Use a small max size to force sampling
		compressedPayload, err := logBuffer.ConstructPayload(&plog.ProtoMarshaler{}, nil, nil, 2000)
		require.NoError(t, err)
		require.LessOrEqual(t, len(compressedPayload), 2000)

		decompressed, err := decompress(compressedPayload)
		require.NoError(t, err)

		unmarshaler := &plog.ProtoUnmarshaler{}
		actual, err := unmarshaler.UnmarshalLogs(decompressed)
		require.NoError(t, err)
		// Should have fewer logs due to sampling (sampling was required to fit)
		require.Less(t, actual.LogRecordCount(), 1000)
	})

	t.Run("Returns error and clears buffer when too large even at 1%", func(t *testing.T) {
		logBuffer := NewLogBuffer(10000)

		// Add a very large payload
		payload := plog.NewLogs()
		rl := payload.ResourceLogs().AppendEmpty()
		sl := rl.ScopeLogs().AppendEmpty()
		for i := 0; i < 1000; i++ {
			lr := sl.LogRecords().AppendEmpty()
			lr.Body().SetStr("This is a test log message with a lot of content to make it very large and exceed the size limit even at 1 percent retention")
			lr.Attributes().PutStr("attribute1", "value1")
			lr.Attributes().PutStr("attribute2", "value2")
		}
		logBuffer.Add(payload)

		// Use an extremely small max size that even 1% can't fit
		_, err := logBuffer.ConstructPayload(&plog.ProtoMarshaler{}, nil, nil, 10)
		require.Error(t, err)
		require.Contains(t, err.Error(), "snapshot buffer is too large to construct payload")

		// Buffer should be cleared
		require.Equal(t, 0, logBuffer.Len())
	})

	t.Run("Empty buffer returns empty payload", func(t *testing.T) {
		logBuffer := NewLogBuffer(100)
		compressedPayload, err := logBuffer.ConstructPayload(&plog.ProtoMarshaler{}, nil, nil, 1000)
		require.NoError(t, err)

		decompressed, err := decompress(compressedPayload)
		require.NoError(t, err)

		unmarshaler := &plog.ProtoUnmarshaler{}
		actual, err := unmarshaler.UnmarshalLogs(decompressed)
		require.NoError(t, err)
		require.Equal(t, 0, actual.LogRecordCount())
	})
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

	compressedPayload, err := metricBuffer.ConstructPayload(&pmetric.ProtoMarshaler{}, nil, nil, 1000)
	require.NoError(t, err)

	payload, err := decompress(compressedPayload)
	require.NoError(t, err)

	unmarshaler := &pmetric.ProtoUnmarshaler{}
	actual, err := unmarshaler.UnmarshalMetrics(payload)
	require.NoError(t, err)
	require.Equal(t, 3, actual.DataPointCount())
}

func TestMetricBufferConstructPayloadSampling(t *testing.T) {
	t.Run("Samples when payload exceeds max size", func(t *testing.T) {
		metricBuffer := NewMetricBuffer(1000)

		// Add many data points to create a large payload
		payload := pmetric.NewMetrics()
		rm := payload.ResourceMetrics().AppendEmpty()
		sm := rm.ScopeMetrics().AppendEmpty()
		m := sm.Metrics().AppendEmpty()
		m.SetName("test_metric")
		m.SetEmptyGauge()
		for i := 0; i < 500; i++ {
			dp := m.Gauge().DataPoints().AppendEmpty()
			dp.SetIntValue(int64(i))
			dp.Attributes().PutStr("key", "value")
		}
		metricBuffer.Add(payload)

		// Use a small max size to force sampling
		compressedPayload, err := metricBuffer.ConstructPayload(&pmetric.ProtoMarshaler{}, nil, nil, 500)
		require.NoError(t, err)
		require.LessOrEqual(t, len(compressedPayload), 500)

		decompressed, err := decompress(compressedPayload)
		require.NoError(t, err)

		unmarshaler := &pmetric.ProtoUnmarshaler{}
		actual, err := unmarshaler.UnmarshalMetrics(decompressed)
		require.NoError(t, err)
		// Should have fewer data points due to sampling
		require.Less(t, actual.DataPointCount(), 500)
	})

	t.Run("Returns error and clears buffer when too large even at 1%", func(t *testing.T) {
		metricBuffer := NewMetricBuffer(10000)

		// Add a very large payload
		payload := pmetric.NewMetrics()
		rm := payload.ResourceMetrics().AppendEmpty()
		sm := rm.ScopeMetrics().AppendEmpty()
		m := sm.Metrics().AppendEmpty()
		m.SetName("test_metric")
		m.SetEmptyGauge()
		for i := 0; i < 1000; i++ {
			dp := m.Gauge().DataPoints().AppendEmpty()
			dp.SetIntValue(int64(i))
			dp.Attributes().PutStr("attribute1", "value1")
			dp.Attributes().PutStr("attribute2", "value2")
		}
		metricBuffer.Add(payload)

		// Use an extremely small max size that even 1% can't fit
		_, err := metricBuffer.ConstructPayload(&pmetric.ProtoMarshaler{}, nil, nil, 10)
		require.Error(t, err)
		require.Contains(t, err.Error(), "snapshot buffer is too large to construct payload")

		// Buffer should be cleared
		require.Equal(t, 0, metricBuffer.Len())
	})

	t.Run("Empty buffer returns empty payload", func(t *testing.T) {
		metricBuffer := NewMetricBuffer(100)
		compressedPayload, err := metricBuffer.ConstructPayload(&pmetric.ProtoMarshaler{}, nil, nil, 1000)
		require.NoError(t, err)

		decompressed, err := decompress(compressedPayload)
		require.NoError(t, err)

		unmarshaler := &pmetric.ProtoUnmarshaler{}
		actual, err := unmarshaler.UnmarshalMetrics(decompressed)
		require.NoError(t, err)
		require.Equal(t, 0, actual.DataPointCount())
	})
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

	compressedPayload, err := traceBuffer.ConstructPayload(&ptrace.ProtoMarshaler{}, nil, nil, 1000)
	require.NoError(t, err)

	payload, err := decompress(compressedPayload)
	require.NoError(t, err)

	unmarshaler := &ptrace.ProtoUnmarshaler{}
	actual, err := unmarshaler.UnmarshalTraces(payload)
	require.NoError(t, err)
	require.Equal(t, 3, actual.SpanCount())
}

func TestTraceBufferConstructPayloadSampling(t *testing.T) {
	t.Run("Samples when payload exceeds max size", func(t *testing.T) {
		traceBuffer := NewTraceBuffer(2000)

		// Add many spans with unique content to prevent effective compression
		payload := ptrace.NewTraces()
		rs := payload.ResourceSpans().AppendEmpty()
		ss := rs.ScopeSpans().AppendEmpty()
		for i := 0; i < 1000; i++ {
			span := ss.Spans().AppendEmpty()
			span.SetName(fmt.Sprintf("test-span-%d", i))
			span.Attributes().PutStr("key", fmt.Sprintf("value-%d", i))
			span.Attributes().PutInt("index", int64(i))
		}
		traceBuffer.Add(payload)

		// Use a small max size to force sampling
		compressedPayload, err := traceBuffer.ConstructPayload(&ptrace.ProtoMarshaler{}, nil, nil, 2000)
		require.NoError(t, err)
		require.LessOrEqual(t, len(compressedPayload), 2000)

		decompressed, err := decompress(compressedPayload)
		require.NoError(t, err)

		unmarshaler := &ptrace.ProtoUnmarshaler{}
		actual, err := unmarshaler.UnmarshalTraces(decompressed)
		require.NoError(t, err)
		// Should have fewer spans due to sampling (sampling was required to fit)
		require.Less(t, actual.SpanCount(), 1000)
	})

	t.Run("Returns error and clears buffer when too large even at 1%", func(t *testing.T) {
		traceBuffer := NewTraceBuffer(10000)

		// Add a very large payload
		payload := ptrace.NewTraces()
		rs := payload.ResourceSpans().AppendEmpty()
		ss := rs.ScopeSpans().AppendEmpty()
		for i := 0; i < 1000; i++ {
			span := ss.Spans().AppendEmpty()
			span.SetName("test-span-with-a-longer-name-to-increase-payload-size")
			span.Attributes().PutStr("attribute1", "value1")
			span.Attributes().PutStr("attribute2", "value2")
		}
		traceBuffer.Add(payload)

		// Use an extremely small max size that even 1% can't fit
		_, err := traceBuffer.ConstructPayload(&ptrace.ProtoMarshaler{}, nil, nil, 10)
		require.Error(t, err)
		require.Contains(t, err.Error(), "snapshot buffer is too large to construct payload")

		// Buffer should be cleared
		require.Equal(t, 0, traceBuffer.Len())
	})

	t.Run("Empty buffer returns empty payload", func(t *testing.T) {
		traceBuffer := NewTraceBuffer(100)
		compressedPayload, err := traceBuffer.ConstructPayload(&ptrace.ProtoMarshaler{}, nil, nil, 1000)
		require.NoError(t, err)

		decompressed, err := decompress(compressedPayload)
		require.NoError(t, err)

		unmarshaler := &ptrace.ProtoUnmarshaler{}
		actual, err := unmarshaler.UnmarshalTraces(decompressed)
		require.NoError(t, err)
		require.Equal(t, 0, actual.SpanCount())
	})
}

func TestCompress(t *testing.T) {
	t.Run("Compresses and decompresses data correctly", func(t *testing.T) {
		original := []byte("This is some test data that should be compressed and decompressed correctly")
		compressed, err := compress(original)
		require.NoError(t, err)
		require.NotEmpty(t, compressed)

		decompressed, err := decompress(compressed)
		require.NoError(t, err)
		require.Equal(t, original, decompressed)
	})

	t.Run("Compresses empty data", func(t *testing.T) {
		compressed, err := compress([]byte{})
		require.NoError(t, err)

		decompressed, err := decompress(compressed)
		require.NoError(t, err)
		require.Empty(t, decompressed)
	})

	t.Run("Compressed data is smaller for repetitive content", func(t *testing.T) {
		// Create repetitive data that compresses well
		original := make([]byte, 10000)
		for i := range original {
			original[i] = 'a'
		}
		compressed, err := compress(original)
		require.NoError(t, err)
		require.Less(t, len(compressed), len(original))
	})
}
