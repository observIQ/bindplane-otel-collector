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
	"context"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestProcessor_Logs(t *testing.T) {
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID := "throughputmeasurement/1"

	tmp, err := NewThroughputMeasurements(mp, processorID, map[string]string{})
	require.NoError(t, err)

	logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
	require.NoError(t, err)

	tmp.AddLogs(context.Background(), logs)

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var logSize, logCount int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "otelcol_processor_throughputmeasurement_log_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key("processor"))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				logSize = sum.DataPoints[0].Value

			case "otelcol_processor_throughputmeasurement_log_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key("processor"))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				logCount = sum.DataPoints[0].Value
			}

		}
	}

	require.Equal(t, int64(3974), logSize)
	require.Equal(t, int64(3974), tmp.LogSize())
	require.Equal(t, int64(16), logCount)
	require.Equal(t, int64(16), tmp.LogCount())
}

func TestProcessor_Metrics(t *testing.T) {
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID := "throughputmeasurement/1"

	tmp, err := NewThroughputMeasurements(mp, processorID, map[string]string{})
	require.NoError(t, err)

	metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
	require.NoError(t, err)

	tmp.AddMetrics(context.Background(), metrics)

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var metricSize, datapointCount int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "otelcol_processor_throughputmeasurement_metric_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key("processor"))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				metricSize = sum.DataPoints[0].Value

			case "otelcol_processor_throughputmeasurement_metric_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key("processor"))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				datapointCount = sum.DataPoints[0].Value
			}

		}
	}

	require.Equal(t, int64(5675), metricSize)
	require.Equal(t, int64(5675), tmp.MetricSize())
	require.Equal(t, int64(37), datapointCount)
	require.Equal(t, int64(37), tmp.DatapointCount())
}

func TestProcessor_Traces(t *testing.T) {
	manualReader := metric.NewManualReader()
	defer manualReader.Shutdown(context.Background())

	mp := metric.NewMeterProvider(
		metric.WithReader(manualReader),
	)
	defer mp.Shutdown(context.Background())

	processorID := "throughputmeasurement/1"

	tmp, err := NewThroughputMeasurements(mp, processorID, map[string]string{})
	require.NoError(t, err)

	traces, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
	require.NoError(t, err)

	tmp.AddTraces(context.Background(), traces)

	var rm metricdata.ResourceMetrics
	require.NoError(t, manualReader.Collect(context.Background(), &rm))

	// Extract the metrics we care about from the metrics we collected
	var traceSize, spanCount int64

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "otelcol_processor_throughputmeasurement_trace_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key("processor"))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				traceSize = sum.DataPoints[0].Value

			case "otelcol_processor_throughputmeasurement_trace_count":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				processorAttr, ok := sum.DataPoints[0].Attributes.Value(attribute.Key("processor"))
				require.True(t, ok, "processor attribute was not found")
				require.Equal(t, processorID, processorAttr.AsString())

				spanCount = sum.DataPoints[0].Value
			}

		}
	}

	require.Equal(t, int64(16767), traceSize)
	require.Equal(t, int64(16767), tmp.TraceSize())
	require.Equal(t, int64(178), spanCount)
	require.Equal(t, int64(178), tmp.SpanCount())
}

func TestResettableThroughputMeasurementsRegistry(t *testing.T) {
	t.Run("Test registered measurements are in OTLP payload (no count metrics)", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(false)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		tmp, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{})
		require.NoError(t, err)

		traces, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
		require.NoError(t, err)

		metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
		require.NoError(t, err)

		tmp.AddLogs(context.Background(), logs)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp)

		actualMetrics := reg.OTLPMeasurements(nil)

		expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "expected", "throughput_measurements_no_count.yaml"))
		require.NoError(t, err)

		require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreTimestamp()))
	})

	t.Run("Test registered measurements are in OTLP payload (with count metrics)", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(true)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		tmp, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{})
		require.NoError(t, err)

		traces, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
		require.NoError(t, err)

		metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
		require.NoError(t, err)

		tmp.AddLogs(context.Background(), logs)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp)

		actualMetrics := reg.OTLPMeasurements(nil)

		expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "expected", "throughput_measurements_count.yaml"))
		require.NoError(t, err)

		require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreTimestamp()))
	})

	t.Run("Test registered measurements are in OTLP payload (extra attributes)", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(false)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		tmp, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{
			"gateway": "true",
		})
		require.NoError(t, err)

		traces, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
		require.NoError(t, err)

		metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
		require.NoError(t, err)

		tmp.AddLogs(context.Background(), logs)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp)

		actualMetrics := reg.OTLPMeasurements(nil)

		expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "expected", "throughput_measurements_extra_attrs.yaml"))
		require.NoError(t, err)

		require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreTimestamp()))
	})

	t.Run("Test reset removes registered measurements", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(true)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		tmp, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{})
		require.NoError(t, err)

		traces, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
		require.NoError(t, err)

		metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		logs, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
		require.NoError(t, err)

		tmp.AddLogs(context.Background(), logs)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp)

		reg.Reset()

		require.NoError(t, pmetrictest.CompareMetrics(pmetric.NewMetrics(), reg.OTLPMeasurements(nil)))
	})
}
