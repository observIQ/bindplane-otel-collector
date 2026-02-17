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

package snapshot

import (
	"math/rand/v2"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// logPosition is a tuple of the resource index, scope index, and log index of a log record
type logPosition struct {
	resourceIdx int
	scopeIdx    int
	logIdx      int
}

// generateLogPositions generates a list of log positions for all log records in the given logs
func generateLogPositions(logs plog.Logs) []logPosition {
	positions := make([]logPosition, 0)
	for ri := 0; ri < logs.ResourceLogs().Len(); ri++ {
		for si := 0; si < logs.ResourceLogs().At(ri).ScopeLogs().Len(); si++ {
			for li := 0; li < logs.ResourceLogs().At(ri).ScopeLogs().At(si).LogRecords().Len(); li++ {
				positions = append(positions, logPosition{resourceIdx: ri, scopeIdx: si, logIdx: li})
			}
		}
	}
	return positions
}

// randomSampleLogs samples a given percentage of log records from the given logs based on the given positions
// Uses rand.Shuffle to randomize the positions array before sampling to ensure a random sample.
// Returns a new logs object with only the sampled log records.
func randomSampleLogs(originalLogs plog.Logs, positions []logPosition, retentionPercent int) plog.Logs {
	// If retention is 100%, return the original logs
	// If retention is 0%, set to 1%
	switch retentionPercent {
	case 100:
		return originalLogs
	case 0:
		retentionPercent = 1
	}

	// Randomize the positions array to ensure a random sample
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	// Calculate the number of log records to keep based on the retention percentage
	targetCount := (originalLogs.LogRecordCount() * retentionPercent) / 100

	// Create a map of the positions to keep
	keepPositions := make(map[logPosition]bool, targetCount)
	for i := 0; i < targetCount; i++ {
		keepPositions[positions[i]] = true
	}

	// Rebuild the logs with only the kept log records without modifying the original logs
	result := plog.NewLogs()
	for ri := 0; ri < originalLogs.ResourceLogs().Len(); ri++ {
		srcRL := originalLogs.ResourceLogs().At(ri)
		var dstRL plog.ResourceLogs
		rlCreated := false

		for si := 0; si < srcRL.ScopeLogs().Len(); si++ {
			srcSL := srcRL.ScopeLogs().At(si)
			var dstSL plog.ScopeLogs
			slCreated := false

			for li := 0; li < srcSL.LogRecords().Len(); li++ {
				if keepPositions[logPosition{resourceIdx: ri, scopeIdx: si, logIdx: li}] {
					if !rlCreated {
						dstRL = result.ResourceLogs().AppendEmpty()
						srcRL.Resource().CopyTo(dstRL.Resource())
						rlCreated = true
					}
					if !slCreated {
						dstSL = dstRL.ScopeLogs().AppendEmpty()
						srcSL.Scope().CopyTo(dstSL.Scope())
						slCreated = true
					}
					srcSL.LogRecords().At(li).CopyTo(dstSL.LogRecords().AppendEmpty())
				}
			}
		}
	}

	return result
}

// dataPointPosition is a tuple identifying a specific data point in the metrics structure
type dataPointPosition struct {
	resourceIdx  int
	scopeIdx     int
	metricIdx    int
	dataPointIdx int
}

// generateDataPointPositions generates a list of data point positions for all data points in the given metrics
func generateDataPointPositions(metrics pmetric.Metrics) []dataPointPosition {
	positions := make([]dataPointPosition, 0)
	for ri := 0; ri < metrics.ResourceMetrics().Len(); ri++ {
		for si := 0; si < metrics.ResourceMetrics().At(ri).ScopeMetrics().Len(); si++ {
			for mi := 0; mi < metrics.ResourceMetrics().At(ri).ScopeMetrics().At(si).Metrics().Len(); mi++ {
				metric := metrics.ResourceMetrics().At(ri).ScopeMetrics().At(si).Metrics().At(mi)
				dpCount := getDataPointCount(metric)
				for di := 0; di < dpCount; di++ {
					positions = append(positions, dataPointPosition{resourceIdx: ri, scopeIdx: si, metricIdx: mi, dataPointIdx: di})
				}
			}
		}
	}
	return positions
}

// getDataPointCount returns the number of data points in a metric based on its type
func getDataPointCount(metric pmetric.Metric) int {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		return metric.Gauge().DataPoints().Len()
	case pmetric.MetricTypeSum:
		return metric.Sum().DataPoints().Len()
	case pmetric.MetricTypeHistogram:
		return metric.Histogram().DataPoints().Len()
	case pmetric.MetricTypeExponentialHistogram:
		return metric.ExponentialHistogram().DataPoints().Len()
	case pmetric.MetricTypeSummary:
		return metric.Summary().DataPoints().Len()
	default:
		return 0
	}
}

