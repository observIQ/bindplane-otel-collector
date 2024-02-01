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
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// filterMetrics filters the metrics by the given query and timestamp.
// The returned payload cannot be assumed to be a copy, so it should not be modified.
func filterMetrics(m pmetric.Metrics, searchQuery *string, minimumTimestamp *time.Time) pmetric.Metrics {
	// No filters specified, filtered metrics are trivially the same as input metrics
	if searchQuery == nil && minimumTimestamp == nil {
		return m
	}

	filteredMetrics := pmetric.NewMetrics()
	resourceMetrics := m.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		filteredResourceLogs := filterResourceMetrics(resourceMetrics.At(i), searchQuery, minimumTimestamp)

		// Don't append empty resource metrics
		if filteredResourceLogs.ScopeMetrics().Len() != 0 {
			filteredResourceLogs.MoveTo(filteredMetrics.ResourceMetrics().AppendEmpty())
		}
	}

	return filteredMetrics
}

func filterResourceMetrics(rm pmetric.ResourceMetrics, searchQuery *string, minimumTimestamp *time.Time) pmetric.ResourceMetrics {
	filteredResourceMetrics := pmetric.NewResourceMetrics()

	// Copy old resource to filtered resource
	resource := rm.Resource()
	resource.CopyTo(filteredResourceMetrics.Resource())

	// Apply query to resource
	queryMatchesResource := true // default to true if no query specified
	if searchQuery != nil {
		queryMatchesResource = queryMatchesMap(resource.Attributes(), *searchQuery)
	}

	scopeMetrics := rm.ScopeMetrics()
	for i := 0; i < scopeMetrics.Len(); i++ {
		filteredScopeMetrics := filterScopeMetrics(rm.ScopeMetrics().At(i), queryMatchesResource, searchQuery, minimumTimestamp)

		// Don't append empty scope metrics
		if filteredScopeMetrics.Metrics().Len() != 0 {
			filteredScopeMetrics.MoveTo(filteredResourceMetrics.ScopeMetrics().AppendEmpty())
		}
	}

	return filteredResourceMetrics
}

func filterScopeMetrics(sm pmetric.ScopeMetrics, queryMatchesResource bool, searchQuery *string, minimumTimestamp *time.Time) pmetric.ScopeMetrics {
	filteredScopeMetrics := pmetric.NewScopeMetrics()
	metrics := sm.Metrics()
	for i := 0; i < metrics.Len(); i++ {
		m := metrics.At(i)
		filteredMetric := filterMetric(m, queryMatchesResource, searchQuery, minimumTimestamp)

		if !metricIsEmpty(filteredMetric) {
			filteredMetric.MoveTo(filteredScopeMetrics.Metrics().AppendEmpty())
		}
	}

	return filteredScopeMetrics
}

func filterMetric(m pmetric.Metric, queryMatchesResource bool, searchQuery *string, minimumTimestamp *time.Time) pmetric.Metric {
	filteredMetric := pmetric.NewMetric()
	// Copy metric to filtered metric
	filteredMetric.SetName(m.Name())
	filteredMetric.SetDescription(m.Description())
	filteredMetric.SetUnit(m.Unit())

	// Apply query to metric
	queryMatchesMetric := true // default to true if no query specified
	// Skip if we already know the query matches the resource
	if !queryMatchesResource && searchQuery != nil {
		queryMatchesMetric = metricMatchesQuery(m, *searchQuery)
	}

	switch m.Type() {
	case pmetric.MetricTypeGauge:
		filteredGauge := filterGauge(m.Gauge(), queryMatchesResource, queryMatchesMetric, searchQuery, minimumTimestamp)
		filteredGauge.MoveTo(filteredMetric.SetEmptyGauge())
	case pmetric.MetricTypeSum:
		filteredSum := filterSum(m.Sum(), queryMatchesResource, queryMatchesMetric, searchQuery, minimumTimestamp)
		filteredSum.MoveTo(filteredMetric.SetEmptySum())
	case pmetric.MetricTypeHistogram:
		filteredHistogram := filterHistogram(m.Histogram(), queryMatchesResource, queryMatchesMetric, searchQuery, minimumTimestamp)
		filteredHistogram.MoveTo(filteredMetric.SetEmptyHistogram())
	case pmetric.MetricTypeExponentialHistogram:
		filteredExponentialHistogram := filterExponentialHistogram(m.ExponentialHistogram(), queryMatchesResource, queryMatchesMetric, searchQuery, minimumTimestamp)
		filteredExponentialHistogram.MoveTo(filteredMetric.SetEmptyExponentialHistogram())
	case pmetric.MetricTypeSummary:
		filteredSummary := filterSummary(m.Summary(), queryMatchesResource, queryMatchesMetric, searchQuery, minimumTimestamp)
		filteredSummary.MoveTo(filteredMetric.SetEmptySummary())
	case pmetric.MetricTypeEmpty:
		// Ignore empty
	}

	return filteredMetric
}

