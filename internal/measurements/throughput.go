// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package measurements provides code to help manage throughput measurements for BindPlane and
// the throughput measurement processor.
package measurements

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ThroughputMeasurementsRegistry represents a registry for the throughputmeasurement processor to
// register their ThroughputMeasurements.
type ThroughputMeasurementsRegistry interface {
	// RegisterThroughputMeasurements registers the measurements for the given processor.
	// It should return an error if the processor has already been registered.
	RegisterThroughputMeasurements(processorID string, measurements *ThroughputMeasurements) error
}

// ThroughputMeasurements represents all captured throughput metrics.
// It allows for incrementing and querying the current values of throughtput metrics
type ThroughputMeasurements struct {
	logSize, metricSize, traceSize      *int64Counter
	logCount, datapointCount, spanCount *int64Counter
	attributes                          attribute.Set
}

// NewThroughputMeasurements initializes a new ThroughputMeasurements, adding metrics for the measurements to the meter provider.
func NewThroughputMeasurements(mp metric.MeterProvider, processorID string, extraAttributes map[string]string) (*ThroughputMeasurements, error) {
	meter := mp.Meter("github.com/observiq/bindplane-otel-collector/internal/measurements")

	logSize, err := meter.Int64Counter(
		metricName("log_data_size"),
		metric.WithDescription("Size of the log package passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create log_data_size counter: %w", err)
	}

	metricSize, err := meter.Int64Counter(
		metricName("metric_data_size"),
		metric.WithDescription("Size of the metric package passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric_data_size counter: %w", err)
	}

	traceSize, err := meter.Int64Counter(
		metricName("trace_data_size"),
		metric.WithDescription("Size of the trace package passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace_data_size counter: %w", err)
	}

	logCount, err := meter.Int64Counter(
		metricName("log_count"),
		metric.WithDescription("Count of the number log records passed to the processor"),
		metric.WithUnit("{logs}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create log_count counter: %w", err)
	}

	datapointCount, err := meter.Int64Counter(
		metricName("metric_count"),
		metric.WithDescription("Count of the number datapoints passed to the processor"),
		metric.WithUnit("{datapoints}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric_count counter: %w", err)
	}

	spanCount, err := meter.Int64Counter(
		metricName("trace_count"),
		metric.WithDescription("Count of the number spans passed to the processor"),
		metric.WithUnit("{spans}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace_count counter: %w", err)
	}

	attrs := createMeasurementsAttributeSet(processorID, extraAttributes)

	return &ThroughputMeasurements{
		logSize:        newInt64Counter(logSize, attrs),
		logCount:       newInt64Counter(logCount, attrs),
		metricSize:     newInt64Counter(metricSize, attrs),
		datapointCount: newInt64Counter(datapointCount, attrs),
		traceSize:      newInt64Counter(traceSize, attrs),
		spanCount:      newInt64Counter(spanCount, attrs),
		attributes:     attrs,
	}, nil
}

// AddLogs records throughput metrics for the provided logs.
func (tm *ThroughputMeasurements) AddLogs(ctx context.Context, l plog.Logs) {
	sizer := plog.ProtoMarshaler{}

	tm.logSize.Add(ctx, int64(sizer.LogsSize(l)))
	tm.logCount.Add(ctx, int64(l.LogRecordCount()))
}

// AddMetrics records throughput metrics for the provided metrics.
func (tm *ThroughputMeasurements) AddMetrics(ctx context.Context, m pmetric.Metrics) {
	sizer := pmetric.ProtoMarshaler{}

	tm.metricSize.Add(ctx, int64(sizer.MetricsSize(m)))
	tm.datapointCount.Add(ctx, int64(m.DataPointCount()))
}

// AddTraces records throughput metrics for the provided traces.
func (tm *ThroughputMeasurements) AddTraces(ctx context.Context, t ptrace.Traces) {
	sizer := ptrace.ProtoMarshaler{}

	tm.traceSize.Add(ctx, int64(sizer.TracesSize(t)))
	tm.spanCount.Add(ctx, int64(t.SpanCount()))
}

// LogSize returns the total size in bytes of all log payloads added to this ThroughputMeasurements.
func (tm *ThroughputMeasurements) LogSize() int64 {
	return tm.logSize.Val()
}

