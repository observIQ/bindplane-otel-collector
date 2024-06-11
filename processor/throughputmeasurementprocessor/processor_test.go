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

package throughputmeasurementprocessor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/ptracetest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestProcessor_Logs(t *testing.T) {
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID := "throughputmeasurement/1"

	tmp, err := newThroughputMeasurementProcessor(zap.NewNop(), mp, &Config{
		Enabled:       true,
		SamplingRatio: 1,
	}, processorID)
	require.NoError(t, err)

	logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
	require.NoError(t, err)

	processedLogs, err := tmp.processLogs(context.Background(), logs)
	require.NoError(t, err)

	// Output logs should be the same as input logs (passthrough check)
	require.NoError(t, plogtest.CompareLogs(logs, processedLogs))

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var logSize, logCount int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "processor_throughputmeasurement_log_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				logSize = sum.DataPoints[0].Value

			case "processor_throughputmeasurement_log_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				logCount = sum.DataPoints[0].Value
			}

		}
	}

	require.Equal(t, int64(3974), logSize)
	require.Equal(t, int64(16), logCount)
}

func TestProcessor_Metrics(t *testing.T) {
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID := "throughputmeasurement/1"

	tmp, err := newThroughputMeasurementProcessor(zap.NewNop(), mp, &Config{
		Enabled:       true,
		SamplingRatio: 1,
	}, processorID)
	require.NoError(t, err)

	metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
	require.NoError(t, err)

	processedMetrics, err := tmp.processMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Output metrics should be the same as input logs (passthrough check)
	require.NoError(t, pmetrictest.CompareMetrics(metrics, processedMetrics))

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var metricSize, datapointCount int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "processor_throughputmeasurement_metric_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				metricSize = sum.DataPoints[0].Value

			case "processor_throughputmeasurement_metric_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				datapointCount = sum.DataPoints[0].Value
			}

		}
	}

	require.Equal(t, int64(5675), metricSize)
	require.Equal(t, int64(37), datapointCount)
}

func TestProcessor_Traces(t *testing.T) {
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID := "throughputmeasurement/1"

	tmp, err := newThroughputMeasurementProcessor(zap.NewNop(), mp, &Config{
		Enabled:       true,
		SamplingRatio: 1,
	}, processorID)
	require.NoError(t, err)

	traces, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
	require.NoError(t, err)

	processedTraces, err := tmp.processTraces(context.Background(), traces)
	require.NoError(t, err)

	// Output traces should be the same as input logs (passthrough check)
	require.NoError(t, ptracetest.CompareTraces(traces, processedTraces))

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var traceSize, spanCount int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "processor_throughputmeasurement_trace_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				traceSize = sum.DataPoints[0].Value

			case "processor_throughputmeasurement_trace_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				spanCount = sum.DataPoints[0].Value
			}

		}
	}

	require.Equal(t, int64(16767), traceSize)
	require.Equal(t, int64(178), spanCount)
}

// Test that 2 instances with the same processor ID add their metrics together
func TestProcessor_Logs_TwoInstancesSameID(t *testing.T) {
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID := "throughputmeasurement/1"

	tmp1, err := newThroughputMeasurementProcessor(zap.NewNop(), mp, &Config{
		Enabled:       true,
		SamplingRatio: 1,
	}, processorID)
	require.NoError(t, err)

	tmp2, err := newThroughputMeasurementProcessor(zap.NewNop(), mp, &Config{
		Enabled:       true,
		SamplingRatio: 1,
	}, processorID)
	require.NoError(t, err)

	logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
	require.NoError(t, err)

	_, err = tmp1.processLogs(context.Background(), logs)
	require.NoError(t, err)

	_, err = tmp2.processLogs(context.Background(), logs)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var logSize, logCount int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "processor_throughputmeasurement_log_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				logSize = sum.DataPoints[0].Value

			case "processor_throughputmeasurement_log_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(processorAttributeName))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				logCount = sum.DataPoints[0].Value
			}

		}
	}

	require.Equal(t, int64(2*3974), logSize)
	require.Equal(t, int64(2*16), logCount)
}

func TestProcessor_Logs_TwoInstancesDifferentID(t *testing.T) {
	// Test that different IDs shouldn't overlap, but instead create distinct datapoints.
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID1 := "throughputmeasurement/1"
	processorID2 := "throughputmeasurement/2"

	tmp1, err := newThroughputMeasurementProcessor(zap.NewNop(), mp, &Config{
		Enabled:       true,
		SamplingRatio: 1,
	}, processorID1)
	require.NoError(t, err)

	tmp2, err := newThroughputMeasurementProcessor(zap.NewNop(), mp, &Config{
		Enabled:       true,
		SamplingRatio: 1,
	}, processorID2)
	require.NoError(t, err)

	logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
	require.NoError(t, err)

	_, err = tmp1.processLogs(context.Background(), logs)
	require.NoError(t, err)

	// Ingest twice on the second processor so we get a different count for proc2
	_, err = tmp2.processLogs(context.Background(), logs)
	require.NoError(t, err)
	_, err = tmp2.processLogs(context.Background(), logs)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var logSize1, logCount1, logSize2, logCount2 int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "processor_throughputmeasurement_log_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 2, len(sum.DataPoints))

				for _, dp := range sum.DataPoints {
					processorAttr, ok := dp.Attributes.Value(attribute.Key(processorAttributeName))
					require.True(t, ok, "processor attribute was not found")

					switch processorAttr.AsString() {
					case processorID1:
						logSize1 = dp.Value
					case processorID2:
						logSize2 = dp.Value
					default:
						require.Fail(t, "ID %s should not be present in log data size metrics", processorAttr.AsString())
					}
				}

			case "processor_throughputmeasurement_log_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 2, len(sum.DataPoints))

				for _, dp := range sum.DataPoints {
					processorAttr, ok := dp.Attributes.Value(attribute.Key(processorAttributeName))
					require.True(t, ok, "processor attribute was not found")

					switch processorAttr.AsString() {
					case processorID1:
						logCount1 = dp.Value
					case processorID2:
						logCount2 = dp.Value
					default:
						require.Fail(t, "ID %s should not be present in log count metrics", processorAttr.AsString())
					}
				}
			}

		}
	}

	require.Equal(t, int64(3974), logSize1)
	require.Equal(t, int64(16), logCount1)

	require.Equal(t, int64(2*3974), logSize2)
	require.Equal(t, int64(2*16), logCount2)
}