// randomSampleMetrics samples a given percentage of data points from the given metrics based on the given positions
// Uses rand.Shuffle to randomize the positions array before sampling to ensure a random sample.
// Returns a new metrics object with only the sampled data points.
func randomSampleMetrics(originalMetrics pmetric.Metrics, positions []dataPointPosition, retentionPercent int) pmetric.Metrics {
	// If retention is 100%, return the original metrics
	// If retention is 0%, set to 1%
	switch retentionPercent {
	case 100:
		return originalMetrics
	case 0:
		retentionPercent = 1
	}

	// Randomize the positions array to ensure a random sample
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	// Calculate the number of data points to keep based on the retention percentage
	targetCount := (originalMetrics.DataPointCount() * retentionPercent) / 100

	// Create a map of the positions to keep
	keepPositions := make(map[dataPointPosition]bool, targetCount)
	for i := 0; i < targetCount; i++ {
		keepPositions[positions[i]] = true
	}

	// Rebuild the metrics with only the kept data points without modifying the original metrics
	result := pmetric.NewMetrics()
	for ri := 0; ri < originalMetrics.ResourceMetrics().Len(); ri++ {
		srcRM := originalMetrics.ResourceMetrics().At(ri)
		var dstRM pmetric.ResourceMetrics
		rmCreated := false

		for si := 0; si < srcRM.ScopeMetrics().Len(); si++ {
			srcSM := srcRM.ScopeMetrics().At(si)
			var dstSM pmetric.ScopeMetrics
			smCreated := false

			for mi := 0; mi < srcSM.Metrics().Len(); mi++ {
				srcMetric := srcSM.Metrics().At(mi)
				var dstMetric pmetric.Metric
				metricCreated := false

				dpCount := getDataPointCount(srcMetric)
				for di := 0; di < dpCount; di++ {
					if keepPositions[dataPointPosition{resourceIdx: ri, scopeIdx: si, metricIdx: mi, dataPointIdx: di}] {
						if !rmCreated {
							dstRM = result.ResourceMetrics().AppendEmpty()
							srcRM.Resource().CopyTo(dstRM.Resource())
							rmCreated = true
						}
						if !smCreated {
							dstSM = dstRM.ScopeMetrics().AppendEmpty()
							srcSM.Scope().CopyTo(dstSM.Scope())
							smCreated = true
						}
						if !metricCreated {
							dstMetric = dstSM.Metrics().AppendEmpty()
							dstMetric.SetName(srcMetric.Name())
							dstMetric.SetDescription(srcMetric.Description())
							dstMetric.SetUnit(srcMetric.Unit())
							initMetricDataPoints(srcMetric, dstMetric)
							metricCreated = true
						}
						copyDataPoint(srcMetric, dstMetric, di)
					}
				}
			}
		}
	}

	return result
}

// initMetricDataPoints initializes the destination metric with the same type as the source
func initMetricDataPoints(src, dst pmetric.Metric) {
	switch src.Type() {
	case pmetric.MetricTypeGauge:
		dst.SetEmptyGauge()
	case pmetric.MetricTypeSum:
		dst.SetEmptySum()
		dst.Sum().SetIsMonotonic(src.Sum().IsMonotonic())
		dst.Sum().SetAggregationTemporality(src.Sum().AggregationTemporality())
	case pmetric.MetricTypeHistogram:
		dst.SetEmptyHistogram()
		dst.Histogram().SetAggregationTemporality(src.Histogram().AggregationTemporality())
	case pmetric.MetricTypeExponentialHistogram:
		dst.SetEmptyExponentialHistogram()
		dst.ExponentialHistogram().SetAggregationTemporality(src.ExponentialHistogram().AggregationTemporality())
	case pmetric.MetricTypeSummary:
		dst.SetEmptySummary()
	}
}

