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
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// LogBuffer is a buffer for plog.Logs
type LogBuffer struct {
	mutex     sync.Mutex
	buffer    []plog.Logs
	idealSize int
}

// NewLogBuffer creates a logBuffer with the ideal size set
func NewLogBuffer(idealSize int) *LogBuffer {
	return &LogBuffer{
		buffer:    make([]plog.Logs, 0),
		idealSize: idealSize,
	}
}

// Len counts the number of log records in all Log payloads in buffer
func (l *LogBuffer) Len() int {
	size := 0
	for _, ld := range l.buffer {
		size += ld.LogRecordCount()
	}

	return size
}

// Add adds the new log payload and adjust buffer to keep ideal size
func (l *LogBuffer) Add(ld plog.Logs) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	logSize := ld.LogRecordCount()
	bufferSize := l.Len()
	switch {
	// The number of logs is more than idealSize so reset this to just this log set
	case logSize > l.idealSize:
		l.buffer = []plog.Logs{ld}

	// Haven't reached idealSize yet so add this
	case logSize+bufferSize < l.idealSize:
		l.buffer = append(l.buffer, ld)

	// Adding this will put us over idealSize so and add the new logs.
	// Only remove the oldest if it does not bring buffer under idealSize
	case logSize+bufferSize >= l.idealSize:
		l.buffer = append(l.buffer, ld)

		// Remove items from the buffer until we find one that if we remove it will put us under the ideal size
		for {
			newBufferSize := l.Len()
			oldest := l.buffer[0]

			// If removing this one will put us under ideal size then break
			if newBufferSize-oldest.LogRecordCount() < l.idealSize {
				break
			}

			// Remove the oldest
			l.buffer = l.buffer[1:]
		}
	}
}

// ConstructPayload condenses the buffer and serializes to protobuf
func (l *LogBuffer) ConstructPayload(searchQuery *string, minimumTimestamp *time.Time) ([]byte, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	logsMarshaler := &plog.ProtoMarshaler{}

	payloadLogs := plog.NewLogs()
	for _, ld := range l.buffer {
		ld.ResourceLogs().MoveAndAppendTo(payloadLogs.ResourceLogs())
	}

	// update the buffer to retain the current logs which were moved to the new payload
	l.buffer = []plog.Logs{payloadLogs}

	// Filter the payload
	filteredPayload := filterLogs(payloadLogs, searchQuery, minimumTimestamp)

	payload, err := logsMarshaler.MarshalLogs(filteredPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to construct payload: %w", err)
	}

	return payload, nil
}

// MetricBuffer is a buffer for pmetric.Metrics
type MetricBuffer struct {
	mutex     sync.Mutex
	buffer    []pmetric.Metrics
	idealSize int
}

// NewMetricBuffer creates a metricBuffer with the ideal size set
func NewMetricBuffer(idealSize int) *MetricBuffer {
	return &MetricBuffer{
		buffer:    make([]pmetric.Metrics, 0),
		idealSize: idealSize,
	}
}

// Len counts the number of data points in all Metric payloads in buffer
func (l *MetricBuffer) Len() int {
	size := 0
	for _, md := range l.buffer {
		size += md.DataPointCount()
	}

	return size
}

// Add adds the new metric payload and adjust buffer to keep ideal size
func (l *MetricBuffer) Add(md pmetric.Metrics) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	metricSize := md.DataPointCount()
	bufferSize := l.Len()
	switch {
	// The number of metrics is more than idealSize so reset this to just this metric set
	case metricSize > l.idealSize:
		l.buffer = []pmetric.Metrics{md}

	// Haven't reached idealSize yet so add this
	case metricSize+bufferSize < l.idealSize:
		l.buffer = append(l.buffer, md)

	// Adding this will put us over idealSize so and add the new metrics.
	// Only remove the oldest if it does not bring buffer under idealSize
	case metricSize+bufferSize >= l.idealSize:
		l.buffer = append(l.buffer, md)

		// Remove items from the buffer until we find one that if we remove it will put us under the ideal size
		for {
			newBufferSize := l.Len()
			oldest := l.buffer[0]

			// If removing this one will put us under ideal size then break
			if newBufferSize-oldest.DataPointCount() < l.idealSize {
				break
			}

			// Remove the oldest
			l.buffer = l.buffer[1:]
		}
	}
}

// ConstructPayload condenses the buffer and serializes to protobuf
func (l *MetricBuffer) ConstructPayload() ([]byte, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	metricMarshaler := &pmetric.ProtoMarshaler{}

	payloadMetrics := pmetric.NewMetrics()
	for _, md := range l.buffer {
		md.ResourceMetrics().MoveAndAppendTo(payloadMetrics.ResourceMetrics())
	}

	// update the buffer to retain the current metrics which were moved to the new payload
	l.buffer = []pmetric.Metrics{payloadMetrics}

	payload, err := metricMarshaler.MarshalMetrics(payloadMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to construct payload: %w", err)
	}

	return payload, nil
}

// TraceBuffer is a buffer for ptrace.Traces
type TraceBuffer struct {
	mutex     sync.Mutex
	buffer    []ptrace.Traces
	idealSize int
}

// NewTraceBuffer creates a traceBuffer with the ideal size set
func NewTraceBuffer(idealSize int) *TraceBuffer {
	return &TraceBuffer{
		buffer:    make([]ptrace.Traces, 0),
		idealSize: idealSize,
	}
}

// Len counts the number of spans in all Traces payloads in buffer
func (l *TraceBuffer) Len() int {
	size := 0
	for _, td := range l.buffer {
		size += td.SpanCount()
	}

	return size
}

// Add adds the new trace payload and adjust buffer to keep ideal size
func (l *TraceBuffer) Add(td ptrace.Traces) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	traceSize := td.SpanCount()
	bufferSize := l.Len()
	switch {
	// The number of traces is more than idealSize so reset this to just this trace set
	case traceSize > l.idealSize:
		l.buffer = []ptrace.Traces{td}

	// Haven't reached idealSize yet so add this
	case traceSize+bufferSize < l.idealSize:
		l.buffer = append(l.buffer, td)

	// Adding this will put us over idealSize so and add the new traces.
	// Only remove the oldest if it does not bring buffer under idealSize
	case traceSize+bufferSize >= l.idealSize:
		l.buffer = append(l.buffer, td)

		// Remove items from the buffer until we find one that if we remove it will put us under the ideal size
		for {
			newBufferSize := l.Len()
			oldest := l.buffer[0]

			// If removing this one will put us under ideal size then break
			if newBufferSize-oldest.SpanCount() < l.idealSize {
				break
			}

			// Remove the oldest
			l.buffer = l.buffer[1:]
		}
	}
}

// ConstructPayload condenses the buffer and serializes to protobuf
func (l *TraceBuffer) ConstructPayload() ([]byte, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	traceMarshaler := &ptrace.ProtoMarshaler{}

	payloadTraces := ptrace.NewTraces()
	for _, md := range l.buffer {
		md.ResourceSpans().MoveAndAppendTo(payloadTraces.ResourceSpans())
	}

	// update the buffer to retain the current traces which were moved to the new payload
	l.buffer = []ptrace.Traces{payloadTraces}

	payload, err := traceMarshaler.MarshalTraces(payloadTraces)
	if err != nil {
		return nil, fmt.Errorf("failed to construct payload: %w", err)
	}

	return payload, nil
}
