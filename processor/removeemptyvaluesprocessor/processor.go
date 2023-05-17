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

package removeemptyvaluesprocessor

import (
	"context"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type emptyValueProcessor struct {
	logger                 *zap.Logger
	c                      Config
	excludeResourceKeySet  map[string]struct{}
	excludeAttributeKeySet map[string]struct{}
	excludeBodyKeySet      map[string]struct{}
}

func newEmptyValueProcessor(logger *zap.Logger, cfg Config) *emptyValueProcessor {
	var (
		excludeResourceKeySet  = make(map[string]struct{})
		excludeAttributeKeySet = make(map[string]struct{})
		excludeBodyKeySet      = make(map[string]struct{})
	)

	for _, mapKey := range cfg.ExcludeKeys {
		switch mapKey.field {
		case attributesField:
			excludeAttributeKeySet[mapKey.key] = struct{}{}
		case resourceField:
			excludeResourceKeySet[mapKey.key] = struct{}{}
		case bodyField:
			excludeBodyKeySet[mapKey.key] = struct{}{}
		}
	}

	return &emptyValueProcessor{
		logger:                 logger,
		c:                      cfg,
		excludeResourceKeySet:  excludeResourceKeySet,
		excludeAttributeKeySet: excludeAttributeKeySet,
		excludeBodyKeySet:      excludeBodyKeySet,
	}
}

func (evp *emptyValueProcessor) SkipResourceAttributes() bool {
	// If only the field is specified, but no trailing key, the whole resource should be skipped
	_, ok := evp.excludeResourceKeySet[""]
	return ok
}

func (evp *emptyValueProcessor) SkipAttributes() bool {
	// If only the field is specified, but no trailing key, the whole attributes map should be skipped
	_, ok := evp.excludeAttributeKeySet[""]
	return ok
}

func (evp *emptyValueProcessor) SkipBody() bool {
	// If only the field is specified, but no trailing key, the whole body should be skipped
	_, ok := evp.excludeBodyKeySet[""]
	return ok
}

func (evp *emptyValueProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	resourceSpans := td.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		resourceSpan := resourceSpans.At(i)
		scopeSpans := resourceSpan.ScopeSpans()

		if !evp.SkipResourceAttributes() {
			cleanMap(resourceSpan.Resource().Attributes(), evp.c, evp.excludeResourceKeySet)
		}

		if evp.SkipAttributes() {
			// Skip loops for attributes if we don't need to clean them.
			continue
		}

		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()

			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				cleanMap(span.Attributes(), evp.c, evp.excludeAttributeKeySet)
			}
		}
	}

	return td, nil
}

func (evp *emptyValueProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	resourceLogs := ld.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		resourceLog := resourceLogs.At(i)
		scopeLogs := resourceLog.ScopeLogs()

		if !evp.SkipResourceAttributes() {
			cleanMap(resourceLog.Resource().Attributes(), evp.c, evp.excludeResourceKeySet)
		}

		for j := 0; j < scopeLogs.Len(); j++ {
			scopeLog := scopeLogs.At(j)
			logRecords := scopeLog.LogRecords()

			for k := 0; k < logRecords.Len(); k++ {
				logRecord := logRecords.At(k)
				if !evp.SkipAttributes() {
					cleanMap(logRecord.Attributes(), evp.c, evp.excludeAttributeKeySet)
				}

				if !evp.SkipBody() {
					cleanLogBody(logRecord, evp.c, evp.excludeBodyKeySet)
				}
			}
		}
	}

	return ld, nil
}

func (evp *emptyValueProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	resourceMetrics := md.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		resourceMetric := resourceMetrics.At(i)
		scopeMetrics := resourceMetric.ScopeMetrics()

		if !evp.SkipResourceAttributes() {
			cleanMap(resourceMetric.Resource().Attributes(), evp.c, evp.excludeResourceKeySet)
		}

		if evp.SkipAttributes() {
			// Skip loops for attributes if we don't need to clean them.
			continue
		}

		for j := 0; j < scopeMetrics.Len(); j++ {
			scopeMetric := scopeMetrics.At(j)
			metrics := scopeMetric.Metrics()

			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				cleanMetricAttrs(metric, evp.c, evp.excludeAttributeKeySet)
			}
		}
	}
	return md, nil
}

// cleanMap removes empty values from the map, as defined by the config.
func cleanMap(m pcommon.Map, c Config, excludeKeys map[string]struct{}) {
	m.RemoveIf(func(s string, v pcommon.Value) bool {
		if _, ok := excludeKeys[s]; ok {
			return false
		}

		switch v.Type() {
		case pcommon.ValueTypeEmpty:
			return c.RemoveNulls
		case pcommon.ValueTypeMap:
			subMap := v.Map()
			cleanMap(subMap, c, trimMapKeyPrefix(s, excludeKeys))
			return c.RemoveEmptyMaps && subMap.Len() == 0
		case pcommon.ValueTypeSlice:
			return c.RemoveEmptyLists && v.Slice().Len() == 0
		case pcommon.ValueTypeStr:
			return shouldFilterString(v.Str(), c.EmptyStringValues)
		}

		return false
	})
}

// trimMapKeyPrefix returns the provided keys with the specified prefix removed.
// Any keys that don't have the prefix are removed from the returned list.
func trimMapKeyPrefix(prefix string, keySet map[string]struct{}) map[string]struct{} {
	outKeys := make(map[string]struct{}, len(keySet))
	for key := range keySet {
		// TODO: use strings.CutPrefix when we update to go1.20
		trimmedKey := strings.TrimPrefix(key, prefix+".")
		if len(trimmedKey) == len(key) {
			// the original key was left untrimmed, so this must not have the prefix
			continue
		}

		outKeys[trimmedKey] = struct{}{}
	}

	return outKeys
}

// shouldFilterString returns true if the given string should be considered an "empty" value,
// according to the config.
func shouldFilterString(s string, emptyValues []string) bool {
	for _, filteredString := range emptyValues {
		if strings.EqualFold(s, filteredString) {
			return true
		}
	}

	return false
}

// cleanMetricAttrs removes any attributes that should be considered empty from all the datapoints in the metrics.
func cleanMetricAttrs(metric pmetric.Metric, c Config, keys map[string]struct{}) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c, keys)
		}

	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c, keys)
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c, keys)
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c, keys)
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			cleanMap(dp.Attributes(), c, keys)
		}
	default:
		// skip metric if None or unknown type
	}
}

// cleanLogBody removes empty values from the log body.
func cleanLogBody(lr plog.LogRecord, c Config, keys map[string]struct{}) {
	body := lr.Body()
	switch body.Type() {
	case pcommon.ValueTypeMap:
		bodyMap := body.Map()
		cleanMap(bodyMap, c, keys)
		if c.RemoveEmptyMaps && bodyMap.Len() == 0 {
			pcommon.NewValueEmpty().CopyTo(body)
		}
	case pcommon.ValueTypeSlice:
		bodySlice := body.Slice()
		if c.RemoveEmptyLists && bodySlice.Len() == 0 {
			pcommon.NewValueEmpty().CopyTo(body)
		}
	case pcommon.ValueTypeStr:
		if shouldFilterString(body.Str(), c.EmptyStringValues) {
			pcommon.NewValueEmpty().CopyTo(body)
		}
	}
}
