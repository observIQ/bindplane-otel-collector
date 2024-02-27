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

// Package metrics defines how to send the different pmetric types to Snowflake
package metrics // "github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/metrics"

import (
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func generateTestMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	// resource
	resource := metrics.ResourceMetrics().AppendEmpty()
	resource.SetSchemaUrl("resource_test_metrics")
	resource.Resource().Attributes().FromRaw(map[string]any{"a1": "resource_attributes"})
	resource.Resource().SetDroppedAttributesCount(1)

	//scope
	scope := resource.ScopeMetrics().AppendEmpty()
	scope.SetSchemaUrl("scope_test_metrics")
	scope.Scope().SetName("unit_test_scope_metrics")
	scope.Scope().SetVersion("v0")
	scope.Scope().SetDroppedAttributesCount(1)
	scope.Scope().Attributes().FromRaw(map[string]any{"a1": "scope_attributes", "parent": "resource"})

	// exponential histogram metrics
	ehmMetrics := scope.Metrics().AppendEmpty()
	ehmMetrics.SetName("exponential histogram metrics")
	ehmMetrics.SetDescription("eh metrics for unit tests")
	ehmMetrics.SetUnit("m/s")
	ehm := ehmMetrics.SetEmptyExponentialHistogram()
	ehm.SetAggregationTemporality(pmetric.AggregationTemporalityUnspecified)
	for i := 0; i < 3; i++ {
		dp := ehm.DataPoints().AppendEmpty()

		dp.SetCount(uint64(i))
		dp.SetSum(float64(i) + 0.01)
		dp.SetScale(int32(i + 1))
		dp.SetZeroCount(uint64(i + 2))
		dp.SetZeroThreshold(float64(i) + 2.1)
		dp.SetFlags(pmetric.DataPointFlags(i))
		dp.SetMax(float64(i) + 3.2)
		dp.SetMin(float64(i) + 2.3)
		dp.Positive().SetOffset(int32(i))
		dp.Positive().BucketCounts().FromRaw([]uint64{uint64(i), 1, 2, 3, 4})
		dp.Negative().SetOffset(int32(i + 1))
		dp.Negative().BucketCounts().FromRaw([]uint64{5, 6, 7, 8, uint64(i)})

		// exemplars
		generateExemplars(dp.Exemplars())
	}

	// gauge metrics
	gaugeMetrics := scope.Metrics().AppendEmpty()
	gaugeMetrics.SetName("gauge metrics")
	gaugeMetrics.SetDescription("gauge metrics for unit tests")
	gaugeMetrics.SetUnit("N")
	gauge := gaugeMetrics.SetEmptyGauge()
	for i := 0; i < 4; i++ {
		dp := gauge.DataPoints().AppendEmpty()

		dp.SetDoubleValue(float64(i) + 1.23)
		dp.SetFlags(pmetric.DataPointFlags(i))
		dp.Attributes().FromRaw(map[string]any{"a1": i, "a2": "gauge attributes"})

		//exemplars
		generateExemplars(dp.Exemplars())
	}

	// histogram metrics
	histogramMetrics := scope.Metrics().AppendEmpty()
	histogramMetrics.SetName("histogram metrics")
	histogramMetrics.SetDescription("histogram metrics for unit tests")
	histogramMetrics.SetUnit("mi/h")
	histogram := histogramMetrics.SetEmptyHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	for i := 0; i < 2; i++ {
		dp := histogram.DataPoints().AppendEmpty()

		dp.SetCount(uint64(i) + 3)
		dp.SetFlags(pmetric.DataPointFlags(i))
		dp.SetMax(float64(i) + 3.4)
		dp.SetMin(float64(i) + 2.1)
		dp.SetSum(float64(i) + 0.2)
		dp.BucketCounts().FromRaw([]uint64{1, 3, 0, 4})
		dp.ExplicitBounds().FromRaw([]float64{0.3, 4.1, 2.01, float64(i) + 1.1})

		// exemplars
		generateExemplars(dp.Exemplars())
	}

	// sum metrics
	sumMetrics := scope.Metrics().AppendEmpty()
	sumMetrics.SetName("sum metrics")
	sumMetrics.SetDescription("sum metrics for unit tests")
	sumMetrics.SetUnit("mL")
	sum := sumMetrics.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
	for i := 0; i < 3; i++ {
		dp := sum.DataPoints().AppendEmpty()

		dp.SetDoubleValue(float64(i) + 1.13)
		dp.SetFlags(pmetric.DataPointFlags(i))
		dp.Attributes().FromRaw(map[string]any{"a1": i, "a2": "sum attributes"})

		// exemplars
		generateExemplars(dp.Exemplars())
	}

	// summary metrics
	summaryMetrics := scope.Metrics().AppendEmpty()
	summaryMetrics.SetName("summary metrics")
	summaryMetrics.SetDescription("summary metrics for unit tests")
	summaryMetrics.SetUnit("m^2")
	summary := summaryMetrics.SetEmptySummary()
	for i := 0; i < 2; i++ {
		dp := summary.DataPoints().AppendEmpty()

		dp.SetCount(uint64(i) + 1)
		dp.SetFlags(pmetric.DataPointFlags(i))
		dp.SetSum(float64(i) + 2.03)

		qv := dp.QuantileValues().AppendEmpty()
		qv.SetQuantile(float64(i))
		qv.SetValue(float64(i) + 1.7)
	}

	return metrics
}

func generateExemplars(es pmetric.ExemplarSlice) {
	for i := 0; i < 2; i++ {
		e := es.AppendEmpty()
		e.SetDoubleValue(float64(i) + 2.1)
		e.FilteredAttributes().FromRaw(map[string]any{"a1": "exemplar attribute", "a2": i})
	}

	e := es.AppendEmpty()
	e.SetIntValue(3)
}
