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

package awss3eventreceiver_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"github.com/observiq/bindplane-otel-collector/internal/aws/client"

	"github.com/observiq/bindplane-otel-collector/internal/aws/fake"
	rcvr "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/metadatatest"
)

func TestNewS3EventReceiver(t *testing.T) {
	set := receivertest.NewNopSettings(metadata.Type)
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestNewS3EventReceiverValidationError(t *testing.T) {
	set := receivertest.NewNopSettings(metadata.Type)
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://invalid-url"
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.Error(t, err)
	assert.Nil(t, receiver)
	assert.Contains(t, err.Error(), "invalid SQS URL format")
}

func TestRegionExtractionFromSQSURL(t *testing.T) {
	set := receivertest.NewNopSettings(metadata.Type)
	f := rcvr.NewFactory()

	t.Run("valid SQS URL", func(t *testing.T) {
		cfg := f.CreateDefaultConfig().(*rcvr.Config)
		cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
		next := consumertest.NewNop()

		receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
		require.NoError(t, err)
		assert.NotNil(t, receiver)

		// Verify the region was extracted correctly
		region, err := client.ParseRegionFromSQSURL(cfg.SQSQueueURL)
		assert.NoError(t, err)
		assert.Equal(t, "us-west-2", region)
	})

	t.Run("invalid SQS URL", func(t *testing.T) {
		cfg := f.CreateDefaultConfig().(*rcvr.Config)
		cfg.SQSQueueURL = "https://invalid-url"
		next := consumertest.NewNop()

		receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
		require.Error(t, err)
		assert.Nil(t, receiver)
		assert.Contains(t, err.Error(), "invalid SQS URL format")
	})

}

func TestStartShutdown(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	ctx := context.Background()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	cfg.StandardPollInterval = 10 * time.Millisecond
	next := consumertest.NewNop()

	set := receivertest.NewNopSettings(metadata.Type)
	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	require.NoError(t, receiver.Start(ctx, host))
	require.NoError(t, receiver.Shutdown(ctx))
}

func TestReceiver(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	testCases := []struct {
		name        string
		objectSets  []map[string]map[string]string
		expectLines int
	}{
		{
			name: "single object",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
			expectLines: 1,
		},
		{
			name: "multiple objects",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
						"mykey2": "myvalue2",
					},
					"mybucket2": {
						"mykey3": "myvalue3",
						"mykey4": "myvalue4",
					},
				},
			},
			expectLines: 4,
		},
		{
			name: "multiple objects with different buckets",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1",
						"mykey2": "myvalue2",
					},
					"mybucket2": {
						"mykey3": "myvalue3",
						"mykey4": "myvalue4",
					},
				},
			},
			expectLines: 4,
		},
		{
			name: "multiple objects multiple lines",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1\nmyvalue2",
						"mykey2": "myvalue3\nmyvalue4",
					},
				},
			},
			expectLines: 4,
		},
		{
			name: "multiple objects multiple lines with different buckets",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1\nmyvalue2",
						"mykey2": "myvalue3\nmyvalue4",
					},
					"mybucket2": {
						"mykey3": "myvalue5\nmyvalue6",
						"mykey4": "myvalue7\nmyvalue8",
					},
				},
			},
			expectLines: 8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

			var numObjects int
			for _, objectSet := range tc.objectSets {
				for _, bucket := range objectSet {
					numObjects += len(bucket)
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			f := rcvr.NewFactory()
			cfg := f.CreateDefaultConfig().(*rcvr.Config)
			cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
			cfg.StandardPollInterval = 50 * time.Millisecond
			sink := new(consumertest.LogsSink)

			set := receivertest.NewNopSettings(metadata.Type)
			receiver, err := f.CreateLogs(context.Background(), set, cfg, sink)
			require.NoError(t, err)
			require.NotNil(t, receiver)

			host := componenttest.NewNopHost()
			require.NoError(t, receiver.Start(ctx, host))

			defer func() {
				require.NoError(t, receiver.Shutdown(ctx))
			}()

			var totalObjects int
			for _, objSet := range tc.objectSets {
				for _, bucket := range objSet {
					totalObjects += len(bucket)
				}
			}

			require.Eventually(t, func() bool {
				return len(sink.AllLogs()) == totalObjects
			}, time.Second, 100*time.Millisecond)

			var numRecords int
			for _, logs := range sink.AllLogs() {
				numRecords += logs.LogRecordCount()
			}
			require.Equal(t, tc.expectLines, numRecords)

			_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.Equal(t, fake.ErrEmptyQueue, err)
		})
	}
}

func TestManyObjects(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	ctx := t.Context()
	fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

	numBuckets := 10
	numObjectsPerBucket := 100

	logContent := "line1\nline2\nline3"
	linesPerObject := strings.Count(logContent, "\n") + 1

	objects := make(map[string]map[string]string)
	for b := 0; b < numBuckets; b++ {
		bucketName := fmt.Sprintf("bucket%d", b)
		objects[bucketName] = make(map[string]string)

		for o := 0; o < numObjectsPerBucket; o++ {
			key := fmt.Sprintf("object-%d", o)
			objects[bucketName][key] = logContent
		}
	}

	fakeAWS.CreateObjects(t, objects)

	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	cfg.StandardPollInterval = 50 * time.Millisecond
	sink := new(consumertest.LogsSink)

	set := receivertest.NewNopSettings(metadata.Type)
	receiver, err := f.CreateLogs(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, receiver)

	host := componenttest.NewNopHost()
	require.NoError(t, receiver.Start(ctx, host))

	defer func() {
		require.NoError(t, receiver.Shutdown(ctx))
	}()

	var totalObjects int
	for _, bucketObjs := range objects {
		totalObjects += len(bucketObjs)
	}

	require.Eventually(t, func() bool {
		return len(sink.AllLogs()) == totalObjects
	}, 10*time.Second, 100*time.Millisecond)

	var numRecords int
	for _, logs := range sink.AllLogs() {
		numRecords += logs.LogRecordCount()
	}

	// Calculate expected total lines
	expectedTotalLines := totalObjects * linesPerObject
	require.Equal(t, expectedTotalLines, numRecords)

	_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.Equal(t, fake.ErrEmptyQueue, err)
}

