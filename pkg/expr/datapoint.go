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

package expr

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Metric specific fields for use in expressions
const (
	MetricNameField     = "metric_name"
	DatapointValueField = "datapoint_value"
)

// Datapoint is the simplified representation of a metric datapoint.
type Datapoint = map[string]any

func convertMetricToDatapoints(metric pmetric.Metric, resource map[string]any) []Datapoint {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		datapoints := make([]Datapoint, 0, dps.Len())
		for i := 0; i < dps.Len(); i++ {
			datapoints = append(datapoints, convertNumberDatapoint(dps.At(i), resource, metric.Name()))
		}
		return datapoints
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		datapoints := make([]Datapoint, 0, dps.Len())
		for i := 0; i < dps.Len(); i++ {
			datapoints = append(datapoints, convertNumberDatapoint(dps.At(i), resource, metric.Name()))
		}
		return datapoints
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		datapoints := make([]Datapoint, 0, dps.Len())
		for i := 0; i < dps.Len(); i++ {
			datapoints = append(datapoints, convertGenericDatapoint(dps.At(i), resource, metric.Name()))
		}
		return datapoints
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		datapoints := make([]Datapoint, 0, dps.Len())
		for i := 0; i < dps.Len(); i++ {
			datapoints = append(datapoints, convertGenericDatapoint(dps.At(i), resource, metric.Name()))
		}
		return datapoints
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		datapoints := make([]Datapoint, 0, dps.Len())
		for i := 0; i < dps.Len(); i++ {
			datapoints = append(datapoints, convertGenericDatapoint(dps.At(i), resource, metric.Name()))
		}
		return datapoints
	}
	return nil
}

type genericDatapoint interface {
	Attributes() pcommon.Map
}

func convertGenericDatapoint[T genericDatapoint](dp T, resource map[string]any, name string) Datapoint {
	return Datapoint{
		ResourceField:   resource,
		AttributesField: dp.Attributes().AsRaw(),
		MetricNameField: name,
	}
}

func convertNumberDatapoint(dp pmetric.NumberDataPoint, resource map[string]any, name string) Datapoint {
	datapoint := convertGenericDatapoint(dp, resource, name)

	switch dp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		datapoint[DatapointValueField] = dp.IntValue()
	case pmetric.NumberDataPointValueTypeDouble:
		datapoint[DatapointValueField] = dp.DoubleValue()
	}

	return datapoint
}

// DatapointResourceGroup represents a pmetric.ResourceMetrics as native go types
type DatapointResourceGroup struct {
	Resource   map[string]any
	Datapoints []Datapoint
}

// ConvertToDatapointResourceGroup converts a pmetric.Metrics into a slice of DatapointResourceGroup
func ConvertToDatapointResourceGroup(metrics pmetric.Metrics) []DatapointResourceGroup {
	groups := make([]DatapointResourceGroup, 0, metrics.ResourceMetrics().Len())

	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		resourceMetrics := metrics.ResourceMetrics().At(i)
		resource := resourceMetrics.Resource().Attributes().AsRaw()
		group := DatapointResourceGroup{
			Resource:   resource,
			Datapoints: make([]Datapoint, 0, resourceMetrics.ScopeMetrics().Len()),
		}
		for j := 0; j < resourceMetrics.ScopeMetrics().Len(); j++ {
			metricSlice := resourceMetrics.ScopeMetrics().At(j).Metrics()
			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)
				group.Datapoints = append(group.Datapoints, convertMetricToDatapoints(metric, resource)...)
			}
		}
		groups = append(groups, group)
	}

	return groups
}
