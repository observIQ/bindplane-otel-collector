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

package removeemptyvalueprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type emptyValueProcessor struct {
	logger *zap.Logger
	c      Config
}

func newEmptyValueProcessor(logger *zap.Logger, cfg Config) *emptyValueProcessor {
	return &emptyValueProcessor{
		logger: logger,
		c:      cfg,
	}
}

func (sp *emptyValueProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	resourceSpans := td.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		resourceSpan := resourceSpans.At(i)
		scopeSpans := resourceSpan.ScopeSpans()

		if sp.c.EnableResourceAttributes {
			cleanMap(resourceSpan.Resource().Attributes(), sp.c)
		}

		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()

			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				if sp.c.EnableAttributes {
					cleanMap(span.Attributes(), sp.c)
				}
			}
		}
	}

	return td, nil
}

func (sp *emptyValueProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	resourceLogs := ld.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		resourceLog := resourceLogs.At(i)
		scopeLogs := resourceLog.ScopeLogs()

		if sp.c.EnableResourceAttributes {
			cleanMap(resourceLog.Resource().Attributes(), sp.c)
		}

		for j := 0; j < scopeLogs.Len(); j++ {
			scopeLog := scopeLogs.At(j)
			logRecords := scopeLog.LogRecords()

			for k := 0; k < logRecords.Len(); k++ {
				logRecord := logRecords.At(k)
				if sp.c.EnableAttributes {
					cleanMap(logRecord.Attributes(), sp.c)
				}

				if sp.c.EnableLogBody {
					cleanLogBody(logRecord, sp.c)
				}
			}
		}
	}

	return ld, nil
}

func (sp *emptyValueProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	resourceMetrics := md.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		resourceMetric := resourceMetrics.At(i)
		scopeMetrics := resourceMetric.ScopeMetrics()

		if sp.c.EnableResourceAttributes {
			cleanMap(resourceMetric.Resource().Attributes(), sp.c)
		}

		for j := 0; j < scopeMetrics.Len(); j++ {
			scopeMetric := scopeMetrics.At(j)
			metrics := scopeMetric.Metrics()

			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				if sp.c.EnableAttributes {
					cleanMetricAttrs(metric, sp.c)
				}
			}
		}
	}
	return md, nil
}

func cleanMap(m pcommon.Map, c Config) {
	m.RemoveIf(func(s string, v pcommon.Value) bool {
		switch v.Type() {
		case pcommon.ValueTypeEmpty:
			return c.RemoveNulls
		case pcommon.ValueTypeMap:
			subMap := v.Map()
			cleanMap(subMap, c)
			return subMap.Len() == 0 && c.RemoveEmptyMaps
		case pcommon.ValueTypeSlice:
			s := v.Slice()
			cleanSlice(s, c)
			return s.Len() == 0 && c.RemoveEmptyLists
		case pcommon.ValueTypeStr:
			str := v.Str()
			return shouldFilterString(str, c)
		}

		return false
	})
}

func cleanSlice(s pcommon.Slice, c Config) {
	s.RemoveIf(func(v pcommon.Value) bool {
		switch v.Type() {
		case pcommon.ValueTypeEmpty:
			return c.RemoveNulls
		case pcommon.ValueTypeMap:
			subMap := v.Map()
			cleanMap(subMap, c)
			return subMap.Len() == 0 && c.RemoveEmptyMaps
		case pcommon.ValueTypeSlice:
			s := v.Slice()
			cleanSlice(s, c)
			return s.Len() == 0 && c.RemoveEmptyLists
		case pcommon.ValueTypeStr:
			str := v.Str()
			return shouldFilterString(str, c)
		}

		return false
	})
}

func shouldFilterString(s string, c Config) bool {
	for _, filteredString := range c.EmptyStringValues {
		if s == filteredString {
			return true
		}
	}

	return false
}

func cleanMetricAttrs(metric pmetric.Metric, c Config) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c)
		}

	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c)
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c)
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c)
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c)
		}
	default:
		// skip metric if None or unknown type
	}
}

func cleanLogBody(lr plog.LogRecord, c Config) {
	body := lr.Body()
	switch body.Type() {
	case pcommon.ValueTypeMap:
		bodyMap := body.Map()
		cleanMap(bodyMap, c)
		if bodyMap.Len() == 0 && c.RemoveEmptyMaps {
			pcommon.NewValueEmpty().CopyTo(body)
		}
	case pcommon.ValueTypeSlice:
		bodySlice := body.Slice()
		cleanSlice(bodySlice, c)
		if bodySlice.Len() == 0 && c.RemoveEmptyLists {
			pcommon.NewValueEmpty().CopyTo(body)
		}
	case pcommon.ValueTypeStr:
		if shouldFilterString(body.Str(), c) {
			pcommon.NewValueEmpty().CopyTo(body)
		}
	}
}
