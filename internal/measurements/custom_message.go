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

package measurements

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// ReportMeasurementsV1Capability is the capability needed to report measurements to bindplane
	ReportMeasurementsV1Capability = "com.bindplane.measurements.v1"
	// ReportMeasurementsType is the type for reporting measurements to BindPlane
	ReportMeasurementsType = "reportMeasurements"
)

// OTLPThroughputMeasurements converts a single ThroughputMeasurements to a pmetric.MetricSlice
func OTLPThroughputMeasurements(tm *ThroughputMeasurements, includeCountMetrics bool, extraAttributes map[string]string) pmetric.MetricSlice {
	s := pmetric.NewMetricSlice()

	attrs := pcommon.NewMap()
	sdkAttrs := tm.Attributes()
	attrIter := sdkAttrs.Iter()

	for attrIter.Next() {
		kv := attrIter.Attribute()
		switch kv.Value.Type() {
		case attribute.STRING:
			attrs.PutStr(string(kv.Key), kv.Value.AsString())
		default: // Do nothing for non-string attributes; Attributes for throughput metrics can only be strings for now.
		}
	}

	for k, v := range extraAttributes {
		attrs.PutStr(k, v)
	}

	ts := pcommon.NewTimestampFromTime(time.Now())

	addOTLPSum(s, "otelcol_processor_throughputmeasurement_log_data_size", tm.LogSize(), attrs, ts)
	addOTLPSum(s, "otelcol_processor_throughputmeasurement_metric_data_size", tm.MetricSize(), attrs, ts)
	addOTLPSum(s, "otelcol_processor_throughputmeasurement_trace_data_size", tm.TraceSize(), attrs, ts)

	if includeCountMetrics {
		addOTLPSum(s, "otelcol_processor_throughputmeasurement_log_count", tm.LogCount(), attrs, ts)
		addOTLPSum(s, "otelcol_processor_throughputmeasurement_metric_count", tm.DatapointCount(), attrs, ts)
		addOTLPSum(s, "otelcol_processor_throughputmeasurement_trace_count", tm.TraceSize(), attrs, ts)
	}

	return s
}

func addOTLPSum(ms pmetric.MetricSlice, name string, value int64, attrs pcommon.Map, now pcommon.Timestamp) {
	if value == 0 {
		// Ignore value if it's 0
		return
	}
	m := ms.AppendEmpty()

	m.SetName(name)
	m.SetEmptySum()
	s := m.Sum()

	dp := s.DataPoints().AppendEmpty()
	dp.SetIntValue(value)
	attrs.CopyTo(dp.Attributes())
	dp.SetTimestamp(now)
}
