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

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	rcvr "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/fake"
)

func TestNewS3EventReceiver(t *testing.T) {
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestNewS3EventReceiverValidationError(t *testing.T) {
	set := receivertest.NewNopSettings()
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
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()

	t.Run("valid SQS URL", func(t *testing.T) {
		cfg := f.CreateDefaultConfig().(*rcvr.Config)
		cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
		next := consumertest.NewNop()

		receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
		require.NoError(t, err)
		assert.NotNil(t, receiver)

		// Verify the region was extracted correctly
		region, err := bpaws.ParseRegionFromSQSURL(cfg.SQSQueueURL)
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
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	cfg.StandardPollInterval = 10 * time.Millisecond
	next := consumertest.NewNop()

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
			ctx := context.Background()
			fakeAWS := fake.NewClient(t).(*fake.AWS)

			var numObjects int
			for _, objectSet := range tc.objectSets {
				for _, bucket := range objectSet {
					numObjects += len(bucket)
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			set := receivertest.NewNopSettings()
			f := rcvr.NewFactory()
			cfg := f.CreateDefaultConfig().(*rcvr.Config)
			cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
			cfg.StandardPollInterval = 50 * time.Millisecond
			sink := new(consumertest.LogsSink)

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

	ctx := context.Background()
	fakeAWS := fake.NewClient(t).(*fake.AWS)

	numBuckets := 10
	numObjectsPerBucket := 100
	numObjectsPerMessage := 4 // Test assumes this divides evenly into numObjectsPerBucket

	// Create standard multiline log formats
	sampleLogs := []string{
		"line1\nline2\nline3",
		"line1\nline2\nline3\nline4\nline5",
		"line1\nline2",
		"line1\nline2\nline3\nline4",
	}

	// Create buckets of objects and batches
	var allBatches []map[string]map[string]string
	var expectLines int

	for b := 0; b < numBuckets; b++ {
		bucketName := fmt.Sprintf("bucket%d", b)

		for o := 0; o < numObjectsPerBucket; o += numObjectsPerMessage {
			// Create a new batch of objects
			batchObjects := make(map[string]map[string]string)
			batchObjects[bucketName] = make(map[string]string)

			for i := 0; i < numObjectsPerMessage; i++ {
				// Log contents are just the sample logs repeated
				key := fmt.Sprintf("object-%d-%d", o, i)
				log := sampleLogs[i%len(sampleLogs)]
				batchObjects[bucketName][key] = log
				expectLines += strings.Count(log, "\n") + 1
			}

			// Add this batch to our collection
			allBatches = append(allBatches, batchObjects)
		}
	}

	// Process all but the last batch
	for i := 0; i < len(allBatches)-1; i++ {
		fakeAWS.CreateObjects(t, allBatches[i])
	}

	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	cfg.StandardPollInterval = 50 * time.Millisecond
	sink := new(consumertest.LogsSink)

	receiver, err := f.CreateLogs(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, receiver)

	host := componenttest.NewNopHost()
	require.NoError(t, receiver.Start(ctx, host))

	// Send the last batch after starting the receiver
	lastBatch := allBatches[len(allBatches)-1]
	fakeAWS.CreateObjects(t, lastBatch)

	defer func() {
		require.NoError(t, receiver.Shutdown(ctx))
	}()

	// Calculate total objects across all batches
	var totalObjects int
	for _, batch := range allBatches {
		for _, bucketObjs := range batch {
			totalObjects += len(bucketObjs)
		}
	}

	// Wait for all log batches to be processed
	require.Eventually(t, func() bool {
		return len(sink.AllLogs()) == totalObjects
	}, 10*time.Second, 100*time.Millisecond)

	// Count actual records and compare to expected
	var numRecords int
	for _, logs := range sink.AllLogs() {
		numRecords += logs.LogRecordCount()
	}
	require.Equal(t, expectLines, numRecords)

	_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.Equal(t, fake.ErrEmptyQueue, err)
}
