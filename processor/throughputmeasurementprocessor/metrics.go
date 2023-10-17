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

package throughputmeasurementprocessor

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const tagProcessorKey = "processor"

var (
	processorTagKey = tag.MustNewKey(tagProcessorKey)
	logDataSize     = stats.Int64("log_data_size", "Size of the log package passed to the processor", stats.UnitBytes)
	metricDataSize  = stats.Int64("metric_data_size", "Size of the metric package passed to the processor", stats.UnitBytes)
	traceDataSize   = stats.Int64("trace_data_size", "Size of the trace package passed to the processor", stats.UnitBytes)
	logCount        = stats.Int64("log_count", "Count of the number log records passed to the processor", stats.UnitDimensionless)
	metricCount     = stats.Int64("metric_count", "Count of the number metric data points passed to the processor", stats.UnitDimensionless)
	traceCount      = stats.Int64("trace_count", "Count of the number trace spans passed to the processor", stats.UnitDimensionless)
)

func metricViews() []*view.View {
	processorTagKeys := []tag.Key{processorTagKey}

	return []*view.View{
		{
			Name:        processorhelper.BuildCustomMetricName(string(typeStr), logDataSize.Name()),
			Description: logDataSize.Description(),
			Measure:     logDataSize,
			TagKeys:     processorTagKeys,
			Aggregation: view.Sum(),
		},
		{
			Name:        processorhelper.BuildCustomMetricName(string(typeStr), metricDataSize.Name()),
			Description: metricDataSize.Description(),
			Measure:     metricDataSize,
			TagKeys:     processorTagKeys,
			Aggregation: view.Sum(),
		},
		{
			Name:        processorhelper.BuildCustomMetricName(string(typeStr), traceDataSize.Name()),
			Description: traceDataSize.Description(),
			Measure:     traceDataSize,
			TagKeys:     processorTagKeys,
			Aggregation: view.Sum(),
		},
		{
			Name:        processorhelper.BuildCustomMetricName(string(typeStr), logCount.Name()),
			Description: logCount.Description(),
			Measure:     logCount,
			TagKeys:     processorTagKeys,
			Aggregation: view.Sum(),
		},
		{
			Name:        processorhelper.BuildCustomMetricName(string(typeStr), metricCount.Name()),
			Description: metricCount.Description(),
			Measure:     metricCount,
			TagKeys:     processorTagKeys,
			Aggregation: view.Sum(),
		},
		{
			Name:        processorhelper.BuildCustomMetricName(string(typeStr), traceCount.Name()),
			Description: traceCount.Description(),
			Measure:     traceCount,
			TagKeys:     processorTagKeys,
			Aggregation: view.Sum(),
		},
	}
}