func TestReceiverMetrics(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	testCases := []struct {
		name                   string
		objectSets             []map[string]map[string]string
		expectLines            int
		expectedObjectsHandled int64
		expectedBatchSum       int64
		expectedBatchCount     uint64
		expectedBucketCounts   [15]uint64
		expectedMin            int64
		expectedMax            int64
	}{
		{
			name: "single object",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
			expectLines:            1,
			expectedObjectsHandled: 1,
			expectedBatchSum:       1,
			expectedBatchCount:     1,
			expectedBucketCounts:   [15]uint64{1},
			expectedMin:            1,
			expectedMax:            1,
		},
		{
			name: "multiple objects",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
						"mykey2": "myvalue2",
					},
					"mybucket2": {
						"mykey3": "myvalue3",
						"mykey4": "myvalue4",
					},
				},
			},
			expectLines:            4,
			expectedObjectsHandled: 4,
			expectedBatchSum:       4,
			expectedBatchCount:     4,
			expectedBucketCounts:   [15]uint64{4},
			expectedMin:            1,
			expectedMax:            1,
		},
		{
			name: "multiple objects with different buckets",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1",
						"mykey2": "myvalue2",
					},
					"mybucket2": {
						"mykey3": "myvalue3",
						"mykey4": "myvalue4",
					},
				},
			},
			expectLines:            4,
			expectedObjectsHandled: 4,
			expectedBatchSum:       4,
			expectedBatchCount:     4,
			expectedBucketCounts:   [15]uint64{4},
			expectedMin:            1,
			expectedMax:            1,
		},
		{
			name: "multiple objects multiple lines",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1\nmyvalue2",
						"mykey2": "myvalue3\nmyvalue4",
					},
				},
			},
			expectLines:            4,
			expectedObjectsHandled: 2,
			expectedBatchSum:       4,
			expectedBatchCount:     2,
			expectedBucketCounts:   [15]uint64{0, 2},
			expectedMin:            2,
			expectedMax:            2,
		},
		{
			name: "multiple objects multiple lines with different buckets",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1\nmyvalue2",
						"mykey2": "myvalue3\nmyvalue4",
					},
					"mybucket2": {
						"mykey3": "myvalue5\nmyvalue6",
						"mykey4": "myvalue7\nmyvalue8",
					},
				},
			},
			expectLines:            8,
			expectedObjectsHandled: 4,
			expectedBatchSum:       8,
			expectedBatchCount:     4,
			expectedBucketCounts:   [15]uint64{0, 4},
			expectedMin:            2,
			expectedMax:            2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

			for _, objectSet := range tc.objectSets {
				fakeAWS.CreateObjects(t, objectSet)
			}

			f := rcvr.NewFactory()
			cfg := f.CreateDefaultConfig().(*rcvr.Config)
			cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
			cfg.StandardPollInterval = 50 * time.Millisecond
			sink := new(consumertest.LogsSink)

			// Set up telemetry testing
			testTel := componenttest.NewTelemetry()
			defer func() {
				require.NoError(t, testTel.Shutdown(ctx))
			}()

			set := metadatatest.NewSettings(testTel)
			receiver, err := f.CreateLogs(context.Background(), set, cfg, sink)
			require.NoError(t, err)
			require.NotNil(t, receiver)

			host := componenttest.NewNopHost()
			require.NoError(t, receiver.Start(ctx, host))

			defer func() {
				require.NoError(t, receiver.Shutdown(ctx))
			}()

			var totalObjects int
			for _, objSet := range tc.objectSets {
				for _, bucket := range objSet {
					totalObjects += len(bucket)
				}
			}

			require.Eventually(t, func() bool {
				return len(sink.AllLogs()) == totalObjects
			}, time.Second, 100*time.Millisecond)

			var numRecords int
			for _, logs := range sink.AllLogs() {
				numRecords += logs.LogRecordCount()
			}
			require.Equal(t, tc.expectLines, numRecords)

			_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.Equal(t, fake.ErrEmptyQueue, err)

			// Test s3event.objects_handled metric
			// This should equal the number of objects processed
			metadatatest.AssertEqualS3eventObjectsHandled(t, testTel,
				[]metricdata.DataPoint[int64]{{Value: tc.expectedObjectsHandled}},
				metricdatatest.IgnoreTimestamp())

			// Test s3event.batch_size metric
			// This should record histogram entries for each batch
			metadatatest.AssertEqualS3eventBatchSize(t, testTel,
				[]metricdata.HistogramDataPoint[int64]{{
					Count:        tc.expectedBatchCount,
					Sum:          tc.expectedBatchSum,
					BucketCounts: tc.expectedBucketCounts[:],
					Bounds:       []float64{1, 5, 10, 100, 250, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000, 1000000},
					Min:          metricdata.NewExtrema(tc.expectedMin),
					Max:          metricdata.NewExtrema(tc.expectedMax),
				}},
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

		})
	}
}
