// Copyright observIQ, Inc.
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

package restapireceiver

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// convertJSONToMetrics converts an array of JSON objects to pmetric.Metrics.
// Each JSON object becomes one metric data point.
func convertJSONToMetrics(data []map[string]any, cfg *MetricsConfig, logger *zap.Logger) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, item := range data {
		// Extract numeric value from the item
		value, valueKey := extractNumericValue(item)
		if value == nil {
			logger.Debug("skipping item without numeric value", zap.Any("item", item))
			continue
		}

		// Extract metric name (defaults to "restapi.metric")
		metricName := "restapi.metric"
		if cfg.NameField != "" {
			if nameVal, ok := item[cfg.NameField]; ok {
				if nameStr, ok := nameVal.(string); ok && nameStr != "" {
					metricName = nameStr
				}
			}
		}

		// Extract metric description (defaults to "Metric from REST API")
		metricDescription := "Metric from REST API"
		if cfg.DescriptionField != "" {
			if descVal, ok := item[cfg.DescriptionField]; ok {
				if descStr, ok := descVal.(string); ok && descStr != "" {
					metricDescription = descStr
				}
			}
		}

		// Extract metric type (defaults to "gauge")
		metricType := "gauge"
		if cfg.TypeField != "" {
			if typeVal, ok := item[cfg.TypeField]; ok {
				if typeStr, ok := typeVal.(string); ok && typeStr != "" {
					metricType = typeStr
				}
			}
		}

		// Extract metric unit (defaults to empty string)
		metricUnit := ""
		if cfg.UnitField != "" {
			if unitVal, ok := item[cfg.UnitField]; ok {
				if unitStr, ok := unitVal.(string); ok && unitStr != "" {
					metricUnit = unitStr
				}
			}
		}

		// Create a new metric
		metric := scopeMetrics.Metrics().AppendEmpty()
		metric.SetName(metricName)
		metric.SetDescription(metricDescription)
		if metricUnit != "" {
			metric.SetUnit(metricUnit)
		}

		// Create metric data point based on type
		var dataPointAttrs pcommon.Map
		switch metricType {
		case "sum":
			sum := metric.SetEmptySum()
			sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
			sum.SetIsMonotonic(false) // Can be configured if needed
			dataPoint := sum.DataPoints().AppendEmpty()
			dataPoint.SetDoubleValue(*value)
			dataPointAttrs = dataPoint.Attributes()

			// Set timestamp
			timestamp := extractTimestamp(item)
			if timestamp > 0 {
				dataPoint.SetTimestamp(timestamp)
			} else {
				dataPoint.SetTimestamp(now)
			}
		case "histogram":
			histogram := metric.SetEmptyHistogram()
			histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
			dataPoint := histogram.DataPoints().AppendEmpty()
			dataPoint.SetCount(1)
			dataPoint.SetSum(*value)
			dataPointAttrs = dataPoint.Attributes()

			// Set timestamp
			timestamp := extractTimestamp(item)
			if timestamp > 0 {
				dataPoint.SetTimestamp(timestamp)
			} else {
				dataPoint.SetTimestamp(now)
			}
		case "summary":
			summary := metric.SetEmptySummary()
			dataPoint := summary.DataPoints().AppendEmpty()
			dataPoint.SetCount(1)
			dataPoint.SetSum(*value)
			dataPointAttrs = dataPoint.Attributes()

			// Set timestamp
			timestamp := extractTimestamp(item)
			if timestamp > 0 {
				dataPoint.SetTimestamp(timestamp)
			} else {
				dataPoint.SetTimestamp(now)
			}
		default: // "gauge" or any unknown type defaults to gauge
			gauge := metric.SetEmptyGauge()
			dataPoint := gauge.DataPoints().AppendEmpty()
			dataPoint.SetDoubleValue(*value)
			dataPointAttrs = dataPoint.Attributes()

			// Set timestamp
			timestamp := extractTimestamp(item)
			if timestamp > 0 {
				dataPoint.SetTimestamp(timestamp)
			} else {
				dataPoint.SetTimestamp(now)
			}
		}

		// Set attributes from all non-value, non-metric-metadata fields
		for key, val := range item {
			// Skip the value field itself
			if key == valueKey {
				continue
			}
			// Skip timestamp fields (already used)
			if isTimestampField(key) {
				continue
			}
			// Skip metric metadata fields
			if cfg.NameField != "" && key == cfg.NameField {
				continue
			}
			if cfg.DescriptionField != "" && key == cfg.DescriptionField {
				continue
			}
			if cfg.TypeField != "" && key == cfg.TypeField {
				continue
			}
			if cfg.UnitField != "" && key == cfg.UnitField {
				continue
			}
			// Add as attribute
			setAttribute(dataPointAttrs, key, val)
		}
	}

	return metrics
}

// extractNumericValue extracts a numeric value from the JSON object.
// It looks for common field names like "value", "count", "total", etc.
// Returns the value and the key it was found under.
func extractNumericValue(item map[string]any) (*float64, string) {
	// Common value field names (in order of preference)
	valueFields := []string{"value", "count", "total", "sum", "amount", "metric", "measurement"}

	// First, try common field names
	for _, fieldName := range valueFields {
		if val, exists := item[fieldName]; exists {
			if value := toFloat64(val); value != nil {
				return value, fieldName
			}
		}
	}

	// If no common field found, look for any numeric field
	for key, val := range item {
		if value := toFloat64(val); value != nil {
			return value, key
		}
	}

	return nil, ""
}

// toFloat64 converts various numeric types to float64.
func toFloat64(val any) *float64 {
	switch v := val.(type) {
	case float64:
		return &v
	case float32:
		f := float64(v)
		return &f
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case int32:
		f := float64(v)
		return &f
	case uint:
		f := float64(v)
		return &f
	case uint64:
		f := float64(v)
		return &f
	case uint32:
		f := float64(v)
		return &f
	}
	return nil
}

// setAttribute sets an attribute value, handling various types.
func setAttribute(attrs pcommon.Map, key string, val any) {
	switch v := val.(type) {
	case string:
		attrs.PutStr(key, v)
	case bool:
		attrs.PutBool(key, v)
	case int, int64, int32, uint, uint64, uint32:
		if f := toFloat64(v); f != nil {
			attrs.PutDouble(key, *f)
		}
	case float64, float32:
		if f := toFloat64(v); f != nil {
			attrs.PutDouble(key, *f)
		}
	case map[string]any:
		// For nested maps, convert to string representation or skip
		// In a more sophisticated implementation, we could flatten these
		attrs.PutStr(key, "nested_map")
	case []any:
		// For arrays, convert to string representation or skip
		attrs.PutStr(key, "array")
	default:
		// For other types, try to convert to string
		attrs.PutStr(key, "unknown_type")
	}
}

// isTimestampField checks if a field name is likely a timestamp field.
func isTimestampField(fieldName string) bool {
	timestampFields := []string{"timestamp", "time", "created_at", "createdAt", "date", "datetime", "@timestamp"}
	for _, tf := range timestampFields {
		if fieldName == tf {
			return true
		}
	}
	return false
}
