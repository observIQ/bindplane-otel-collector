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
	"go.opentelemetry.io/otel/metric/noop"
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

	tmp.AddLogs(context.Background(), logs, false)

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

func TestProcessor_LogsWithLogRecordOriginal(t *testing.T) {
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

	// Add log.record.original to all log records
	resourceLogs := logs.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		resourceLog := resourceLogs.At(i)
		scopeLogs := resourceLog.ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			scopeLog := scopeLogs.At(j)
			logRecords := scopeLog.LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				logRecord := logRecords.At(k)
				// Set original to a fixed test value (20 bytes) for all records
				logRecord.Attributes().PutStr("log.record.original", "12345678901234567890")
			}
		}
	}

	tmp.AddLogs(context.Background(), logs, true)

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

	// Verify that the log size is based on the full log content
	expectedLogSize := int64(4727)
	// Verify that the log raw bytes size is based on the original log content
	expectedLogRawBytes := int64(16 * 20)
	require.Equal(t, expectedLogSize, logSize)
	require.Equal(t, expectedLogSize, tmp.LogSize())
	require.Equal(t, expectedLogRawBytes, tmp.logLogRawBytes.Val())
	require.Equal(t, int64(16), logCount)
	require.Equal(t, int64(16), tmp.LogCount())
}

func TestProcessor_LogsWithoutLogRecordOriginal(t *testing.T) {
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

	// Set a fixed body value for all log records to make testing easier
	resourceLogs := logs.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		resourceLog := resourceLogs.At(i)
		scopeLogs := resourceLog.ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			scopeLog := scopeLogs.At(j)
			logRecords := scopeLog.LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				logRecord := logRecords.At(k)
				// Set body to a fixed test value (25 bytes) for all records
				logRecord.Body().SetStr("1234567890123456789012345")
			}
		}
	}

	tmp.AddLogs(context.Background(), logs, true)

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

	// Verify that the log size is based on the full log content
	expectedLogSize := int64(1792)
	// Verify that the log raw bytes size is based on the log body content (25 bytes per record)
	expectedLogRawBytes := int64(16 * 25)
	require.Equal(t, expectedLogSize, logSize)
	require.Equal(t, expectedLogSize, tmp.LogSize())
	require.Equal(t, expectedLogRawBytes, tmp.logLogRawBytes.Val())
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

		tmp.AddLogs(context.Background(), logs, true)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp))

		actualMetrics := reg.OTLPMeasurements(nil)

		expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "expected", "throughput_measurements_no_count.yaml"))
		require.NoError(t, err)

		require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreTimestamp()))
	})

	t.Run("Test sequence tracking", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(false)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		tmp, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{})
		require.NoError(t, err)

		metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		// First batch of metrics
		tmp.AddMetrics(context.Background(), metrics)
		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp))

		// Get first batch of measurements
		firstMetrics := reg.OTLPMeasurements(nil)
		require.NotEmpty(t, firstMetrics.DataPointCount())

		// Add more metrics
		tmp.AddMetrics(context.Background(), metrics)

		// Get second batch of measurements
		secondMetrics := reg.OTLPMeasurements(nil)
		require.NotEmpty(t, secondMetrics.DataPointCount())

		// Verify sequence numbers
		require.Equal(t, int64(2), tmp.SequenceNumber())

		reg.measurements.Range(func(key, value any) bool {
			processorID := key.(string)
			if processorID == "throughputmeasurement/1" {
				require.Equal(t, int64(2), value.(*processorMeasurements).lastCollectedSequence)
			}
			return true
		})

		// Verify no new metrics if sequence hasn't changed
		emptyMetrics := reg.OTLPMeasurements(nil)
		require.Empty(t, emptyMetrics.DataPointCount())
	})

	t.Run("Test multiple processors with different sequences", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(false)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		// Create two processors
		tmp1, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{})
		require.NoError(t, err)
		tmp2, err := NewThroughputMeasurements(mp, "throughputmeasurement/2", map[string]string{})
		require.NoError(t, err)

		metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		// Add metrics to first processor
		tmp1.AddMetrics(context.Background(), metrics)
		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp1))

		// Get first batch of measurements
		firstMetrics := reg.OTLPMeasurements(nil)
		require.NotEmpty(t, firstMetrics.DataPointCount())

		// Add metrics to second processor
		tmp2.AddMetrics(context.Background(), metrics)
		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/2", tmp2))

		// Get second batch of measurements
		secondMetrics := reg.OTLPMeasurements(nil)
		require.NotEmpty(t, secondMetrics.DataPointCount())

		// Verify sequence numbers
		require.Equal(t, int64(1), tmp1.SequenceNumber())
		require.Equal(t, int64(1), tmp2.SequenceNumber())

		// Add more metrics to both processors
		tmp1.AddMetrics(context.Background(), metrics)
		tmp1.AddMetrics(context.Background(), metrics) // simulate out of sync between processors
		tmp2.AddMetrics(context.Background(), metrics)
		reg.OTLPMeasurements(nil)

		reg.measurements.Range(func(key, value any) bool {
			processorID := key.(string)
			if processorID == "throughputmeasurement/1" {
				require.Equal(t, int64(3), value.(*processorMeasurements).lastCollectedSequence)
			}
			if processorID == "throughputmeasurement/2" {
				require.Equal(t, int64(2), value.(*processorMeasurements).lastCollectedSequence)
			}
			return true
		})
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

		tmp.AddLogs(context.Background(), logs, true)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp))

		actualMetrics := reg.OTLPMeasurements(nil)

		expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "expected", "throughput_measurements_count.yaml"))
		require.NoError(t, err)

		require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreTimestamp()))
	})

	t.Run("Test only metrics throughput", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(false)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		tmp, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{})
		require.NoError(t, err)

		metrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		tmp.AddMetrics(context.Background(), metrics)

		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp))

		actualMetrics := reg.OTLPMeasurements(nil)

		expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "expected", "throughput_measurements_metrics_only.yaml"))
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

		tmp.AddLogs(context.Background(), logs, true)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp))

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

		tmp.AddLogs(context.Background(), logs, false)
		tmp.AddMetrics(context.Background(), metrics)
		tmp.AddTraces(context.Background(), traces)

		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp))

		reg.Reset()

		require.NoError(t, pmetrictest.CompareMetrics(pmetric.NewMetrics(), reg.OTLPMeasurements(nil)))
	})

	t.Run("Double registering is an error", func(t *testing.T) {
		reg := NewResettableThroughputMeasurementsRegistry(false)

		mp := noop.NewMeterProvider()

		tmp, err := NewThroughputMeasurements(mp, "throughputmeasurement/1", map[string]string{})
		require.NoError(t, err)

		require.NoError(t, reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp))

		err = reg.RegisterThroughputMeasurements("throughputmeasurement/1", tmp)
		require.Error(t, err)
		require.Equal(t, err.Error(), `measurements for processor "throughputmeasurement/1" was already registered`)
	})
}