// copyDataPoint copies a specific data point from src to dst metric
func copyDataPoint(src, dst pmetric.Metric, idx int) {
	switch src.Type() {
	case pmetric.MetricTypeGauge:
		src.Gauge().DataPoints().At(idx).CopyTo(dst.Gauge().DataPoints().AppendEmpty())
	case pmetric.MetricTypeSum:
		src.Sum().DataPoints().At(idx).CopyTo(dst.Sum().DataPoints().AppendEmpty())
	case pmetric.MetricTypeHistogram:
		src.Histogram().DataPoints().At(idx).CopyTo(dst.Histogram().DataPoints().AppendEmpty())
	case pmetric.MetricTypeExponentialHistogram:
		src.ExponentialHistogram().DataPoints().At(idx).CopyTo(dst.ExponentialHistogram().DataPoints().AppendEmpty())
	case pmetric.MetricTypeSummary:
		src.Summary().DataPoints().At(idx).CopyTo(dst.Summary().DataPoints().AppendEmpty())
	}
}

// spanPosition is a tuple identifying a specific span in the traces structure
type spanPosition struct {
	resourceIdx int
	scopeIdx    int
	spanIdx     int
}

// generateSpanPositions generates a list of span positions for all spans in the given traces
func generateSpanPositions(traces ptrace.Traces) []spanPosition {
	positions := make([]spanPosition, 0)
	for ri := 0; ri < traces.ResourceSpans().Len(); ri++ {
		for si := 0; si < traces.ResourceSpans().At(ri).ScopeSpans().Len(); si++ {
			for spi := 0; spi < traces.ResourceSpans().At(ri).ScopeSpans().At(si).Spans().Len(); spi++ {
				positions = append(positions, spanPosition{resourceIdx: ri, scopeIdx: si, spanIdx: spi})
			}
		}
	}
	return positions
}

// randomSampleTraces samples a given percentage of spans from the given traces based on the given positions
// Uses rand.Shuffle to randomize the positions array before sampling to ensure a random sample.
// Returns a new traces object with only the sampled spans.
func randomSampleTraces(originalTraces ptrace.Traces, positions []spanPosition, retentionPercent int) ptrace.Traces {
	// If retention is 100%, return the original traces
	// If retention is 0%, set to 1%
	switch retentionPercent {
	case 100:
		return originalTraces
	case 0:
		retentionPercent = 1
	}

	// Randomize the positions array to ensure a random sample
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	// Calculate the number of spans to keep based on the retention percentage
	targetCount := (originalTraces.SpanCount() * retentionPercent) / 100

	// Create a map of the positions to keep
	keepPositions := make(map[spanPosition]bool, targetCount)
	for i := 0; i < targetCount; i++ {
		keepPositions[positions[i]] = true
	}

	// Rebuild the traces with only the kept spans without modifying the original traces
	result := ptrace.NewTraces()
	for ri := 0; ri < originalTraces.ResourceSpans().Len(); ri++ {
		srcRS := originalTraces.ResourceSpans().At(ri)
		var dstRS ptrace.ResourceSpans
		rsCreated := false

		for si := 0; si < srcRS.ScopeSpans().Len(); si++ {
			srcSS := srcRS.ScopeSpans().At(si)
			var dstSS ptrace.ScopeSpans
			ssCreated := false

			for spi := 0; spi < srcSS.Spans().Len(); spi++ {
				if keepPositions[spanPosition{resourceIdx: ri, scopeIdx: si, spanIdx: spi}] {
					if !rsCreated {
						dstRS = result.ResourceSpans().AppendEmpty()
						srcRS.Resource().CopyTo(dstRS.Resource())
						rsCreated = true
					}
					if !ssCreated {
						dstSS = dstRS.ScopeSpans().AppendEmpty()
						srcSS.Scope().CopyTo(dstSS.Scope())
						ssCreated = true
					}
					srcSS.Spans().At(spi).CopyTo(dstSS.Spans().AppendEmpty())
				}
			}
		}
	}

	return result
}
