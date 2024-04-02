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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver"

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver/internal/azureblob"
	blobmocks "github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver/internal/azureblob/mocks"
	"github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
)

func Test_newMetricsReceiver(t *testing.T) {
	mockClient := setNewAzureBlobClient(t)
	testType, err := component.NewType("test")
	require.NoError(t, err)

	id := component.NewID(testType)
	testLogger := zap.NewNop()
	cfg := &Config{
		StartingTime: "2023-10-02T17:00",
		EndingTime:   "2023-10-02T17:01",
	}
	co := consumertest.NewNop()
	r, err := newMetricsReceiver(id, testLogger, cfg, co)
	require.NoError(t, err)

	require.Equal(t, testLogger, r.logger)
	require.Equal(t, id, r.id)
	require.Equal(t, mockClient, r.azureClient)
	require.Equal(t, component.DataTypeMetrics, r.supportedTelemetry)
	require.IsType(t, &metricsConsumer{}, r.consumer)
}

func Test_newLogsReceiver(t *testing.T) {
	mockClient := setNewAzureBlobClient(t)
	testType, err := component.NewType("test")
	require.NoError(t, err)

	id := component.NewID(testType)
	testLogger := zap.NewNop()
	cfg := &Config{
		StartingTime: "2023-10-02T17:00",
		EndingTime:   "2023-10-02T17:01",
	}
	co := consumertest.NewNop()
	r, err := newLogsReceiver(id, testLogger, cfg, co)
	require.NoError(t, err)

	require.Equal(t, testLogger, r.logger)
	require.Equal(t, id, r.id)
	require.Equal(t, mockClient, r.azureClient)
	require.Equal(t, component.DataTypeLogs, r.supportedTelemetry)
	require.IsType(t, &logsConsumer{}, r.consumer)
}

func Test_newTracesReceiver(t *testing.T) {
	mockClient := setNewAzureBlobClient(t)
	testType, err := component.NewType("test")
	require.NoError(t, err)

	id := component.NewID(testType)
	testLogger := zap.NewNop()
	cfg := &Config{
		StartingTime: "2023-10-02T17:00",
		EndingTime:   "2023-10-02T17:01",
	}
	co := consumertest.NewNop()
	r, err := newTracesReceiver(id, testLogger, cfg, co)
	require.NoError(t, err)

	require.Equal(t, testLogger, r.logger)
	require.Equal(t, id, r.id)
	require.Equal(t, mockClient, r.azureClient)
	require.Equal(t, component.DataTypeTraces, r.supportedTelemetry)
	require.IsType(t, &tracesConsumer{}, r.consumer)
}

