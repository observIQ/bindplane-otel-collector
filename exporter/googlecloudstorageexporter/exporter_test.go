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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-contrib/exporter/googlecloudstorageexporter"

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/observiq/bindplane-otel-contrib/exporter/googlecloudstorageexporter/internal/metadata"
	"github.com/observiq/bindplane-otel-contrib/exporter/googlecloudstorageexporter/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

// newTestTelemetryBuilder creates a TelemetryBuilder backed by the given MeterProvider for testing.
func newTestTelemetryBuilder(t *testing.T, mp *sdkmetric.MeterProvider) *metadata.TelemetryBuilder {
	t.Helper()
	settings := component.TelemetrySettings{
		MeterProvider: mp,
	}
	tb, err := metadata.NewTelemetryBuilder(settings)
	require.NoError(t, err)
	return tb
}

func Test_exporter_Capabilities(t *testing.T) {
	exp := &googleCloudStorageExporter{}
	capabilities := exp.Capabilities()
	require.False(t, capabilities.MutatesData)
}

func Test_exporter_metricsDataPusher(t *testing.T) {
	cfg := &Config{
		BucketName:   "bucket",
		ProjectID:    "project",
		FolderName:   "folder",
		ObjectPrefix: "prefix",
		Partition:    minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (storageClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input pmetric.Metrics, _ []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(nil, errors.New("marshal"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Storage client error",
			mockGen: func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(nil)

				return mockStorageClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			tb := newTestTelemetryBuilder(t, mp)

			md, expectedBytes := generateTestMetrics(t)
			mockStorageClient, mockMarshaler := tc.mockGen(t, md, expectedBytes)
			exp := &googleCloudStorageExporter{
				cfg:           cfg,
				storageClient: mockStorageClient,
				logger:        zap.NewNop(),
				marshaler:     mockMarshaler,
				telemetry:     tb,
			}

			err := exp.metricsDataPusher(context.Background(), md)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

		})
	}
}

func Test_exporter_logsDataPusher(t *testing.T) {
	cfg := &Config{
		BucketName:   "bucket",
		ProjectID:    "project",
		FolderName:   "folder",
		ObjectPrefix: "prefix",
		Partition:    minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input plog.Logs, expectBuff []byte) (storageClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input plog.Logs, _ []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(nil, errors.New("marshal"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Storage client error",
			mockGen: func(t *testing.T, input plog.Logs, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input plog.Logs, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(nil)

				return mockStorageClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			tb := newTestTelemetryBuilder(t, mp)

			ld, expectedBytes := generateTestLogs(t)
			mockStorageClient, mockMarshaler := tc.mockGen(t, ld, expectedBytes)
			exp := &googleCloudStorageExporter{
				cfg:           cfg,
				storageClient: mockStorageClient,
				logger:        zap.NewNop(),
				marshaler:     mockMarshaler,
				telemetry:     tb,
			}

			err := exp.logsDataPusher(context.Background(), ld)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

		})
	}
}

func Test_exporter_traceDataPusher(t *testing.T) {
	cfg := &Config{
		BucketName:   "bucket",
		ProjectID:    "project",
		FolderName:   "folder",
		ObjectPrefix: "prefix",
		Partition:    minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input ptrace.Traces, expectBuff []byte) (storageClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input ptrace.Traces, _ []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(nil, errors.New("marshal"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Storage client error",
			mockGen: func(t *testing.T, input ptrace.Traces, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input ptrace.Traces, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(nil)

				return mockStorageClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			tb := newTestTelemetryBuilder(t, mp)

			td, expectedBytes := generateTestTraces(t)
			mockStorageClient, mockMarshaler := tc.mockGen(t, td, expectedBytes)
			exp := &googleCloudStorageExporter{
				cfg:           cfg,
				storageClient: mockStorageClient,
				logger:        zap.NewNop(),
				marshaler:     mockMarshaler,
				telemetry:     tb,
			}

			err := exp.tracesDataPusher(context.Background(), td)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

		})
	}
}

func Test_exporter_getObjectName(t *testing.T) {
	testCases := []struct {
		desc          string
		cfg           *Config
		telemetryType string
		expectedRegex string
	}{
		{
			desc: "Base Empty config",
			cfg: &Config{
				BucketName: "bucket",
				ProjectID:  "project",
				Partition:  minutePartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/minute=\d{2}/metrics_\d+\.json$`,
		},
		{
			desc: "Base Empty config hour",
			cfg: &Config{
				BucketName: "bucket",
				ProjectID:  "project",
				Partition:  hourPartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/metrics_\d+\.json$`,
		},
		{
			desc: "Full config",
			cfg: &Config{
				BucketName:   "bucket",
				ProjectID:    "project",
				FolderName:   "folder",
				ObjectPrefix: "prefix",
				Partition:    minutePartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^folder/year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/minute=\d{2}/prefixmetrics_\d+\.json$`,
		},
	}

	for _, tc := range testCases {
		currentTc := tc
		t.Run(currentTc.desc, func(t *testing.T) {
			t.Parallel()
			mockMarshaller := mocks.NewMockMarshaler(t)
			mockMarshaller.EXPECT().Format().Return("json")

			exp := googleCloudStorageExporter{
				cfg:       currentTc.cfg,
				marshaler: mockMarshaller,
			}

			actual := exp.getObjectName(currentTc.telemetryType)
			require.Regexp(t, regexp.MustCompile(currentTc.expectedRegex), actual)
		})
	}
}

func Test_classifyError(t *testing.T) {
	testCases := []struct {
		desc     string
		err      error
		expected string
	}{
		{
			desc:     "nil error",
			err:      nil,
			expected: "none",
		},
		{
			desc:     "deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: "timeout",
		},
		{
			desc:     "wrapped deadline exceeded",
			err:      fmt.Errorf("upload failed: %w", context.DeadlineExceeded),
			expected: "timeout",
		},
		{
			desc:     "arbitrary error",
			err:      errors.New("something went wrong"),
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.expected, classifyError(tc.err))
		})
	}
}

// collectMetrics is a test helper that collects metrics from a ManualReader.
func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	return rm
}

// findMetric searches collected metrics for one matching the given name.
func findMetric(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

func Test_logsDataPusher_RecordsMetrics_Success(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	tb := newTestTelemetryBuilder(t, mp)

	cfg := &Config{
		BucketName:     "test-bucket",
		BucketLocation: "US",
		Partition:      minutePartition,
	}

	ld, expectedBytes := generateTestLogs(t)

	mockStorageClient := mocks.NewMockStorageClient(t)
	mockMarshaler := mocks.NewMockMarshaler(t)
	mockMarshaler.EXPECT().MarshalLogs(ld).Return(expectedBytes, nil)
	mockMarshaler.EXPECT().Format().Return("json")
	mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectedBytes).Return(nil)

	exp := &googleCloudStorageExporter{
		cfg:           cfg,
		storageClient: mockStorageClient,
		logger:        zap.NewNop(),
		marshaler:     mockMarshaler,
		telemetry:     tb,
	}

	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	rm := collectMetrics(t, reader)

	// Verify payload size was recorded
	m := findMetric(rm, "otelcol_exporter_payload_size")
	require.NotNil(t, m, "exporter_payload_size metric should be recorded")
	hist := m.Data.(metricdata.Histogram[int64])
	require.Len(t, hist.DataPoints, 1)
	require.Equal(t, int64(len(expectedBytes)), hist.DataPoints[0].Sum)
	assertHasAttribute(t, hist.DataPoints[0].Attributes, "encoding", "json")
	assertHasAttribute(t, hist.DataPoints[0].Attributes, "bucket", "test-bucket")

	// Verify request duration was recorded with error=none
	m = findMetric(rm, "otelcol_exporter_request_duration")
	require.NotNil(t, m, "exporter_request_duration metric should be recorded")
	durHist := m.Data.(metricdata.Histogram[int64])
	require.Len(t, durHist.DataPoints, 1)
	assertHasAttribute(t, durHist.DataPoints[0].Attributes, "error", "none")
	assertHasAttribute(t, durHist.DataPoints[0].Attributes, "bucket", "test-bucket")
	assertHasAttribute(t, durHist.DataPoints[0].Attributes, "location", "US")

	// Verify upload bytes total was incremented
	m = findMetric(rm, "otelcol_exporter_upload_bytes_total")
	require.NotNil(t, m, "exporter_upload_bytes_total metric should be recorded")
	sum := m.Data.(metricdata.Sum[int64])
	require.Len(t, sum.DataPoints, 1)
	require.Equal(t, int64(len(expectedBytes)), sum.DataPoints[0].Value)

	// Verify upload inflight returned to 0
	m = findMetric(rm, "otelcol_exporter_upload_inflight")
	require.NotNil(t, m, "exporter_upload_inflight metric should be recorded")
	inflightSum := m.Data.(metricdata.Sum[int64])
	require.Len(t, inflightSum.DataPoints, 1)
	require.Equal(t, int64(0), inflightSum.DataPoints[0].Value)

	// Verify timeout total was NOT incremented
	m = findMetric(rm, "otelcol_exporter_timeout_total")
	if m != nil {
		timeoutSum := m.Data.(metricdata.Sum[int64])
		for _, dp := range timeoutSum.DataPoints {
			require.Equal(t, int64(0), dp.Value)
		}
	}
}

func Test_logsDataPusher_RecordsMetrics_Timeout(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	tb := newTestTelemetryBuilder(t, mp)

	cfg := &Config{
		BucketName:     "test-bucket",
		BucketLocation: "US",
		Partition:      minutePartition,
	}

	ld, expectedBytes := generateTestLogs(t)

	mockStorageClient := mocks.NewMockStorageClient(t)
	mockMarshaler := mocks.NewMockMarshaler(t)
	mockMarshaler.EXPECT().MarshalLogs(ld).Return(expectedBytes, nil)
	mockMarshaler.EXPECT().Format().Return("json")
	mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectedBytes).Return(context.DeadlineExceeded)

	exp := &googleCloudStorageExporter{
		cfg:           cfg,
		storageClient: mockStorageClient,
		logger:        zap.NewNop(),
		marshaler:     mockMarshaler,
		telemetry:     tb,
	}

	err := exp.logsDataPusher(context.Background(), ld)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	rm := collectMetrics(t, reader)

	// Verify request duration recorded with error=timeout
	m := findMetric(rm, "otelcol_exporter_request_duration")
	require.NotNil(t, m)
	durHist := m.Data.(metricdata.Histogram[int64])
	require.Len(t, durHist.DataPoints, 1)
	assertHasAttribute(t, durHist.DataPoints[0].Attributes, "error", "timeout")

	// Verify timeout total was incremented
	m = findMetric(rm, "otelcol_exporter_timeout_total")
	require.NotNil(t, m, "exporter_timeout_total metric should be recorded")
	sum := m.Data.(metricdata.Sum[int64])
	require.Len(t, sum.DataPoints, 1)
	require.Equal(t, int64(1), sum.DataPoints[0].Value)

	// Verify upload bytes total was NOT incremented
	m = findMetric(rm, "otelcol_exporter_upload_bytes_total")
	if m != nil {
		bytesSum := m.Data.(metricdata.Sum[int64])
		for _, dp := range bytesSum.DataPoints {
			require.Equal(t, int64(0), dp.Value)
		}
	}
}

func Test_logsDataPusher_RecordsMetrics_UnknownError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	tb := newTestTelemetryBuilder(t, mp)

	cfg := &Config{
		BucketName:     "test-bucket",
		BucketLocation: "US",
		Partition:      minutePartition,
	}

	ld, expectedBytes := generateTestLogs(t)

	mockStorageClient := mocks.NewMockStorageClient(t)
	mockMarshaler := mocks.NewMockMarshaler(t)
	mockMarshaler.EXPECT().MarshalLogs(ld).Return(expectedBytes, nil)
	mockMarshaler.EXPECT().Format().Return("json")
	mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectedBytes).Return(errors.New("network error"))

	exp := &googleCloudStorageExporter{
		cfg:           cfg,
		storageClient: mockStorageClient,
		logger:        zap.NewNop(),
		marshaler:     mockMarshaler,
		telemetry:     tb,
	}

	err := exp.logsDataPusher(context.Background(), ld)
	require.Error(t, err)

	rm := collectMetrics(t, reader)

	// Verify request duration recorded with error=unknown
	m := findMetric(rm, "otelcol_exporter_request_duration")
	require.NotNil(t, m)
	durHist := m.Data.(metricdata.Histogram[int64])
	require.Len(t, durHist.DataPoints, 1)
	assertHasAttribute(t, durHist.DataPoints[0].Attributes, "error", "unknown")

	// Verify neither timeout total nor upload bytes total were incremented
	m = findMetric(rm, "otelcol_exporter_timeout_total")
	if m != nil {
		sum := m.Data.(metricdata.Sum[int64])
		for _, dp := range sum.DataPoints {
			require.Equal(t, int64(0), dp.Value)
		}
	}

	m = findMetric(rm, "otelcol_exporter_upload_bytes_total")
	if m != nil {
		sum := m.Data.(metricdata.Sum[int64])
		for _, dp := range sum.DataPoints {
			require.Equal(t, int64(0), dp.Value)
		}
	}
}

func Test_uploadInflight_DuringUpload(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	tb := newTestTelemetryBuilder(t, mp)

	cfg := &Config{
		BucketName:     "test-bucket",
		BucketLocation: "US",
		Partition:      minutePartition,
	}

	ld, expectedBytes := generateTestLogs(t)

	// Use a channel to block the upload so we can observe inflight=1
	uploadStarted := make(chan struct{})
	uploadContinue := make(chan struct{})

	mockStorageClient := mocks.NewMockStorageClient(t)
	mockMarshaler := mocks.NewMockMarshaler(t)
	mockMarshaler.EXPECT().MarshalLogs(ld).Return(expectedBytes, nil)
	mockMarshaler.EXPECT().Format().Return("json")
	mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectedBytes).
		RunAndReturn(func(_ context.Context, _ string, _ []byte) error {
			close(uploadStarted)
			<-uploadContinue
			return nil
		})

	exp := &googleCloudStorageExporter{
		cfg:           cfg,
		storageClient: mockStorageClient,
		logger:        zap.NewNop(),
		marshaler:     mockMarshaler,
		telemetry:     tb,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = exp.logsDataPusher(context.Background(), ld)
	}()

	// Wait for upload to start
	<-uploadStarted

	// Collect metrics while upload is in-flight
	rm := collectMetrics(t, reader)
	m := findMetric(rm, "otelcol_exporter_upload_inflight")
	require.NotNil(t, m, "exporter_upload_inflight metric should be recorded")
	inflightSum := m.Data.(metricdata.Sum[int64])
	require.Len(t, inflightSum.DataPoints, 1)
	require.Equal(t, int64(1), inflightSum.DataPoints[0].Value)

	// Let upload complete
	close(uploadContinue)
	wg.Wait()

	// Verify inflight returned to 0
	rm = collectMetrics(t, reader)
	m = findMetric(rm, "otelcol_exporter_upload_inflight")
	require.NotNil(t, m)
	inflightSum = m.Data.(metricdata.Sum[int64])
	require.Len(t, inflightSum.DataPoints, 1)
	require.Equal(t, int64(0), inflightSum.DataPoints[0].Value)
}

// assertHasAttribute checks that the given attribute set contains a string attribute with the expected value.
func assertHasAttribute(t *testing.T, attrs attribute.Set, key, expectedValue string) {
	t.Helper()
	val, ok := attrs.Value(attribute.Key(key))
	require.True(t, ok, "attribute %q not found", key)
	require.Equal(t, expectedValue, val.AsString(), "attribute %q has wrong value", key)
}
