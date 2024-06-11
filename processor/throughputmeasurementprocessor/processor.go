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
	"context"
	"fmt"
	"math/rand"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const processorAttributeName = "processor"

type throughputMeasurementProcessor struct {
	logger                              *zap.Logger
	enabled                             bool
	samplingCutOffRatio                 float64
	attrSet                             attribute.Set
	logSize, metricSize, traceSize      metric.Int64Counter
	logCount, datapointCount, spanCount metric.Int64Counter
	tracesSizer                         ptrace.Sizer
	metricsSizer                        pmetric.Sizer
	logsSizer                           plog.Sizer
}

func newThroughputMeasurementProcessor(logger *zap.Logger, mp metric.MeterProvider, cfg *Config, processorID string) (*throughputMeasurementProcessor, error) {
	meter := mp.Meter("github.com/observiq/bindplane-agent/processor/throughputmeasurementprocessor")

	logSize, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(componentType.String(), "log_data_size"),
		metric.WithDescription("Size of the log package passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create log_data_size counter: %w", err)
	}

	metricSize, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(componentType.String(), "metric_data_size"),
		metric.WithDescription("Size of the metric package passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric_data_size counter: %w", err)
	}

	traceSize, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(componentType.String(), "trace_data_size"),
		metric.WithDescription("Size of the trace package passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace_data_size counter: %w", err)
	}

	logCount, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(componentType.String(), "log_count"),
		metric.WithDescription("Count of the number log records passed to the processor"),
		metric.WithUnit("{logs}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create log_count counter: %w", err)
	}

	datapointCount, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(componentType.String(), "metric_count"),
		metric.WithDescription("Count of the number datapoints passed to the processor"),
		metric.WithUnit("{datapoints}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric_count counter: %w", err)
	}

	spanCount, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(componentType.String(), "trace_count"),
		metric.WithDescription("Count of the number spans passed to the processor"),
		metric.WithUnit("{spans}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace_count counter: %w", err)
	}

	return &throughputMeasurementProcessor{
		logger:              logger,
		enabled:             cfg.Enabled,
		samplingCutOffRatio: cfg.SamplingRatio,
		logSize:             logSize,
		metricSize:          metricSize,
		traceSize:           traceSize,
		logCount:            logCount,
		datapointCount:      datapointCount,
		spanCount:           spanCount,
		attrSet:             attribute.NewSet(attribute.String(processorAttributeName, processorID)),
		tracesSizer:         &ptrace.ProtoMarshaler{},
		metricsSizer:        &pmetric.ProtoMarshaler{},
		logsSizer:           &plog.ProtoMarshaler{},
	}, nil
}

func (tmp *throughputMeasurementProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if tmp.enabled {
		//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
		if rand.Float64() <= tmp.samplingCutOffRatio {
			tmp.traceSize.Add(ctx, int64(tmp.tracesSizer.TracesSize(td)), metric.WithAttributeSet(tmp.attrSet))
			tmp.spanCount.Add(ctx, int64(td.SpanCount()), metric.WithAttributeSet(tmp.attrSet))
		}
	}

	return td, nil
}

func (tmp *throughputMeasurementProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if tmp.enabled {
		//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
		if rand.Float64() <= tmp.samplingCutOffRatio {
			tmp.logSize.Add(ctx, int64(tmp.logsSizer.LogsSize(ld)), metric.WithAttributeSet(tmp.attrSet))
			tmp.logCount.Add(ctx, int64(ld.LogRecordCount()), metric.WithAttributeSet(tmp.attrSet))
		}
	}

	return ld, nil
}

func (tmp *throughputMeasurementProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if tmp.enabled {
		//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
		if rand.Float64() <= tmp.samplingCutOffRatio {
			tmp.metricSize.Add(ctx, int64(tmp.metricsSizer.MetricsSize(md)), metric.WithAttributeSet(tmp.attrSet))
			tmp.datapointCount.Add(ctx, int64(md.DataPointCount()), metric.WithAttributeSet(tmp.attrSet))
		}
	}

	return md, nil
}
