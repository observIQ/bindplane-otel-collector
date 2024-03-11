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

package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// getCurrentTime is a variable that holds the current time function. It is used to mock time in tests.
var getCurrentTime = func() time.Time { return time.Now().UTC() }

// otlpGenerator is a replay generator for logs, metrics, and traces.
// It generates a stream of telemetry based on the json embedded in the configuration,
// each record identical save for the timestamp.
type otlpGenerator struct {
	cfg           GeneratorConfig
	logger        *zap.Logger
	telemetryType component.DataType
	logs          plog.Logs
	logsStart     time.Time
	metrics       pmetric.Metrics
	metricsStart  time.Time
	traces        ptrace.Traces
	tracesStart   time.Time
}

func newOtlpGenerator(cfg GeneratorConfig, logger *zap.Logger) *otlpGenerator {
	lg := &otlpGenerator{
		cfg:     cfg,
		logger:  logger,
		logs:    plog.NewLogs(),
		metrics: pmetric.NewMetrics(),
		traces:  ptrace.NewTraces(),
	}

	// validation already proves this exists, is a string, and a component.DataType
	lg.telemetryType = component.Type(lg.cfg.AdditionalConfig["telemetry_type"].(string))
	jsonBytes := []byte(lg.cfg.AdditionalConfig["otlp_json"].(string))

	var err error
	switch lg.telemetryType {
	case component.DataTypeLogs:
		marshaler := plog.JSONUnmarshaler{}
		lg.logs, err = marshaler.UnmarshalLogs(jsonBytes)
		// validation should catch this error
		if err != nil {
			logger.Warn("error unmarshalling otlp logs json", zap.Error(err))
		}
		lg.adjustLogTimes()
	case component.DataTypeMetrics:
		marshaler := pmetric.JSONUnmarshaler{}
		lg.metrics, err = marshaler.UnmarshalMetrics(jsonBytes)
		// validation should catch this error
		if err != nil {
			logger.Warn("error unmarshalling otlp metrics json", zap.Error(err))
		}
		lg.adjustMetricTimes()
	case component.DataTypeTraces:
		marshaler := ptrace.JSONUnmarshaler{}
		lg.traces, err = marshaler.UnmarshalTraces(jsonBytes)
		// validation should catch this error
		if err != nil {
			logger.Warn("error unmarshalling otlp traces json", zap.Error(err))
		}
		lg.adjustTraceTimes()
	}

	return lg
}

// logRecordUpdater is a function that updates the timestamp of a log record
// since we need to iterate through the logs and metrics in three different places
type logRecordUpdater func(dp plog.LogRecord)

// generic function to update the timestamps of all log records using the provided updater,
// or in the case of findLastLogTime, to find the most recent timestamp.
func (g *otlpGenerator) updateLogTimes(updater logRecordUpdater) {
	for i := 0; i < g.logs.ResourceLogs().Len(); i++ {
		resourceLogs := g.logs.ResourceLogs().At(i)
		for k := 0; k < resourceLogs.ScopeLogs().Len(); k++ {
			scopeLogs := resourceLogs.ScopeLogs().At(k)
			for j := 0; j < scopeLogs.LogRecords().Len(); j++ {
				log := scopeLogs.LogRecords().At(j)
				updater(log)
			}
		}
	}
}

// findLastLogTime finds the log with the most recent timestamp
func (g *otlpGenerator) findLastLogTime() time.Time {
	maxTime := time.Time{}
	g.updateLogTimes(func(log plog.LogRecord) {
		if t := log.Timestamp().AsTime(); t.After(maxTime) {
			maxTime = t
		}
	})
	return maxTime
}

// adjustTraceTimes changes the log timestamp to be relative to the current time, placing
// the log with timestamp maxTime at the current time.
func (g *otlpGenerator) adjustLogTimes() {
	now := getCurrentTime()
	maxTime := g.findLastLogTime()

	g.updateLogTimes(func(log plog.LogRecord) {
		delta := maxTime.Sub(log.Timestamp().AsTime())
		log.SetTimestamp(pcommon.NewTimestampFromTime(now.Add(-delta)))
	})
	g.logsStart = now
}

// generateLogs generates a new set of logs with updated timestamps, adding the time
// since the last set of logs was generated to the timestamps.
func (g *otlpGenerator) generateLogs() plog.Logs {
	now := getCurrentTime()
	timeSince := now.Sub(g.logsStart)
	g.updateLogTimes(func(log plog.LogRecord) {
		timeStamp := log.Timestamp().AsTime().Add(timeSince)
		log.SetTimestamp(pcommon.NewTimestampFromTime(timeStamp))
	})
	g.logsStart = now
	return g.logs
}

// timeStamped is convenience interface for updating the timestamps of metric datapoints in a generic way,
// since all metric datapoint types duck-type to this interface.
type timeStamped interface {
	Timestamp() pcommon.Timestamp
	SetTimestamp(pcommon.Timestamp)
}

type dataPointUpdater func(dp timeStamped)

