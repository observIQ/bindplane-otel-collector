// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// extractAttributesFromTraces extracts specified attribute keys from traces.
// It checks span attributes, resource attributes, and scope attributes in that order.
// Returns an attribute.Set for use as a map key.
func extractAttributesFromTraces(td ptrace.Traces, attrKeys []string) attribute.Set {
	var attrs []attribute.KeyValue
	
	// We need to extract attributes from the first span we find
	// Note: This assumes all spans in the batch should have the same grouping attributes
	if td.ResourceSpans().Len() > 0 {
		rs := td.ResourceSpans().At(0)
		resourceAttrs := rs.Resource().Attributes()
		
		if rs.ScopeSpans().Len() > 0 {
			ss := rs.ScopeSpans().At(0)
			scopeAttrs := ss.Scope().Attributes()
			
			if ss.Spans().Len() > 0 {
				spanAttrs := ss.Spans().At(0).Attributes()
				
				// Extract each requested attribute key
				for _, key := range attrKeys {
					attrs = append(attrs, extractAttribute(key, spanAttrs, scopeAttrs, resourceAttrs))
				}
			}
		}
	}
	
	// If no spans found, return attributes with empty values
	if len(attrs) == 0 {
		for _, key := range attrKeys {
			attrs = append(attrs, attribute.String(key, ""))
		}
	}
	
	return attribute.NewSet(attrs...)
}

// extractAttributesFromLogs extracts specified attribute keys from logs.
// It checks log record attributes, resource attributes, and scope attributes in that order.
func extractAttributesFromLogs(ld plog.Logs, attrKeys []string) attribute.Set {
	var attrs []attribute.KeyValue
	
	if ld.ResourceLogs().Len() > 0 {
		rl := ld.ResourceLogs().At(0)
		resourceAttrs := rl.Resource().Attributes()
		
		if rl.ScopeLogs().Len() > 0 {
			sl := rl.ScopeLogs().At(0)
			scopeAttrs := sl.Scope().Attributes()
			
			if sl.LogRecords().Len() > 0 {
				logAttrs := sl.LogRecords().At(0).Attributes()
				
				for _, key := range attrKeys {
					attrs = append(attrs, extractAttribute(key, logAttrs, scopeAttrs, resourceAttrs))
				}
			}
		}
	}
	
	if len(attrs) == 0 {
		for _, key := range attrKeys {
			attrs = append(attrs, attribute.String(key, ""))
		}
	}
	
	return attribute.NewSet(attrs...)
}

// extractAttributesFromMetrics extracts specified attribute keys from metrics.
// It checks datapoint attributes, resource attributes, and scope attributes in that order.
func extractAttributesFromMetrics(md pmetric.Metrics, attrKeys []string) attribute.Set {
	var attrs []attribute.KeyValue
	
	if md.ResourceMetrics().Len() > 0 {
		rm := md.ResourceMetrics().At(0)
		resourceAttrs := rm.Resource().Attributes()
		
		if rm.ScopeMetrics().Len() > 0 {
			sm := rm.ScopeMetrics().At(0)
			scopeAttrs := sm.Scope().Attributes()
			
			// Try to extract from the first metric's first datapoint
			if sm.Metrics().Len() > 0 {
				metric := sm.Metrics().At(0)
				var datapointAttrs pcommon.Map
				
				// Handle different metric types
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					if metric.Gauge().DataPoints().Len() > 0 {
						datapointAttrs = metric.Gauge().DataPoints().At(0).Attributes()
					}
				case pmetric.MetricTypeSum:
					if metric.Sum().DataPoints().Len() > 0 {
						datapointAttrs = metric.Sum().DataPoints().At(0).Attributes()
					}
				case pmetric.MetricTypeHistogram:
					if metric.Histogram().DataPoints().Len() > 0 {
						datapointAttrs = metric.Histogram().DataPoints().At(0).Attributes()
					}
				case pmetric.MetricTypeExponentialHistogram:
					if metric.ExponentialHistogram().DataPoints().Len() > 0 {
						datapointAttrs = metric.ExponentialHistogram().DataPoints().At(0).Attributes()
					}
				case pmetric.MetricTypeSummary:
					if metric.Summary().DataPoints().Len() > 0 {
						datapointAttrs = metric.Summary().DataPoints().At(0).Attributes()
					}
				}
				
				for _, key := range attrKeys {
					attrs = append(attrs, extractAttribute(key, datapointAttrs, scopeAttrs, resourceAttrs))
				}
			}
		}
	}
	
	if len(attrs) == 0 {
		for _, key := range attrKeys {
			attrs = append(attrs, attribute.String(key, ""))
		}
	}
	
	return attribute.NewSet(attrs...)
}

// extractAttribute extracts a single attribute from multiple attribute maps.
// It checks the maps in order and returns the first match, or an empty string if not found.
func extractAttribute(key string, attrMaps ...pcommon.Map) attribute.KeyValue {
	for _, attrMap := range attrMaps {
		if attrMap.Len() == 0 {
			continue
		}
		
		val, exists := attrMap.Get(key)
		if exists {
			return pcommonValueToOtelAttribute(key, val)
		}
	}
	
	// Not found in any map, return empty string
	return attribute.String(key, "")
}

// pcommonValueToOtelAttribute converts a pcommon.Value to an attribute.KeyValue
func pcommonValueToOtelAttribute(key string, val pcommon.Value) attribute.KeyValue {
	switch val.Type() {
	case pcommon.ValueTypeStr:
		return attribute.String(key, val.Str())
	case pcommon.ValueTypeBool:
		return attribute.Bool(key, val.Bool())
	case pcommon.ValueTypeInt:
		return attribute.Int64(key, val.Int())
	case pcommon.ValueTypeDouble:
		return attribute.Float64(key, val.Double())
	case pcommon.ValueTypeBytes:
		return attribute.String(key, string(val.Bytes().AsRaw()))
	default:
		// For complex types (Map, Slice), convert to string representation
		return attribute.String(key, val.AsString())
	}
}