func Test_fullRehydration(t *testing.T) {
	testType, err := component.NewType("test")
	require.NoError(t, err)

	id := component.NewID(testType)
	testLogger := zap.NewNop()
	cfg := &Config{
		StartingTime: "2023-10-02T17:00",
		EndingTime:   "2023-10-02T18:00",
		PollInterval: 10 * time.Millisecond,
		Container:    "container",
		DeleteOnRead: false,
	}

	t.Run("empty blob polling", func(t *testing.T) {
		var listCounter atomic.Int32

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		mockClient.EXPECT().ListBlobs(mock.Anything, cfg.Container, (*string)(nil), (*string)(nil)).Times(3).Return([]*azureblob.BlobInfo{}, nil, nil).
			Run(func(_ mock.Arguments) {
				listCounter.Add(1)
			})

		// Create new receiver
		testConsumer := &consumertest.MetricsSink{}
		r, err := newMetricsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		checkFunc := func() bool {
			return listCounter.Load() == 3
		}
		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("metrics", func(t *testing.T) {
		// Test data
		metrics, jsonBytes := generateTestMetrics(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*azureblob.BlobInfo{
			{
				Name: "year=2023/month=10/day=02/hour=17/minute=05/blobmetrics_12345.json",
				Size: expectedBuffSize,
			},
			{
				Name: "year=2023/month=10/day=01/hour=17/minute=05/blobmetrics_7890.json",
				Size: 5,
			},
		}

		targetBlob := returnedBlobInfo[0]

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		mockClient.EXPECT().ListBlobs(mock.Anything, cfg.Container, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})

		// Create new receiver
		testConsumer := &consumertest.MetricsSink{}
		r, err := newMetricsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		checkFunc := func() bool {
			return testConsumer.DataPointCount() == metrics.DataPointCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("traces", func(t *testing.T) {
		// Test data
		traces, jsonBytes := generateTestTraces(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*azureblob.BlobInfo{
			{
				Name: "year=2023/month=10/day=02/hour=17/minute=05/blobtraces_12345.json",
				Size: expectedBuffSize,
			},
			{
				Name: "year=2023/month=10/day=01/hour=17/minute=05/blobtraces_7890.json",
				Size: 5,
			},
		}

		targetBlob := returnedBlobInfo[0]

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		mockClient.EXPECT().ListBlobs(mock.Anything, cfg.Container, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})

		// Create new receiver
		testConsumer := &consumertest.TracesSink{}
		r, err := newTracesReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		checkFunc := func() bool {
			return testConsumer.SpanCount() == traces.SpanCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("logs", func(t *testing.T) {
		// Test data
		logs, jsonBytes := generateTestLogs(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*azureblob.BlobInfo{
			{
				Name: "year=2023/month=10/day=02/hour=17/minute=05/bloblogs_12345.json",
				Size: expectedBuffSize,
			},
			{
				Name: "year=2023/month=10/day=01/hour=17/minute=05/bloblogs_7890.json",
				Size: 5,
			},
		}

		targetBlob := returnedBlobInfo[0]

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		mockClient.EXPECT().ListBlobs(mock.Anything, cfg.Container, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})

		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("gzip compression", func(t *testing.T) {
		// Test data
		logs, jsonBytes := generateTestLogs(t)
		compressedBytes := gzipCompressData(t, jsonBytes)
		expectedBuffSize := int64(len(compressedBytes))

		returnedBlobInfo := []*azureblob.BlobInfo{
			{
				Name: "year=2023/month=10/day=02/hour=17/minute=05/bloblogs_12345.json.gz",
				Size: expectedBuffSize,
			},
			{
				Name: "year=2023/month=10/day=01/hour=17/minute=05/bloblogs_7890.json.gz",
				Size: 5,
			},
		}

		targetBlob := returnedBlobInfo[0]

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		mockClient.EXPECT().ListBlobs(mock.Anything, cfg.Container, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, compressedBytes)

			return expectedBuffSize, nil
		})

		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("Delete on Read", func(t *testing.T) {
		cfg.DeleteOnRead = true
		t.Cleanup(func() {
			cfg.DeleteOnRead = false
		})

		// Test data
		logs, jsonBytes := generateTestLogs(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*azureblob.BlobInfo{
			{
				Name: "year=2023/month=10/day=02/hour=17/minute=05/bloblogs_12345.json",
				Size: expectedBuffSize,
			},
			{
				Name: "year=2023/month=10/day=01/hour=17/minute=05/bloblogs_7890.json",
				Size: 5,
			},
		}

		targetBlob := returnedBlobInfo[0]

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		mockClient.EXPECT().ListBlobs(mock.Anything, cfg.Container, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})
		mockClient.EXPECT().DeleteBlob(mock.Anything, cfg.Container, targetBlob.Name).Return(nil)

		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	// This tests verifies all blobs supplied paths are not attempted to be rehydrated.
	t.Run("Skip parsing out of range or invalid paths", func(t *testing.T) {
		// Test data
		logs, jsonBytes := generateTestLogs(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*azureblob.BlobInfo{
			{
				Name: "year=2022/month=10/day=02/hour=17/minute=05/bloblogs_12345.json", // Out of time range
			},
			{
				Name: "year=nope/month=10/day=02/hour=17/minute=05/bloblogs_12345.json", // Bad time parsing
			},
			{
				Name: "bloblogs_7890.json", // Invalid path
			},
			{
				Name: "year=2023/month=10/day=02/hour=17/minute=05/bloblogs_12345.json", // blobs are processed in order so adding a good one at the end to test when we are done
				Size: expectedBuffSize,
			},
		}

		targetBlob := returnedBlobInfo[3]

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		mockClient.EXPECT().ListBlobs(mock.Anything, cfg.Container, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})

		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})
}

func Test_parseBlobPath(t *testing.T) {
	expectedTimeMinute := time.Date(2023, time.January, 04, 12, 02, 0, 0, time.UTC)
	expectedTimeHour := time.Date(2023, time.January, 04, 12, 00, 0, 0, time.UTC)

	testcases := []struct {
		desc         string
		blobName     string
		expectedTime *time.Time
		expectedType component.DataType
		expectedErr  error
	}{
		{
			desc:         "Empty BlobName",
			blobName:     "",
			expectedTime: nil,
			expectedType: component.Type{},
			expectedErr:  errInvalidBlobPath,
		},
		{
			desc:         "Malformed path",
			blobName:     "year=2023/day=04/hour=12/minute=02/blobmetrics_12345.json",
			expectedTime: nil,
			expectedType: component.Type{},
			expectedErr:  errInvalidBlobPath,
		},
		{
			desc:         "Malformed timestamp",
			blobName:     "year=2003/month=00/day=04/hour=12/minute=01/blobmetrics_12345.json",
			expectedTime: nil,
			expectedType: component.Type{},
			expectedErr:  errors.New("parse blob time"),
		},
		{
			desc:         "Prefix, minute, metrics",
			blobName:     "prefix/year=2023/month=01/day=04/hour=12/minute=02/blobmetrics_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: component.DataTypeMetrics,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, minute, metrics",
			blobName:     "year=2023/month=01/day=04/hour=12/minute=02/blobmetrics_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: component.DataTypeMetrics,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, minute, logs",
			blobName:     "year=2023/month=01/day=04/hour=12/minute=02/bloblogs_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: component.DataTypeLogs,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, minute, traces",
			blobName:     "year=2023/month=01/day=04/hour=12/minute=02/blobtraces_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: component.DataTypeTraces,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, hour, metrics",
			blobName:     "year=2023/month=01/day=04/hour=12/blobmetrics_12345.json",
			expectedTime: &expectedTimeHour,
			expectedType: component.DataTypeMetrics,
			expectedErr:  nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			actualTime, actualType, err := parseBlobPath(tc.blobName)
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, tc.expectedTime)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actualTime)
				require.True(t, tc.expectedTime.Equal(*actualTime))
				require.Equal(t, tc.expectedType, actualType)
			}
		})
	}
}

func Test_processBlob(t *testing.T) {
	containerName := "container"

	// Tests jsonData to return for mock jsonData
	jsonData := []byte(`{"one": "two"}`)
	gzipData := gzipCompressData(t, jsonData)

	testcases := []struct {
		desc        string
		info        *azureblob.BlobInfo
		mockSetup   func(*blobmocks.MockBlobClient, *mocks.MockBlobConsumer)
		expectedErr error
	}{
		{
			desc: "Download blob error",
			info: &azureblob.BlobInfo{
				Name: "blob.json",
				Size: 10,
			},
			mockSetup: func(mockClient *blobmocks.MockBlobClient, _ *mocks.MockBlobConsumer) {
				mockClient.EXPECT().DownloadBlob(mock.Anything, containerName, "blob.json", mock.Anything).Return(0, errors.New("bad"))
			},
			expectedErr: errors.New("download blob: bad"),
		},
		{
			desc: "unsupported extension",
			info: &azureblob.BlobInfo{
				Name: "blob.nope",
				Size: 10,
			},
			mockSetup: func(mockClient *blobmocks.MockBlobClient, _ *mocks.MockBlobConsumer) {
				mockClient.EXPECT().DownloadBlob(mock.Anything, containerName, "blob.nope", mock.Anything).Return(0, nil)
			},
			expectedErr: errors.New("unsupported file type: .nope"),
		},
		{
			desc: "Gzip compression",
			info: &azureblob.BlobInfo{
				Name: "blob.json.gz",
				Size: int64(len(gzipData)),
			},
			mockSetup: func(mockClient *blobmocks.MockBlobClient, mockConsumer *mocks.MockBlobConsumer) {
				mockClient.EXPECT().DownloadBlob(mock.Anything, containerName, "blob.json.gz", mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
					copy(buf, gzipData)
					return int64(len(gzipData)), nil
				})

				mockConsumer.EXPECT().Consume(mock.Anything, jsonData).Return(nil)
			},
			expectedErr: nil,
		},
		{
			desc: "Json no compression",
			info: &azureblob.BlobInfo{
				Name: "blob.json",
				Size: int64(len(jsonData)),
			},
			mockSetup: func(mockClient *blobmocks.MockBlobClient, mockConsumer *mocks.MockBlobConsumer) {
				mockClient.EXPECT().DownloadBlob(mock.Anything, containerName, "blob.json", mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
					copy(buf, jsonData)
					return int64(len(jsonData)), nil
				})

				mockConsumer.EXPECT().Consume(mock.Anything, jsonData).Return(nil)
			},
			expectedErr: nil,
		},
		{
			desc: "Consume error",
			info: &azureblob.BlobInfo{
				Name: "blob.json",
				Size: int64(len(jsonData)),
			},
			mockSetup: func(mockClient *blobmocks.MockBlobClient, mockConsumer *mocks.MockBlobConsumer) {
				mockClient.EXPECT().DownloadBlob(mock.Anything, containerName, "blob.json", mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
					copy(buf, jsonData)
					return int64(len(jsonData)), nil
				})

				mockConsumer.EXPECT().Consume(mock.Anything, jsonData).Return(errors.New("bad"))
			},
			expectedErr: errors.New("consume: bad"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			mockClient := blobmocks.NewMockBlobClient(t)
			mockConsumer := mocks.NewMockBlobConsumer(t)

			tc.mockSetup(mockClient, mockConsumer)

			r := &rehydrationReceiver{
				logger: zap.NewNop(),
				cfg: &Config{
					Container: containerName,
				},
				consumer:    mockConsumer,
				azureClient: mockClient,
				ctx:         context.Background(),
			}

			err := r.processBlob(tc.info)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

// setNewAzureBlobClient helper function used to set the newAzureBlobClient
// function with a mock and return the mock.
func setNewAzureBlobClient(t *testing.T) *blobmocks.MockBlobClient {
	t.Helper()
	oldfunc := newAzureBlobClient

	mockClient := blobmocks.NewMockBlobClient(t)

	newAzureBlobClient = func(_ string) (azureblob.BlobClient, error) {
		return mockClient, nil
	}

	t.Cleanup(func() {
		newAzureBlobClient = oldfunc
	})

	return mockClient
}

// runRehydrationValidateTest runs the rehydration tests with the passed in checkFunc
func runRehydrationValidateTest(t *testing.T, r *rehydrationReceiver, checkFunc func() bool) {
	// Start the receiver
	err := r.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Wait for telemetry to be consumed
	require.Eventually(t, checkFunc, time.Second, 10*time.Millisecond)

	// Shutdown receivers
	err = r.Shutdown(context.Background())
	require.NoError(t, err)
}

// gzipCompressData compresses data for testing
func gzipCompressData(t *testing.T, input []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(input)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	return buf.Bytes()
}
