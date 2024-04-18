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

package awss3rehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver"

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/observiq/bindplane-agent/internal/rehydration"
	"github.com/observiq/bindplane-agent/internal/testutils"
	"github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver/internal/aws"
	"github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver/internal/aws/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
)

func Test_newMetricsReceiver(t *testing.T) {
	mockClient := setNewAWSClient(t)
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
	require.Equal(t, mockClient, r.awsClient)
	require.Equal(t, component.DataTypeMetrics, r.supportedTelemetry)
	require.IsType(t, &rehydration.MetricsConsumer{}, r.consumer)
}

func Test_newLogsReceiver(t *testing.T) {
	mockClient := setNewAWSClient(t)
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
	require.Equal(t, mockClient, r.awsClient)
	require.Equal(t, component.DataTypeLogs, r.supportedTelemetry)
	require.IsType(t, &rehydration.LogsConsumer{}, r.consumer)
}

func Test_newTracesReceiver(t *testing.T) {
	mockClient := setNewAWSClient(t)
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
	require.Equal(t, mockClient, r.awsClient)
	require.Equal(t, component.DataTypeTraces, r.supportedTelemetry)
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
		PollInterval: 10 * time.Millisecond,
		S3Bucket:     "bucket",
		DeleteOnRead: false,
	}

	t.Run("empty blob polling", func(t *testing.T) {
		var listCounter atomic.Int32

		// Setup mocks
		mockClient := setNewAWSClient(t)
		mockClient.EXPECT().ListObjects(mock.Anything, cfg.S3Bucket, (*string)(nil), (*string)(nil)).Times(3).Return([]*aws.ObjectInfo{}, nil, nil).
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
		metrics, jsonBytes := testutils.GenerateTestMetrics(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*aws.ObjectInfo{
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
		mockClient := setNewAWSClient(t)
		mockClient.EXPECT().ListObjects(mock.Anything, cfg.S3Bucket, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadObject(mock.Anything, cfg.S3Bucket, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
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
		traces, jsonBytes := testutils.GenerateTestTraces(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*aws.ObjectInfo{
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
		mockClient := setNewAWSClient(t)
		mockClient.EXPECT().ListObjects(mock.Anything, cfg.S3Bucket, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadObject(mock.Anything, cfg.S3Bucket, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
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
		logs, jsonBytes := testutils.GenerateTestLogs(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*aws.ObjectInfo{
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
		mockClient := setNewAWSClient(t)
		mockClient.EXPECT().ListObjects(mock.Anything, cfg.S3Bucket, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadObject(mock.Anything, cfg.S3Bucket, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
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
		logs, jsonBytes := testutils.GenerateTestLogs(t)
		compressedBytes := testutils.GZipCompressData(t, jsonBytes)
		expectedBuffSize := int64(len(compressedBytes))

		returnedBlobInfo := []*aws.ObjectInfo{
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
		mockClient := setNewAWSClient(t)
		mockClient.EXPECT().ListObjects(mock.Anything, cfg.S3Bucket, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadObject(mock.Anything, cfg.S3Bucket, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
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
		logs, jsonBytes := testutils.GenerateTestLogs(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*aws.ObjectInfo{
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
		mockClient := setNewAWSClient(t)
		mockClient.EXPECT().ListObjects(mock.Anything, cfg.S3Bucket, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadObject(mock.Anything, cfg.S3Bucket, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
			require.Len(t, buf, int(expectedBuffSize))

			copy(buf, jsonBytes)

			return expectedBuffSize, nil
		})
		mockClient.EXPECT().DeleteObjects(mock.Anything, cfg.S3Bucket, []string{targetBlob.Name}).Return(nil)

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
		logs, jsonBytes := testutils.GenerateTestLogs(t)
		expectedBuffSize := int64(len(jsonBytes))

		returnedBlobInfo := []*aws.ObjectInfo{
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
		mockClient := setNewAWSClient(t)
		mockClient.EXPECT().ListObjects(mock.Anything, cfg.S3Bucket, (*string)(nil), (*string)(nil)).Return(returnedBlobInfo, nil, nil)
		mockClient.EXPECT().DownloadObject(mock.Anything, cfg.S3Bucket, targetBlob.Name, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ string, buf []byte) (int64, error) {
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

// setNewAWSClient helper function used to set the newAWSS3Client
// function with a mock and return the mock.
func setNewAWSClient(t *testing.T) *mocks.MockS3Client {
	t.Helper()
	oldfunc := newAWSS3Client

	mockClient := mocks.NewMockS3Client(t)

	newAWSS3Client = func(_, _ string) (aws.S3Client, error) {
		return mockClient, nil
	}

	t.Cleanup(func() {
		newAWSS3Client = oldfunc
	})

	return mockClient
}
