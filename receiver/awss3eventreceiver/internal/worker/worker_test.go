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

// Package worker provides a worker that processes S3 event notifications.
package worker_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"

	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/fake"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
)

func TestProcessMessage(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

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
			fakeAWS := fake.NewClient(t).(*fake.AWS)

			var numObjects int
			for _, objectSet := range testCase.objectSets {
				for _, bucket := range objectSet {
					numObjects += len(bucket)
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			set := componenttest.NewNopTelemetrySettings()
			sink := new(consumertest.LogsSink)
			w := worker.New(set, aws.Config{}, sink)

			numCallbacks := 0

			for {
				msg, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
				if err != nil {
					require.ErrorIs(t, err, fake.ErrEmptyQueue)
					break
				}
				for _, msg := range msg.Messages {
					w.ProcessMessage(ctx, msg, "myqueue", func() {
						numCallbacks++
					})
				}
			}

			require.Equal(t, len(testCase.objectSets), numCallbacks)
			require.Equal(t, len(testCase.objectSets), len(sink.AllLogs()))
			var numRecords int
			for _, logs := range sink.AllLogs() {
				numRecords += logs.LogRecordCount()
			}
			require.Equal(t, numObjects, numRecords)

			// Queue should be empty
			_, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.Equal(t, fake.ErrEmptyQueue, err)
		})
	}
}