// updateMetricTimes updates the timestamps of all metrics using the provided updater
func (g *otlpGenerator) updateMetricTimes(updater dataPointUpdater) {
	for i := 0; i < g.metrics.ResourceMetrics().Len(); i++ {
		resourceMetrics := g.metrics.ResourceMetrics().At(i)
		for k := 0; k < resourceMetrics.ScopeMetrics().Len(); k++ {
			scopeMetrics := resourceMetrics.ScopeMetrics().At(k)
			for j := 0; j < scopeMetrics.Metrics().Len(); j++ {
				metric := scopeMetrics.Metrics().At(j)
				switch metric.Type() {
				case pmetric.MetricTypeSum:
					dps := metric.Sum().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						updater(dps.At(l))
					}
				case pmetric.MetricTypeGauge:
					dps := metric.Gauge().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						updater(dps.At(l))
					}
				case pmetric.MetricTypeSummary:
					dps := metric.Summary().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						updater(dps.At(l))
					}
				case pmetric.MetricTypeHistogram:
					dps := metric.Histogram().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						updater(dps.At(l))
					}
				case pmetric.MetricTypeExponentialHistogram:
					dps := metric.ExponentialHistogram().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						updater(dps.At(l))
					}
				}
			}
		}
	}
}

// findLastMetricTime finds the datapoint with the most recent timestamp
func (g *otlpGenerator) findLastMetricTime() time.Time {
	maxTime := time.Time{}
	g.updateMetricTimes(func(ts timeStamped) {
		if t := ts.Timestamp().AsTime(); t.After(maxTime) {
			maxTime = t
		}
	})
	return maxTime
}

// adjustMetricTimes changes the metric timestamps to be relative to the current time, placing
// the metric with timestamp maxTime at the current time.
func (g *otlpGenerator) adjustMetricTimes() {
	now := getCurrentTime()
	maxTime := g.findLastMetricTime()
	g.updateMetricTimes(func(ts timeStamped) {
		delta := maxTime.Sub(ts.Timestamp().AsTime())
		ts.SetTimestamp(pcommon.NewTimestampFromTime(now.Add(-delta)))
	})
	g.metricsStart = now
}

// generateMetrics generates a new set of metrics with updated timestamps, adding the time
// since the last set of metrics was generated to the timestamps.
func (g *otlpGenerator) generateMetrics() pmetric.Metrics {
	// calculate the time since the last baseline time we used to adjust the metrics
	now := getCurrentTime()
	timeSince := now.Sub(g.metricsStart)
	g.updateMetricTimes(func(ts timeStamped) {
		timeStamp := ts.Timestamp().AsTime().Add(timeSince)
		ts.SetTimestamp(pcommon.NewTimestampFromTime(timeStamp))
	})
	// update the baseline time to the current time
	g.metricsStart = now

	return g.metrics
}

// Since we need to iterate through the traces in three different places, we define a function type
// to update the timestamps of spans.
type spanUpdater func(dp ptrace.Span)

// updateTraceTimes updates the timestamps of all spans using the provided updater
func (g *otlpGenerator) updateTraceTimes(updater spanUpdater) {
	for i := 0; i < g.traces.ResourceSpans().Len(); i++ {
		resourceSpans := g.traces.ResourceSpans().At(i)
		for k := 0; k < resourceSpans.ScopeSpans().Len(); k++ {
			scopeSpans := resourceSpans.ScopeSpans().At(k)
			for j := 0; j < scopeSpans.Spans().Len(); j++ {
				span := scopeSpans.Spans().At(j)
				updater(span)
			}
		}
	}
}

// findLastTraceEndTime finds the span with the last end time
func (g *otlpGenerator) findLastTraceEndTime() time.Time {
	maxTime := time.Time{}

	g.updateTraceTimes(func(span ptrace.Span) {
		end := span.EndTimestamp().AsTime()
		if end.After(maxTime) {
			maxTime = end
		}
	})

	return maxTime
}

// adjustTraceTimes changes the start and end times of all spans to be relative to the current time, placing
// the span that ends at maxTime at the current time.
func (g *otlpGenerator) adjustTraceTimes() {
	now := getCurrentTime()
	maxTime := g.findLastTraceEndTime()

	g.updateTraceTimes(func(span ptrace.Span) {
		// delta is the duration in the past this span's end time is before the maxTime
		delta := maxTime.Sub(span.EndTimestamp().AsTime())
		// spanDuration is the length of the span
		spanDuration := span.EndTimestamp().AsTime().Sub(span.StartTimestamp().AsTime())
		// move each span's end time by delta
		endTime := now.Add(-delta)
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
		// set the start time to be the end time minus the original span duration
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(endTime.Add(-spanDuration)))
	})

	// save the current time we used as a baseline to adjust the spans
	g.tracesStart = now
}

func (g *otlpGenerator) generateTraces() ptrace.Traces {
	// calculate the time since the last baseline time we used to adjust the spans
	now := getCurrentTime()
	timeSince := now.Sub(g.tracesStart)
	// add the time since to the start and end times of all spans
	g.updateTraceTimes(func(span ptrace.Span) {
		startTime := span.StartTimestamp().AsTime().Add(timeSince)
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(startTime))

		endTime := span.EndTimestamp().AsTime().Add(timeSince)
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
	})

	// update the baseline time to the current time
	g.tracesStart = now

	return g.traces
}
