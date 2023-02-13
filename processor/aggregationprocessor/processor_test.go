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

package aggregationprocessor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/observiq/observiq-otel-collector/processor/aggregationprocessor/internal/aggregate"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zaptest"
)

const processorStartUnixMilli = 1675866200681

func TestAggregationProcessor(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
		// expectOutput = true means that some part of the input will pass through
		expectOutput bool
		// noAggregate = true means flushing has no output.
		noAggregate bool
	}{
		{
			name:     "gauge",
			filePath: "gauge.json",
		},
		{
			name:     "gauge value is int",
			filePath: "gauge-integer.json",
		},
		{
			name:         "empty datapoint",
			filePath:     "empty-datapoint.json",
			expectOutput: true,
			noAggregate:  true,
		},
		{
			name:         "histogram",
			filePath:     "histogram.json",
			expectOutput: true,
			noAggregate:  true,
		},
		{
			name:         "metric name doesn't match regex",
			filePath:     "metric-name-doesnt-match.json",
			expectOutput: true,
			noAggregate:  true,
		},
		{
			name:     "multiple datapoints",
			filePath: "multiple-datapoints.json",
		},
		{
			name:     "multiple metrics",
			filePath: "multiple-metrics.json",
		},
		{
			name:         "multiple metrics, only one matches regex",
			filePath:     "multiple-metrics-partial-match.json",
			expectOutput: true,
		},
		{
			name:     "multiple of the same metric",
			filePath: "multiple-same-metrics.json",
		},
		{
			name:     "multiple resources",
			filePath: "multiple-resources.json",
		},
		{
			name:         "sum with delta aggregation temporality",
			filePath:     "sum-delta.json",
			expectOutput: true,
			noAggregate:  true,
		},
		{
			name:     "monotonic sum",
			filePath: "sum-monotonic.json",
		},
		{
			name:     "non-monotonic sum",
			filePath: "sum-non-monotonic.json",
		},
	}

	for _, tc := range testCases {
		now := time.UnixMilli(processorStartUnixMilli)
		aggPeriodStart := pcommon.NewTimestampFromTime(now.Add(-1 * time.Minute))
		t.Run(tc.name, func(t *testing.T) {
			consumer := &consumertest.MetricsSink{}
			p, err := newAggregationProcessor(zaptest.NewLogger(t), &Config{
				Interval: 0,
				Include:  `^test\..*$`,
				Aggregations: []AggregateConfig{
					{
						Type:       aggregate.MinType,
						MetricName: "$0.min",
					},
					{
						Type:       aggregate.MaxType,
						MetricName: "$0.max",
					},
					{
						Type:       aggregate.AvgType,
						MetricName: "$0.avg",
					},
				},
			}, consumer)
			require.NoError(t, err)

			p.aggregationPeriodStart = aggPeriodStart
			p.now = func() time.Time {
				return now
			}

			inputMetrics := readMetrics(t, filepath.Join("testdata", "input", tc.filePath))
			require.NoError(t, p.ConsumeMetrics(context.Background(), inputMetrics))

			if tc.expectOutput {
				metrics := consumer.AllMetrics()
				require.NotEmpty(t, metrics, "No metrics were output, but we expected some to be.")

				expectedAggregate := readMetrics(t, filepath.Join("testdata", "output", tc.filePath))
				require.NoError(t, pmetrictest.CompareMetrics(expectedAggregate, metrics[0],
					pmetrictest.IgnoreResourceMetricsOrder(),
					pmetrictest.IgnoreScopeMetricsOrder(),
					pmetrictest.IgnoreMetricsOrder(),
					pmetrictest.IgnoreMetricDataPointsOrder(),
				))

				consumer.Reset()
			} else {
				require.Empty(t, consumer.AllMetrics(), "Metrics were output, but we didn't expect any to be.")
			}

			p.flush()

			if tc.noAggregate {
				require.Empty(t, consumer.AllMetrics(), "Aggregate metrics were output, but we didn't expect any to be.")
			} else {
				metrics := consumer.AllMetrics()
				require.NotEmpty(t, metrics, "No aggregate metrics were output, but we expected some to be.")

				expectedAggregate := readMetrics(t, filepath.Join("testdata", "aggregate", tc.filePath))
				require.NoError(t, pmetrictest.CompareMetrics(expectedAggregate, metrics[0],
					pmetrictest.IgnoreResourceMetricsOrder(),
					pmetrictest.IgnoreScopeMetricsOrder(),
					pmetrictest.IgnoreMetricsOrder(),
					pmetrictest.IgnoreMetricDataPointsOrder(),
				))
			}
		})
	}
}

