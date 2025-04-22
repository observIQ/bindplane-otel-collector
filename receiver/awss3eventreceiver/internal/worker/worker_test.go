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
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"

	"github.com/observiq/bindplane-otel-collector/internal/aws/fake"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
)

func TestProcessMessage(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	longLineLength := 100000
	maxLogSize := 4096
	maxLogsEmitted := 1000

	// Calculate expected fragments for the long line
	// Need to use ceiling division to handle any remainder correctly
	longLine := createLongLine(longLineLength)
	expectedLongLineFragments := (longLineLength + maxLogSize - 1) / maxLogSize

	testCases := []struct {
		name        string
		objectSets  []map[string]map[string]string
		expectLines int
	}{
		{
			name: "single object - single line",
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
			name: "single object - multiple lines",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2\nline3",
					},
				},
			},
			expectLines: 3,
		},
		{
			name: "multiple objects with multiple lines",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2\nline3",
						"mykey2": "line1\nline2",
					},
					"mybucket2": {
						"mykey3": "line1\nline2\nline3\nline4",
						"mykey4": "line1",
					},
				},
			},
			expectLines: 10,
		},
		{
			name: "objects with empty lines",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\n\nline3",
					},
				},
			},
			expectLines: 2,
		},
		{
			name: "objects with trailing newlines",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2\n",
					},
				},
			},
			expectLines: 2,
		},
		{
			name: "object with very long line",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": longLine,
					},
				},
			},
			expectLines: expectedLongLineFragments,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			fakeAWS := fake.NewClient(t).(*fake.AWS)

			var totalObjects int
			for _, objectSet := range testCase.objectSets {
				for _, bucket := range objectSet {
					totalObjects += len(bucket)
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			set := componenttest.NewNopTelemetrySettings()
			sink := new(consumertest.LogsSink)
			w := worker.New(set, aws.Config{}, sink, maxLogSize, maxLogsEmitted)

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
			require.Equal(t, totalObjects, len(sink.AllLogs()), "Expected %d log batches (one per object)", totalObjects)

			var numRecords int
			for _, logs := range sink.AllLogs() {
				numRecords += logs.LogRecordCount()
			}
			require.Equal(t, testCase.expectLines, numRecords)

			_, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.Equal(t, fake.ErrEmptyQueue, err)
		})
	}
}

func createLongLine(length int) string {
	builder := strings.Builder{}
	builder.Grow(length)
	pattern := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for builder.Len() < length {
		builder.WriteString(pattern)
	}
	return builder.String()[:length]
}

func TestEventTypeFiltering(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	maxLogSize := 4096
	maxLogsEmitted := 1000

	testCases := []struct {
		name        string
		eventType   string
		objectSets  []map[string]map[string]string
		expectLines int
		expectLogs  int
	}{
		{
			name:      "s3:ObjectCreated:Put - should be processed",
			eventType: "s3:ObjectCreated:Put",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2",
					},
				},
			},
			expectLines: 2,
			expectLogs:  1,
		},
		{
			name:      "s3:ObjectCreated:CompleteMultipartUpload - should be processed",
			eventType: "s3:ObjectCreated:CompleteMultipartUpload",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2\nline3",
					},
				},
			},
			expectLines: 3,
			expectLogs:  1,
		},
		{
			name:      "s3:ObjectRemoved:Delete - should not be processed",
			eventType: "s3:ObjectRemoved:Delete",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2",
					},
				},
			},
			expectLines: 0,
			expectLogs:  0,
		},
		{
			name:      "s3:ReducedRedundancyLostObject - should not be processed",
			eventType: "s3:ReducedRedundancyLostObject",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2",
					},
				},
			},
			expectLines: 0,
			expectLogs:  0,
		},
		{
			name:      "s3:Replication - should not be processed",
			eventType: "s3:Replication:OperationCompletedReplication",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "line1\nline2",
					},
				},
			},
			expectLines: 0,
			expectLogs:  0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			fakeAWS := fake.NewClient(t).(*fake.AWS)

			for _, objectSet := range testCase.objectSets {
				fakeAWS.CreateObjectsWithEventType(t, testCase.eventType, objectSet)
			}

			set := componenttest.NewNopTelemetrySettings()
			sink := new(consumertest.LogsSink)
			w := worker.New(set, aws.Config{}, sink, maxLogSize, maxLogsEmitted)

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

			if testCase.expectLogs == 0 {
				require.Empty(t, sink.AllLogs())
			} else {
				require.Equal(t, testCase.expectLogs, len(sink.AllLogs()))
			}

			var numRecords int
			for _, logs := range sink.AllLogs() {
				numRecords += logs.LogRecordCount()
			}
			require.Equal(t, testCase.expectLines, numRecords)

			_, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.Equal(t, fake.ErrEmptyQueue, err)
		})
	}
}
