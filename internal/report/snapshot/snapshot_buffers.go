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

// ConstructPayload condenses the buffer and serializes to protobuf. Does not compress the payload to be compatible with both the snapshot reporter and the snapshot processor.
// It ensures that the payload is less than the maximum payload size returning an error if it cannot sample logs within the maximum payload size.
// Uses an increasing retention approach to sample the logs, starting at 1% and increasing by 25% until we reach the maximum payload size allowed.
// Clears the buffer if it cannot sample logs within the maximum payload size. This should allow the next snapshot to have a valid payload size.
func (l *LogBuffer) ConstructPayload(logsMarshaler plog.Marshaler, searchQuery *string, minimumTimestamp *time.Time, maximumPayloadSize int) ([]byte, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	payloadLogs := plog.NewLogs()
	for _, ld := range l.buffer {
		ld.ResourceLogs().MoveAndAppendTo(payloadLogs.ResourceLogs())
	}

	// update the buffer to retain the current logs which were moved to the new payload
	l.buffer = []plog.Logs{payloadLogs}

	// Filter the payload
	filteredPayload := filterLogs(payloadLogs, searchQuery, minimumTimestamp)

	// Generate the positions of the log records in the payload
	logPositions := generateLogPositions(filteredPayload)

	var lastError error
	lastPayload := []byte{}

	// Try marshaling & compressing with decreasing retention: 1%, 25%, 50%, 75%, 100%
	for retentionPercent := 0; retentionPercent <= 100; retentionPercent += 25 {
		// Sample the logs based on the positions and the retention percentage
		logsToMarshal := randomSampleLogs(filteredPayload, logPositions, retentionPercent)

		payload, err := logsMarshaler.MarshalLogs(logsToMarshal)
		if err != nil {
			lastError = fmt.Errorf("failed to construct payload: %w", err)
			break
		}

		// Compress and check size
		compressedPayload, err := compress(payload)
		if err != nil {
			lastError = fmt.Errorf("failed to compress payload: %w", err)
			break
		}

		if len(compressedPayload) > maximumPayloadSize {
			lastError = fmt.Errorf("snapshot buffer is too large to construct payload")
			break
		}

		// If under max size, set the lastPayload and attempt the next retention percentage
		lastPayload = payload
	}

	// If we found a payload that is under the maximum payload size return it regardless of the lastError
	if len(lastPayload) > 0 {
		return lastPayload, nil
	}

	// Encountered an error or we've tried all retentions and still can't fit the payload
	// so clear the buffer and return the last error seen
	l.buffer = []plog.Logs{}
	return nil, lastError
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

// ConstructPayload condenses the buffer and serializes to protobuf. Does not compress the payload to be compatible with both the snapshot reporter and the snapshot processor.
// It ensures that the payload is less than the maximum payload size returning an error if it cannot sample metrics within the maximum payload size.
// Uses an increasing retention approach to sample the metrics, starting at 1% and increasing by 25% until we reach the maximum payload size allowed.
// Clears the buffer if it cannot sample metrics within the maximum payload size. This should allow the next snapshot to have a valid payload size.
func (l *MetricBuffer) ConstructPayload(metricMarshaler pmetric.Marshaler, searchQuery *string, minimumTimestamp *time.Time, maximumPayloadSize int) ([]byte, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	payloadMetrics := pmetric.NewMetrics()
	for _, md := range l.buffer {
		md.ResourceMetrics().MoveAndAppendTo(payloadMetrics.ResourceMetrics())
	}

	// update the buffer to retain the current metrics which were moved to the new payload
	l.buffer = []pmetric.Metrics{payloadMetrics}

	// filter the payload
	filteredPayload := filterMetrics(payloadMetrics, searchQuery, minimumTimestamp)

	// Generate the positions of the data points in the payload
	dataPointPositions := generateDataPointPositions(filteredPayload)

	var lastError error
	lastPayload := []byte{}

	// Try marshaling & compressing with decreasing retention: 1%, 25%, 50%, 75%, 100%
	for retentionPercent := 0; retentionPercent <= 100; retentionPercent += 25 {
		// Sample the metrics based on the positions and the retention percentage
		metricsToMarshal := randomSampleMetrics(filteredPayload, dataPointPositions, retentionPercent)

		payload, err := metricMarshaler.MarshalMetrics(metricsToMarshal)
		if err != nil {
			lastError = fmt.Errorf("failed to construct payload: %w", err)
			break
		}

		// Compress and check size
		compressedPayload, err := compress(payload)
		if err != nil {
			lastError = fmt.Errorf("failed to compress payload: %w", err)
			break
		}

		if len(compressedPayload) > maximumPayloadSize {
			lastError = fmt.Errorf("snapshot buffer is too large to construct payload")
			break
		}

		// If under max size, set the lastPayload and attempt the next retention percentage
		lastPayload = payload
	}

	// If we found a payload that is under the maximum payload size return it regardless of the lastError
	if len(lastPayload) > 0 {
		return lastPayload, nil
	}

	// Encountered an error or we've tried all retentions and still can't fit the payload
	// so clear the buffer and return the last error seen
	l.buffer = []pmetric.Metrics{}
	return nil, lastError
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

// ConstructPayload condenses the buffer and serializes to protobuf. Does not compress the payload to be compatible with both the snapshot reporter and the snapshot processor.
// It ensures that the payload is less than the maximum payload size returning an error if it cannot sample traces within the maximum payload size.
// Uses an increasing retention approach to sample the traces, starting at 1% and increasing by 25% until we reach the maximum payload size allowed.
// Clears the buffer if it cannot sample traces within the maximum payload size. This should allow the next snapshot to have a valid payload size.
func (l *TraceBuffer) ConstructPayload(traceMarshaler ptrace.Marshaler, searchQuery *string, minimumTimestamp *time.Time, maximumPayloadSize int) ([]byte, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	payloadTraces := ptrace.NewTraces()
	for _, md := range l.buffer {
		md.ResourceSpans().MoveAndAppendTo(payloadTraces.ResourceSpans())
	}

	// update the buffer to retain the current traces which were moved to the new payload
	l.buffer = []ptrace.Traces{payloadTraces}

	// Filter the payload
	filteredPayload := filterTraces(payloadTraces, searchQuery, minimumTimestamp)

	// Generate the positions of the spans in the payload
	spanPositions := generateSpanPositions(filteredPayload)

	var lastError error
	lastPayload := []byte{}

	// Try marshaling & compressing with decreasing retention: 1%, 25%, 50%, 75%, 100%
	for retentionPercent := 0; retentionPercent <= 100; retentionPercent += 25 {
		// Sample the traces based on the positions and the retention percentage
		tracesToMarshal := randomSampleTraces(filteredPayload, spanPositions, retentionPercent)

		payload, err := traceMarshaler.MarshalTraces(tracesToMarshal)
		if err != nil {
			lastError = fmt.Errorf("failed to construct payload: %w", err)
			break
		}

		// Compress and check size
		compressedPayload, err := compress(payload)
		if err != nil {
			lastError = fmt.Errorf("failed to compress payload: %w", err)
			break
		}

		if len(compressedPayload) > maximumPayloadSize {
			lastError = fmt.Errorf("snapshot buffer is too large to construct payload")
			break
		}

		// If under max size, set the lastPayload and attempt the next retention percentage
		lastPayload = payload
	}

	// If we found a payload that is under the maximum payload size return it regardless of the lastError
	if len(lastPayload) > 0 {
		return lastPayload, nil
	}

	// Encountered an error or we've tried all retentions and still can't fit the payload
	// so clear the buffer and return the last error seen
	l.buffer = []ptrace.Traces{}
	return nil, lastError
}

// compress gzip compresses the data
func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