// MetricSize returns the total size in bytes of all metric payloads added to this ThroughputMeasurements.
func (tm *ThroughputMeasurements) MetricSize() int64 {
	return tm.metricSize.Val()
}

// TraceSize returns the total size in bytes of all trace payloads added to this ThroughputMeasurements.
func (tm *ThroughputMeasurements) TraceSize() int64 {
	return tm.traceSize.Val()
}

// LogCount return the total number of log records that have been added to this ThroughputMeasurements.
func (tm *ThroughputMeasurements) LogCount() int64 {
	return tm.logCount.Val()
}

// DatapointCount return the total number of datapoints that have been added to this ThroughputMeasurements.
func (tm *ThroughputMeasurements) DatapointCount() int64 {
	return tm.datapointCount.Val()
}

// SpanCount return the total number of spans that have been added to this ThroughputMeasurements.
func (tm *ThroughputMeasurements) SpanCount() int64 {
	return tm.spanCount.Val()
}

// Attributes returns the full set of attributes used on each metric for this ThroughputMeasurements.
func (tm *ThroughputMeasurements) Attributes() attribute.Set {
	return tm.attributes
}

// int64Counter combines a metric.Int64Counter with a atomic.Int64 so that the value of the counter may be
// retrieved.
// The value of the metric counter and val are not guaranteed to be synchronized, but will be eventually consistent.
type int64Counter struct {
	counter    metric.Int64Counter
	val        atomic.Int64
	attributes attribute.Set
}

func newInt64Counter(counter metric.Int64Counter, attributes attribute.Set) *int64Counter {
	return &int64Counter{
		counter:    counter,
		attributes: attributes,
	}
}

func (i *int64Counter) Add(ctx context.Context, delta int64) {
	i.counter.Add(ctx, delta, metric.WithAttributeSet(i.attributes))
	i.val.Add(delta)
}

func (i *int64Counter) Val() int64 {
	return i.val.Load()
}

func metricName(metric string) string {
	return fmt.Sprintf("otelcol_processor_throughputmeasurement_%s", metric)
}

func createMeasurementsAttributeSet(processorID string, extraAttributes map[string]string) attribute.Set {
	attrs := make([]attribute.KeyValue, 0, len(extraAttributes)+1)

	attrs = append(attrs, attribute.String("processor", processorID))
	for k, v := range extraAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	return attribute.NewSet(attrs...)
}

// ResettableThroughputMeasurementsRegistry is a concrete version of ThroughputMeasurementsRegistry that is able to be reset.
type ResettableThroughputMeasurementsRegistry struct {
	measurements     *sync.Map
	emitCountMetrics bool
}

// NewResettableThroughputMeasurementsRegistry creates a new ResettableThroughputMeasurementsRegistry
func NewResettableThroughputMeasurementsRegistry(emitCountMetrics bool) *ResettableThroughputMeasurementsRegistry {
	return &ResettableThroughputMeasurementsRegistry{
		measurements:     &sync.Map{},
		emitCountMetrics: emitCountMetrics,
	}
}

// RegisterThroughputMeasurements registers the ThroughputMeasurements with the registry.
func (ctmr *ResettableThroughputMeasurementsRegistry) RegisterThroughputMeasurements(processorID string, measurements *ThroughputMeasurements) error {
	_, alreadyExists := ctmr.measurements.LoadOrStore(processorID, measurements)
	if alreadyExists {
		return fmt.Errorf("measurements for processor %q was already registered", processorID)
	}

	return nil
}

// Reset unregisters all throughput measurements in this registry
func (ctmr *ResettableThroughputMeasurementsRegistry) Reset() {
	ctmr.measurements = &sync.Map{}
}

// OTLPMeasurements returns all the measurements in this registry as OTLP metrics.
func (ctmr *ResettableThroughputMeasurementsRegistry) OTLPMeasurements(extraAttributes map[string]string) pmetric.Metrics {
	m := pmetric.NewMetrics()
	rm := m.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	ctmr.measurements.Range(func(_, value any) bool {
		tm := value.(*ThroughputMeasurements)
		OTLPThroughputMeasurements(tm, ctmr.emitCountMetrics, extraAttributes).MoveAndAppendTo(sm.Metrics())
		return true
	})

	if m.DataPointCount() == 0 {
		// If there are no datapoints in the metric,
		// we don't want to have an empty ResourceMetrics in the metrics slice.
		return pmetric.NewMetrics()
	}

	return m
}