func filterGauge(g pmetric.Gauge, queryMatchesResource, queryMatchesName bool, searchQuery *string, minimumTimestamp *time.Time) pmetric.Gauge {
	filteredGauge := pmetric.NewGauge()

	dps := g.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if datapointMatches(dp, queryMatchesResource, queryMatchesName, searchQuery, minimumTimestamp) {
			dp.CopyTo(filteredGauge.DataPoints().AppendEmpty())
		}
	}

	return filteredGauge
}

func filterSum(s pmetric.Sum, queryMatchesResource, queryMatchesName bool, searchQuery *string, minimumTimestamp *time.Time) pmetric.Sum {
	filteredSum := pmetric.NewSum()

	dps := s.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if datapointMatches(dp, queryMatchesResource, queryMatchesName, searchQuery, minimumTimestamp) {
			dp.CopyTo(filteredSum.DataPoints().AppendEmpty())
		}
	}

	return filteredSum
}

func filterHistogram(h pmetric.Histogram, queryMatchesResource, queryMatchesName bool, searchQuery *string, minimumTimestamp *time.Time) pmetric.Histogram {
	filteredHistogram := pmetric.NewHistogram()

	dps := h.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if datapointMatches(dp, queryMatchesResource, queryMatchesName, searchQuery, minimumTimestamp) {
			dp.CopyTo(filteredHistogram.DataPoints().AppendEmpty())
		}
	}

	return filteredHistogram
}

func filterExponentialHistogram(eh pmetric.ExponentialHistogram, queryMatchesResource, queryMatchesName bool, searchQuery *string, minimumTimestamp *time.Time) pmetric.ExponentialHistogram {
	filteredExponentialHistogram := pmetric.NewExponentialHistogram()

	dps := eh.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if datapointMatches(dp, queryMatchesResource, queryMatchesName, searchQuery, minimumTimestamp) {
			dp.CopyTo(filteredExponentialHistogram.DataPoints().AppendEmpty())
		}
	}

	return filteredExponentialHistogram
}

func filterSummary(s pmetric.Summary, queryMatchesResource, queryMatchesName bool, searchQuery *string, minimumTimestamp *time.Time) pmetric.Summary {
	filteredSummary := pmetric.NewSummary()

	dps := s.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if datapointMatches(dp, queryMatchesResource, queryMatchesName, searchQuery, minimumTimestamp) {
			dp.CopyTo(filteredSummary.DataPoints().AppendEmpty())
		}
	}

	return filteredSummary
}

func metricMatchesQuery(m pmetric.Metric, query string) bool {
	// Match query against metric name
	return strings.Contains(m.Name(), query)
}

// datapoint is an interface that every concrete datapoint type implements
type datapoint interface {
	Attributes() pcommon.Map
	Timestamp() pcommon.Timestamp
}

func datapointMatches(dp datapoint, queryMatchesResource, queryMatchesName bool, searchQuery *string, minimumTimestamp *time.Time) bool {
	queryAlreadyMatched := queryMatchesResource || queryMatchesName

	queryMatchesDatapoint := true
	if !queryAlreadyMatched && searchQuery != nil {
		queryMatchesDatapoint = datapointMatchesQuery(dp, *searchQuery)
	}

	matchesTimestamp := true
	if minimumTimestamp != nil {
		matchesTimestamp = datapointMatchesTimestamp(dp, *minimumTimestamp)
	}

	matchesQuery := queryMatchesResource || queryMatchesName || queryMatchesDatapoint

	return matchesQuery && matchesTimestamp
}

func datapointMatchesTimestamp(dp datapoint, minimumTimestamp time.Time) bool {
	return dp.Timestamp() >= pcommon.NewTimestampFromTime(minimumTimestamp)
}

func datapointMatchesQuery(dp datapoint, searchQuery string) bool {
	return queryMatchesMap(dp.Attributes(), searchQuery)
}

func metricIsEmpty(m pmetric.Metric) bool {
	switch m.Type() {
	case pmetric.MetricTypeGauge:
		return m.Gauge().DataPoints().Len() == 0
	case pmetric.MetricTypeSum:
		return m.Sum().DataPoints().Len() == 0
	case pmetric.MetricTypeHistogram:
		return m.Histogram().DataPoints().Len() == 0
	case pmetric.MetricTypeExponentialHistogram:
		return m.ExponentialHistogram().DataPoints().Len() == 0
	case pmetric.MetricTypeSummary:
		return m.Summary().DataPoints().Len() == 0
	case pmetric.MetricTypeEmpty:
		return true
	}
	return false
}
