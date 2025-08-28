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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver"

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/internal/testutils"
	"github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver/internal/azureblob"
	blobmocks "github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver/internal/azureblob/mocks"
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
	require.Equal(t, pipeline.SignalMetrics, r.supportedTelemetry)
	require.IsType(t, &rehydration.MetricsConsumer{}, r.consumer)
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
	require.Equal(t, pipeline.SignalLogs, r.supportedTelemetry)
	require.IsType(t, &rehydration.LogsConsumer{}, r.consumer)
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
	require.Equal(t, pipeline.SignalTraces, r.supportedTelemetry)
	require.IsType(t, &rehydration.TracesConsumer{}, r.consumer)
}

func Test_fullRehydration(t *testing.T) {
	testType, err := component.NewType("test")
	require.NoError(t, err)

	id := component.NewID(testType)
	testLogger := zap.NewNop()
	cfg := &Config{
		StartingTime: "2023-10-02T17:00",
		EndingTime:   "2023-10-02T18:00",
		Container:    "container",
		DeleteOnRead: false,
	}

	t.Run("metrics", func(t *testing.T) {
		// Test data
		metrics, jsonBytes := testutils.GenerateTestMetrics(t)
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

		// Create new receiver

		targetBlob := returnedBlobInfo[0]

		// Setup mocks
		mockClient := setNewAzureBlobClient(t)
		testConsumer := &consumertest.MetricsSink{}
		r, err := newMetricsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		mockClient.EXPECT().StreamBlobs(mock.Anything, cfg.Container, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().After(time.Millisecond).Run(func(_ mock.Arguments) {
			r.blobChan <- returnedBlobInfo
		})

		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))
			copy(buf, jsonBytes)
			return expectedBuffSize, nil
		})

		checkFunc := func() bool {
			return testConsumer.DataPointCount() == metrics.DataPointCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("traces", func(t *testing.T) {
		// Test data
		traces, jsonBytes := testutils.GenerateTestTraces(t)
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
		mockClient := setNewAzureBlobClient(t)

		// Create new receiver
		testConsumer := &consumertest.TracesSink{}
		r, err := newTracesReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		// Setup mocks
		mockClient.EXPECT().StreamBlobs(mock.Anything, cfg.Container, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().After(time.Millisecond).Run(func(_ mock.Arguments) {
			r.blobChan <- returnedBlobInfo
		})
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))
			copy(buf, jsonBytes)
			return expectedBuffSize, nil
		})

		checkFunc := func() bool {
			return testConsumer.SpanCount() == traces.SpanCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("logs", func(t *testing.T) {
		// Test data
		logs, jsonBytes := testutils.GenerateTestLogs(t)
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
		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		mockClient.EXPECT().StreamBlobs(mock.Anything, cfg.Container, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().After(time.Millisecond).Run(func(_ mock.Arguments) {
			r.blobChan <- returnedBlobInfo
		})
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("gzip compression", func(t *testing.T) {
		// Test data
		logs, jsonBytes := testutils.GenerateTestLogs(t)
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
		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		mockClient.EXPECT().StreamBlobs(mock.Anything, cfg.Container, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().After(time.Millisecond).Run(func(_ mock.Arguments) {
			r.blobChan <- returnedBlobInfo
		})
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, compressedBytes)

			return expectedBuffSize, nil
		})

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	t.Run("Delete on Read", func(t *testing.T) {
		deleteCfg := &Config{
			StartingTime: cfg.StartingTime,
			EndingTime:   cfg.EndingTime,
			Container:    cfg.Container,
			DeleteOnRead: true,
		}

		// Test data
		logs, jsonBytes := testutils.GenerateTestLogs(t)
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
		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, deleteCfg, testConsumer)
		require.NoError(t, err)

		mockClient.EXPECT().StreamBlobs(mock.Anything, cfg.Container, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().After(time.Millisecond).Run(func(_ mock.Arguments) {
			r.blobChan <- returnedBlobInfo
		})
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})

		mockClient.EXPECT().DeleteBlob(mock.Anything, cfg.Container, targetBlob.Name).Return(nil)

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})

	// This tests verifies all blobs supplied paths are not attempted to be rehydrated.
	t.Run("Skip parsing out of range or invalid paths", func(t *testing.T) {
		// Test data
		logs, jsonBytes := testutils.GenerateTestLogs(t)
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

		// Create new receiver
		testConsumer := &consumertest.LogsSink{}
		r, err := newLogsReceiver(id, testLogger, cfg, testConsumer)
		require.NoError(t, err)

		mockClient.EXPECT().StreamBlobs(mock.Anything, cfg.Container, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().After(time.Millisecond).Run(func(_ mock.Arguments) {
			r.blobChan <- returnedBlobInfo
		})
		mockClient.EXPECT().DownloadBlob(mock.Anything, cfg.Container, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})

		checkFunc := func() bool {
			return testConsumer.LogRecordCount() == logs.LogRecordCount()
		}

		runRehydrationValidateTest(t, r, checkFunc)
	})
}

func Test_processBlob(t *testing.T) {
	containerName := "container"

	// Tests jsonData to return for mock jsonData
	jsonData := []byte(`{"one": "two"}`)
	gzipData := gzipCompressData(t, jsonData)

	testcases := []struct {
		desc        string
		info        *azureblob.BlobInfo
		mockSetup   func(*blobmocks.MockBlobClient, *rehydration.MockConsumer)
		expectedErr error
	}{
		{
			desc: "Download blob error",
			info: &azureblob.BlobInfo{
				Name: "blob.json",
				Size: 10,
			},
			mockSetup: func(mockClient *blobmocks.MockBlobClient, _ *rehydration.MockConsumer) {
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
			mockSetup: func(mockClient *blobmocks.MockBlobClient, _ *rehydration.MockConsumer) {
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
			mockSetup: func(mockClient *blobmocks.MockBlobClient, mockConsumer *rehydration.MockConsumer) {
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
			mockSetup: func(mockClient *blobmocks.MockBlobClient, mockConsumer *rehydration.MockConsumer) {
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
			mockSetup: func(mockClient *blobmocks.MockBlobClient, mockConsumer *rehydration.MockConsumer) {
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
			mockConsumer := rehydration.NewMockConsumer(t)

			tc.mockSetup(mockClient, mockConsumer)

			r := &rehydrationReceiver{
				logger: zap.NewNop(),
				cfg: &Config{
					Container: containerName,
				},
				consumer:    mockConsumer,
				azureClient: mockClient,
			}

			err := r.processBlob(context.Background(), tc.info)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestLogsDeprecationWarnings(t *testing.T) {
	mockClient := setNewAzureBlobClient(t)

	testLogger, ol := observer.New(zap.WarnLevel)

	r := &rehydrationReceiver{
		logger: zap.New(testLogger),
		cfg: &Config{
			StartingTime: "2023-10-02T17:00",
			EndingTime:   "2023-10-02T17:01",
			PollInterval: 1 * time.Second,
			PollTimeout:  1 * time.Second,
		},
		azureClient:     mockClient,
		blobChan:        make(chan []*azureblob.BlobInfo),
		errChan:         make(chan error),
		doneChan:        make(chan struct{}),
		mut:             &sync.Mutex{},
		checkpointStore: storageclient.NewNopStorage(),
	}
	mockClient.EXPECT().StreamBlobs(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).After(time.Millisecond).Run(func(_ mock.Arguments) {
		close(r.doneChan)
	})

	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))

	require.Eventually(t, func() bool {
		foundBothLogs := false
		foundPollInterval := false
		for _, log := range ol.All() {
			if strings.Contains(log.Message, "poll_interval is no longer recognized and will be removed in a future release. batch_size/page_size should be used instead") {
				foundBothLogs = true
			}
			if strings.Contains(log.Message, "poll_interval is no longer recognized and will be removed in a future release. batch_size/page_size should be used instead") {
				foundPollInterval = true
			}
		}
		return foundBothLogs && foundPollInterval
	}, 10*time.Second, 1*time.Second)
}

// setNewAzureBlobClient helper function used to set the newAzureBlobClient
// function with a mock and return the mock.
func setNewAzureBlobClient(t *testing.T) *blobmocks.MockBlobClient {
	t.Helper()
	oldfunc := newAzureBlobClient

	mockClient := blobmocks.NewMockBlobClient(t)

	newAzureBlobClient = func(_ string, _ int, _ int) (azureblob.BlobClient, error) {
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