func TestAggregationProcessorMultipleMetrics(t *testing.T) {
	now := time.UnixMilli(processorStartUnixMilli)
	aggPeriodStart := pcommon.NewTimestampFromTime(now.Add(-1 * time.Minute))
	consumer := &consumertest.MetricsSink{}
	p, err := newAggregationProcessor(zaptest.NewLogger(t), &Config{
		Interval: 0,
		Include:  `^test\..*$`,
		Aggregations: []AggregateConfig{
			{
				Type:       aggregate.MinType,
				MetricName: "$0.min",
			},
			{
				Type:       aggregate.MaxType,
				MetricName: "$0.max",
			},
			{
				Type:       aggregate.AvgType,
				MetricName: "$0.avg",
			},
		},
	}, consumer)
	require.NoError(t, err)

	p.aggregationPeriodStart = aggPeriodStart
	p.now = func() time.Time {
		return now
	}

	metric1 := readMetrics(t, filepath.Join("testdata", "input", "datapoint-1.json"))
	metric2 := readMetrics(t, filepath.Join("testdata", "input", "datapoint-2.json"))

	require.NoError(t, p.ConsumeMetrics(context.Background(), metric1))
	require.NoError(t, p.ConsumeMetrics(context.Background(), metric2))

	require.Empty(t, consumer.AllMetrics())

	p.flush()

	require.Len(t, consumer.AllMetrics(), 1)
	aggregateMetric := consumer.AllMetrics()[0]

	expectedAggregate := readMetrics(t, filepath.Join("testdata", "aggregate", "multiple-metrics-consumed.json"))
	require.NoError(t, pmetrictest.CompareMetrics(expectedAggregate, aggregateMetric,
		pmetrictest.IgnoreResourceMetricsOrder(),
		pmetrictest.IgnoreScopeMetricsOrder(),
		pmetrictest.IgnoreMetricsOrder(),
		pmetrictest.IgnoreMetricDataPointsOrder(),
	))
}

func TestAggregationProcessor_StartShutdown(t *testing.T) {
	t.Run("start then stop", func(t *testing.T) {
		p, err := newAggregationProcessor(zaptest.NewLogger(t), &Config{
			Interval:     10 * time.Second,
			Include:      `^test\..*$`,
			Aggregations: []AggregateConfig{},
		}, &consumertest.MetricsSink{})
		require.NoError(t, err)
		require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
		require.NoError(t, p.Shutdown(context.Background()))
	})

	t.Run("shutdown without start", func(t *testing.T) {
		p, err := newAggregationProcessor(zaptest.NewLogger(t), &Config{
			Interval:     10 * time.Second,
			Include:      `^test\..*$`,
			Aggregations: []AggregateConfig{},
		}, &consumertest.MetricsSink{})
		require.NoError(t, err)
		require.NoError(t, p.Shutdown(context.Background()))
	})

	t.Run("shutdown, context times out", func(t *testing.T) {
		p, err := newAggregationProcessor(zaptest.NewLogger(t), &Config{
			Interval:     10 * time.Second,
			Include:      `^test\..*$`,
			Aggregations: []AggregateConfig{},
		}, &consumertest.MetricsSink{})
		require.NoError(t, err)

		p.wg.Add(1)
		cancelledContext, cancel := context.WithCancel(context.Background())
		cancel()

		require.ErrorIs(t, p.Shutdown(cancelledContext), context.Canceled)
	})
}

func TestAggregationProcessor_Flush(t *testing.T) {
	now := time.UnixMilli(processorStartUnixMilli)
	aggPeriodStart := pcommon.NewTimestampFromTime(now.Add(-1 * time.Minute))

	consumer := &consumertest.MetricsSink{}
	p, err := newAggregationProcessor(zaptest.NewLogger(t), &Config{
		Interval: 500 * time.Millisecond,
		Include:  `^test\..*$`,
		Aggregations: []AggregateConfig{
			{
				Type:       aggregate.MinType,
				MetricName: "$0.min",
			},
			{
				Type:       aggregate.MaxType,
				MetricName: "$0.max",
			},
			{
				Type:       aggregate.AvgType,
				MetricName: "$0.avg",
			},
		},
	}, consumer)
	require.NoError(t, err)

	p.aggregationPeriodStart = aggPeriodStart
	p.now = func() time.Time {
		return now
	}

	inputMetrics := readMetrics(t, filepath.Join("testdata", "input", "gauge.json"))
	require.NoError(t, p.ConsumeMetrics(context.Background(), inputMetrics))
	require.Empty(t, consumer.AllMetrics())
	// We'll start the flushloop after we consume, just to be 100% sure that the ConsumeMetrics function didn't forward to the consumer
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	// Wait for flush
	require.Eventually(t, func() bool {
		return len(consumer.AllMetrics()) > 0
	}, 5*time.Second, 100*time.Millisecond)

	metrics := consumer.AllMetrics()
	expectedAggregate := readMetrics(t, filepath.Join("testdata", "aggregate", "gauge.json"))
	require.NoError(t, pmetrictest.CompareMetrics(expectedAggregate, metrics[0],
		pmetrictest.IgnoreResourceMetricsOrder(),
		pmetrictest.IgnoreScopeMetricsOrder(),
		pmetrictest.IgnoreMetricsOrder(),
		pmetrictest.IgnoreMetricDataPointsOrder(),
	))

	require.NoError(t, p.Shutdown(context.Background()))
}

// TODO: Two separate metric payloads with the same datapoint attributes, but different values

func readMetrics(t *testing.T, path string) pmetric.Metrics {
	t.Helper()

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	unmarshaller := pmetric.JSONUnmarshaler{}
	m, err := unmarshaller.UnmarshalMetrics(b)
	require.NoError(t, err)

	return m
}

// Helper function to write out metrics payloads. Use this to re-generate metric payloads for tests
// func writeMetrics(t *testing.T, path string, m pmetric.Metrics) {
// 	t.Helper()

// 	marshaller := pmetric.JSONMarshaler{}
// 	b, err := marshaller.MarshalMetrics(m)
// 	require.NoError(t, err)

// 	// For formatting
// 	var metricMap map[string]any
// 	require.NoError(t, json.Unmarshal(b, &metricMap))

// 	b, err = json.MarshalIndent(metricMap, "", "    ")
// 	require.NoError(t, err)
// 	b = append(b, '\n')

// 	require.NoError(t, os.WriteFile(path, b, 0666))
// }
