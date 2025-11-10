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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestConvertJSONToMetrics_SimpleArray(t *testing.T) {
	data := []map[string]any{
		{"value": 42.5, "name": "metric1"},
		{"value": 100.0, "name": "metric2"},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	require.Equal(t, 1, metrics.ResourceMetrics().At(0).ScopeMetrics().Len())

	// Should have 2 metric data points
	scopeMetrics := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0)
	require.Equal(t, 2, scopeMetrics.Metrics().Len())

	// Check first metric
	metric1 := scopeMetrics.Metrics().At(0)
	require.Equal(t, "restapi.metric", metric1.Name())
	require.Equal(t, pmetric.MetricTypeGauge, metric1.Type())

	gauge := metric1.Gauge()
	require.Equal(t, 1, gauge.DataPoints().Len())
	dp1 := gauge.DataPoints().At(0)
	require.Equal(t, 42.5, dp1.DoubleValue())

	// Check attributes
	attrs1 := dp1.Attributes()
	require.Equal(t, "metric1", attrs1.AsRaw()["name"])

	// Check second metric
	metric2 := scopeMetrics.Metrics().At(1)
	require.Equal(t, "restapi.metric", metric2.Name())
	gauge2 := metric2.Gauge()
	require.Equal(t, 1, gauge2.DataPoints().Len())
	dp2 := gauge2.DataPoints().At(0)
	require.Equal(t, 100.0, dp2.DoubleValue())
}

func TestConvertJSONToMetrics_EmptyArray(t *testing.T) {
	data := []map[string]any{}
	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	require.Equal(t, 1, metrics.ResourceMetrics().At(0).ScopeMetrics().Len())
	require.Equal(t, 0, metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())
}

func TestConvertJSONToMetrics_WithNumericValue(t *testing.T) {
	data := []map[string]any{
		{"value": 42, "unit": "bytes"},
		{"value": 99.99, "unit": "percent"},
		{"value": int64(1000), "unit": "count"},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	require.Equal(t, 3, metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())

	// Check integer value
	metric1 := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	gauge1 := metric1.Gauge()
	dp1 := gauge1.DataPoints().At(0)
	require.Equal(t, 42.0, dp1.DoubleValue())
	require.Equal(t, "bytes", dp1.Attributes().AsRaw()["unit"])

	// Check float value
	metric2 := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	gauge2 := metric2.Gauge()
	dp2 := gauge2.DataPoints().At(0)
	require.Equal(t, 99.99, dp2.DoubleValue())

	// Check int64 value
	metric3 := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(2)
	gauge3 := metric3.Gauge()
	dp3 := gauge3.DataPoints().At(0)
	require.Equal(t, 1000.0, dp3.DoubleValue())
}

func TestConvertJSONToMetrics_WithAttributes(t *testing.T) {
	data := []map[string]any{
		{
			"value":    50.0,
			"service":  "api",
			"env":      "prod",
			"region":   "us-east-1",
			"instance": "server-1",
		},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	metric := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	gauge := metric.Gauge()
	dp := gauge.DataPoints().At(0)
	attrs := dp.Attributes()

	// All non-value fields should become attributes
	require.Equal(t, "api", attrs.AsRaw()["service"])
	require.Equal(t, "prod", attrs.AsRaw()["env"])
	require.Equal(t, "us-east-1", attrs.AsRaw()["region"])
	require.Equal(t, "server-1", attrs.AsRaw()["instance"])
}

func TestConvertJSONToMetrics_NoValueField(t *testing.T) {
	data := []map[string]any{
		{"name": "metric1", "count": 10},
		{"name": "metric2", "total": 20},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	// Should still create metrics, using first numeric field as value
	require.Equal(t, 2, metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())

	metric1 := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	gauge1 := metric1.Gauge()
	dp1 := gauge1.DataPoints().At(0)
	require.Equal(t, 10.0, dp1.DoubleValue())
}

func TestConvertJSONToMetrics_NoNumericValue(t *testing.T) {
	data := []map[string]any{
		{"name": "metric1", "status": "active"},
		{"name": "metric2", "enabled": true},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	// Should skip items without numeric values
	require.Equal(t, 0, metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())
}

func TestConvertJSONToMetrics_WithTimestamp(t *testing.T) {
	data := []map[string]any{
		{"value": 42.0, "timestamp": "2023-01-01T00:00:00Z"},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	metric := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	gauge := metric.Gauge()
	dp := gauge.DataPoints().At(0)

	// Timestamp should be set
	require.Greater(t, dp.Timestamp(), pcommon.Timestamp(0))
}

func TestConvertJSONToMetrics_MultipleMetrics(t *testing.T) {
	data := []map[string]any{
		{"value": 1.0, "id": "1"},
		{"value": 2.0, "id": "2"},
		{"value": 3.0, "id": "3"},
		{"value": 4.0, "id": "4"},
		{"value": 5.0, "id": "5"},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	require.Equal(t, 5, metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())

	// Verify all metrics are present
	for i := 0; i < 5; i++ {
		metric := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(i)
		gauge := metric.Gauge()
		dp := gauge.DataPoints().At(0)
		expectedValue := float64(i + 1)
		require.Equal(t, expectedValue, dp.DoubleValue())
	}
}

func TestConvertJSONToMetrics_WithNestedFields(t *testing.T) {
	data := []map[string]any{
		{
			"value": 42.0,
			"metadata": map[string]any{
				"source": "api",
				"env":    "prod",
			},
		},
	}

	logger := zap.NewNop()
	metrics := convertJSONToMetrics(data, logger)

	metric := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	gauge := metric.Gauge()
	dp := gauge.DataPoints().At(0)
	attrs := dp.Attributes()

	// Nested fields should be flattened or preserved as attributes
	// For simplicity, we'll convert nested maps to string attributes
	require.NotNil(t, attrs.AsRaw()["metadata"])
}
