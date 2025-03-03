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
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	defer fake.SetFakeConstructorForTest()()

	ctx := context.Background()
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	cfg.PollInterval = 10 * time.Millisecond
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	require.NoError(t, receiver.Start(ctx, host))
	require.NoError(t, receiver.Shutdown(ctx))
}

func TestReceiver(t *testing.T) {
	defer fake.SetFakeConstructorForTest()()

	testCases := []struct {
		name       string
		objectSets []map[string]map[string]string
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
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			fakeAWS := fake.NewClient(aws.Config{}).(*fake.AWS)

			var numObjects int
			for _, objectSet := range testCase.objectSets {
				for _, bucket := range objectSet {
					numObjects += len(bucket)
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			set := receivertest.NewNopSettings()
			f := rcvr.NewFactory()
			cfg := f.CreateDefaultConfig().(*rcvr.Config)
			cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
			cfg.PollInterval = 10 * time.Millisecond
			sink := new(consumertest.LogsSink)

			receiver, err := f.CreateLogs(context.Background(), set, cfg, sink)
			require.NoError(t, err)
			require.NotNil(t, receiver)

			host := componenttest.NewNopHost()
			require.NoError(t, receiver.Start(ctx, host))

			defer func() {
				require.NoError(t, receiver.Shutdown(ctx))
			}()

			require.Eventually(t, func() bool {
				return len(sink.AllLogs()) == len(testCase.objectSets)
			}, time.Second, 100*time.Millisecond)

			var numRecords int
			for _, logs := range sink.AllLogs() {
				numRecords += logs.LogRecordCount()
			}
			require.Equal(t, numObjects, numRecords)

			_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.Equal(t, fake.ErrEmptyQueue, err)
		})
	}
}
