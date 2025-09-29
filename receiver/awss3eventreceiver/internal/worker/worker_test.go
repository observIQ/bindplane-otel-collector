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
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client/mocks"
	"github.com/observiq/bindplane-otel-collector/internal/aws/fake"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
)

func TestURLDecodingInWorker(t *testing.T) {
	tests := []struct {
		name       string
		encodedKey string
		decodedKey string
	}{
		{
			name:       "equals sign encoding",
			encodedKey: "logs/hd%3Dtest/file.txt",
			decodedKey: "logs/hd=test/file.txt",
		},
		{
			name:       "plus sign encoding",
			encodedKey: "logs/test%2Bfile.txt",
			decodedKey: "logs/test+file.txt",
		},
		{
			name:       "space encoding",
			encodedKey: "logs/test%20file.txt",
			decodedKey: "logs/test file.txt",
		},
		{
			name:       "forward slash encoding",
			encodedKey: "logs%2Ftest%2Ffile.txt",
			decodedKey: "logs/test/file.txt",
		},
		{
			name:       "no encoding needed",
			encodedKey: "logs/test/file.txt",
			decodedKey: "logs/test/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer fake.SetFakeConstructorForTest(t)()

			ctx := context.Background()
			core, observedLogs := observer.New(zap.DebugLevel)
			logger := zap.New(core)

			// Create fake AWS client
			fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

			// Create S3 object with the decoded key name (actual S3 object)
			// This simulates the real S3 object existing with the decoded key
			fakeAWS.CreateObjects(t, map[string]map[string]string{
				"test-bucket": {
					tt.decodedKey: "test log line\n",
				},
			})

			// Create consumer
			consumer := &consumertest.LogsSink{}

			// Create worker
			tel := componenttest.NewNopTelemetrySettings()
			tel.Logger = logger

			tb, err := metadata.NewTelemetryBuilder(tel)
			require.NoError(t, err)

			w := worker.New(tel, consumer, fakeAWS, 1024*1024, 1000, 5*time.Minute, 1*time.Minute, 1*time.Hour, worker.WithTelemetryBuilder(tb))

			// Get the message that was created (it will have the decoded key)
			msg, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.NoError(t, err)
			require.Equal(t, 1, len(msg.Messages))

			// Manually modify the message body to contain the encoded key
			// This simulates what happens when S3 event notifications contain URL-encoded keys
			originalBody := *msg.Messages[0].Body
			modifiedBody := strings.ReplaceAll(originalBody, tt.decodedKey, tt.encodedKey)
			msg.Messages[0].Body = &modifiedBody

			// Process the message with the URL-encoded key
			w.ProcessMessage(ctx, msg.Messages[0], "test-queue-url", func() {})

			// Verify logs were consumed
			require.Equal(t, 1, len(consumer.AllLogs()))
			logs := consumer.AllLogs()[0]
			require.Equal(t, 1, logs.LogRecordCount())

			// Verify resource attributes use decoded key
			resourceLogs := logs.ResourceLogs().At(0)
			keyAttr, exists := resourceLogs.Resource().Attributes().Get("aws.s3.key")
			require.True(t, exists)
			require.Equal(t, tt.decodedKey, keyAttr.AsString()) // Should be decoded

			// If URL decoding happened, check that it was logged
			if tt.encodedKey != tt.decodedKey {
				decodingLogs := observedLogs.FilterMessage("URL decoded object key").All()
				require.Equal(t, 1, len(decodingLogs))
				require.Equal(t, tt.encodedKey, decodingLogs[0].ContextMap()["original_key"])
				require.Equal(t, tt.decodedKey, decodingLogs[0].ContextMap()["decoded_key"])
			}
		})
	}
}

func logsFromFile(t *testing.T, filePath string) []map[string]map[string]string {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return []map[string]map[string]string{
		{
			"mybucket": {
				filePath: string(bytes),
			},
		},
	}
}

func TestProcessMessage(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	longLineLength := 100000
	maxLogSize := 4096
	maxLogsEmitted := 1000
	visibilityExtensionInterval := 100 * time.Millisecond

	// Calculate expected fragments for the long line
	// Need to use ceiling division to handle any remainder correctly
	longLine := createLongLine(longLineLength)
	expectedLongLineFragments := (longLineLength + maxLogSize - 1) / maxLogSize

	testCases := []struct {
		name        string
		objectSets  []map[string]map[string]string
		expectLines int
		maxLogSize  int
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
		{
			name:        "parses as JSON and creates 4 log lines from a JSON array in Records",
			objectSets:  logsFromFile(t, "testdata/logs_array_in_records.json"),
			expectLines: 4,
		},
		{
			name:        "attempts to parse as JSON, but fails and creates 294 log lines from text",
			objectSets:  logsFromFile(t, "testdata/logs_array_in_records_after_limit.json"),
			expectLines: 294,
		},
		{
			name:        "parses as JSON and creates 4 log lines from a JSON array",
			objectSets:  logsFromFile(t, "testdata/logs_array.json"),
			expectLines: 4,
		},
		{
			name:        "does not attempt to parse as JSON and creates 4 log lines from text",
			objectSets:  logsFromFile(t, "testdata/json_lines.txt"),
			expectLines: 4,
		},
		{
			name:        "attempts to parse as JSON, but fails and creates 4 log lines from text",
			objectSets:  logsFromFile(t, "testdata/json_lines.json"),
			expectLines: 4,
		},
		{
			name:        "attempts to parse as JSON, but fails after 1 log line",
			objectSets:  logsFromFile(t, "testdata/logs_array_fragment.json"),
			expectLines: 1,
		},
		{
			name:        "does not attempt to parse as JSON and creates 112 log lines",
			objectSets:  logsFromFile(t, "testdata/logs_array_fragment.txt"),
			expectLines: 112,
		},
		{
			name:        "parses as JSON and creates 4 log lines from the Records field ignoring other fields",
			objectSets:  logsFromFile(t, "testdata/logs_array_in_records_one_line.json"),
			expectLines: 4,
		},
		{
			name:        "attempts to parse as JSON, but fails and reads 3 log lines because of maxLogSize",
			objectSets:  logsFromFile(t, "testdata/logs_array_in_records_after_limit_one_line.json"),
			expectLines: 3,
		},
		{
			name:        "attempts to parse as JSON, but fails and reads 1 log line with maxLogSize = 20000",
			objectSets:  logsFromFile(t, "testdata/logs_array_in_records_after_limit_one_line.json"),
			expectLines: 1,
			maxLogSize:  20000,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

			var totalObjects int
			for _, objectSet := range testCase.objectSets {
				for _, bucket := range objectSet {
					totalObjects += len(bucket)
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			if testCase.maxLogSize == 0 {
				testCase.maxLogSize = maxLogSize
			}

			sink := new(consumertest.LogsSink)

			set := componenttest.NewNopTelemetrySettings()

			b, err := metadata.NewTelemetryBuilder(set)
			require.NoError(t, err)

			w := worker.New(set, sink, fakeAWS, testCase.maxLogSize, maxLogsEmitted, visibilityExtensionInterval, 300*time.Second, 6*time.Hour, worker.WithTelemetryBuilder(b))

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

			_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.ErrorIs(t, err, fake.ErrEmptyQueue)
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
	visibilityExtensionInterval := 100 * time.Millisecond

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
			fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

			for _, objectSet := range testCase.objectSets {
				fakeAWS.CreateObjectsWithEventType(t, testCase.eventType, objectSet)
			}

			set := componenttest.NewNopTelemetrySettings()
			sink := new(consumertest.LogsSink)
			b, err := metadata.NewTelemetryBuilder(set)
			require.NoError(t, err)
			w := worker.New(set, sink, fakeAWS, maxLogSize, maxLogsEmitted, visibilityExtensionInterval, 300*time.Second, 6*time.Hour, worker.WithTelemetryBuilder(b))

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

			_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.ErrorIs(t, err, fake.ErrEmptyQueue)
		})
	}
}

func TestMessageVisibilityExtension(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	ctx := context.Background()
	fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

	// Create a test object
	objectSet := map[string]map[string]string{
		"mybucket": {
			"mykey1": "line1\nline2\nline3",
		},
	}
	fakeAWS.CreateObjects(t, objectSet)

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)

	// Use a short extension interval for faster testing
	visibilityExtensionInterval := 50 * time.Millisecond
	visibilityTimeout := 300 * time.Second

	b, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)
	w := worker.New(set, sink, fakeAWS, 4096, 1000, visibilityExtensionInterval, visibilityTimeout, 6*time.Hour, worker.WithTelemetryBuilder(b))

	// Get a message from the queue
	msg, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.NoError(t, err)
	require.Len(t, msg.Messages, 1)

	// Start processing the message
	done := make(chan struct{})
	go func() {
		w.ProcessMessage(ctx, msg.Messages[0], "myqueue", func() {
			close(done)
		})
	}()

	// Wait for a short time to allow visibility extension to occur
	time.Sleep(100 * time.Millisecond)

	// Check that ChangeMessageVisibility was called
	// We can verify this by checking if the message is still invisible
	// (if visibility wasn't extended, it would be visible again)

	// Try to receive the same message again - it should not be available
	// because visibility should have been extended
	_, err2 := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.ErrorIs(t, err2, fake.ErrEmptyQueue, "Message should still be invisible due to visibility extension")

	// Wait for processing to complete
	<-done

	// Now the message should be deleted and not available
	_, err3 := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.ErrorIs(t, err3, fake.ErrEmptyQueue, "Message should be deleted after processing")
}

func TestVisibilityExtensionLogs(t *testing.T) {
	ctx := context.Background()
	mockSQS := &mocks.MockSQSClient{}
	mockS3 := &mocks.MockS3Client{}
	mockClient := &mocks.MockClient{}
	mockClient.EXPECT().SQS().Return(mockSQS)
	mockClient.EXPECT().S3().Return(mockS3)

	// Provide a valid S3 event message
	validS3Event := `{"Records":[{"eventName":"s3:ObjectCreated:Put","s3":{"bucket":{"name":"mybucket"},"object":{"key":"mykey1","size":15}}}]}`

	mockSQS.EXPECT().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput)).Return(&sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{
				Body:          aws.String(validS3Event),
				MessageId:     aws.String("123"),
				ReceiptHandle: aws.String("receipt-handle"),
			},
		},
	}, nil)

	// Mock S3 GetObject to return content after a delay to trigger visibility extension
	mockS3.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
		// Add a delay to simulate processing time and trigger visibility extension
		time.Sleep(10 * time.Millisecond)
		return &s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader("line1\nline2\nline3")),
		}, nil
	})

	mockSQS.EXPECT().DeleteMessage(mock.Anything, mock.Anything).Return(&sqs.DeleteMessageOutput{}, nil)
	mockSQS.EXPECT().ChangeMessageVisibility(mock.Anything, mock.Anything).Return(&sqs.ChangeMessageVisibilityOutput{}, nil)

	// Set up zap observer
	core, recorded := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	set := componenttest.NewNopTelemetrySettings()
	set.Logger = logger

	sink := new(consumertest.LogsSink)
	visibilityExtensionInterval := 1 * time.Millisecond
	visibilityTimeout := 300 * time.Second
	b, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)
	w := worker.New(set, sink, mockClient, 4096, 1000, visibilityExtensionInterval, visibilityTimeout, 6*time.Hour, worker.WithTelemetryBuilder(b))

	msg, err := mockClient.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.NoError(t, err)
	require.Len(t, msg.Messages, 1)

	done := make(chan struct{})
	go func() {
		w.ProcessMessage(ctx, msg.Messages[0], "myqueue", func() { close(done) })
	}()
	// Wait for processing to complete
	<-done

	// Add a small delay to allow visibility extension goroutine to finish logging
	time.Sleep(50 * time.Millisecond)

	t.Logf("recorded logs:")
	for i, entry := range recorded.All() {
		t.Logf("  [%d] %s: %s", i, entry.Level, entry.Message)
	}

	// Check for all expected log messages
	expectedMessages := []string{
		"starting visibility extension monitoring",
		"extending message visibility",
	}

	for _, expectedMsg := range expectedMessages {
		found := false
		for _, entry := range recorded.All() {
			if entry.Message == expectedMsg {
				found = true
				break
			}
		}
		assert.True(t, found, "expected '%s' log message to be present", expectedMsg)
	}
}

func TestExtendToMaxAndStop(t *testing.T) {
	ctx := context.Background()
	mockSQS := &mocks.MockSQSClient{}
	mockS3 := &mocks.MockS3Client{}
	mockClient := &mocks.MockClient{}
	mockClient.EXPECT().SQS().Return(mockSQS)
	mockClient.EXPECT().S3().Return(mockS3)

	// Provide a valid S3 event message
	validS3Event := `{"Records":[{"eventName":"s3:ObjectCreated:Put","s3":{"bucket":{"name":"mybucket"},"object":{"key":"mykey1","size":15}}}]}`

	mockSQS.EXPECT().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput)).Return(&sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{
				Body:          aws.String(validS3Event),
				MessageId:     aws.String("123"),
				ReceiptHandle: aws.String("receipt-handle"),
			},
		},
	}, nil)

	// Mock S3 GetObject to return content after a delay to trigger visibility extension
	mockS3.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
		// Add a delay to simulate processing time and trigger visibility extension
		time.Sleep(100 * time.Millisecond)
		return &s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader("line1\nline2\nline3")),
		}, nil
	})

	mockSQS.EXPECT().DeleteMessage(mock.Anything, mock.Anything).Return(&sqs.DeleteMessageOutput{}, nil)
	mockSQS.EXPECT().ChangeMessageVisibility(mock.Anything, mock.Anything).Return(&sqs.ChangeMessageVisibilityOutput{}, nil)

	// Set up zap observer
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	set := componenttest.NewNopTelemetrySettings()
	set.Logger = logger

	sink := new(consumertest.LogsSink)
	visibilityExtensionInterval := 1 * time.Millisecond
	visibilityTimeout := 5 * time.Millisecond
	maxVisibilityWindow := 100 * time.Millisecond // Short window to trigger max extension

	b, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)
	w := worker.New(set, sink, mockClient, 4096, 1000, visibilityExtensionInterval, visibilityTimeout, maxVisibilityWindow, worker.WithTelemetryBuilder(b))

	msg, err := mockClient.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.NoError(t, err)
	require.Len(t, msg.Messages, 1)

	done := make(chan struct{})
	go func() {
		w.ProcessMessage(ctx, msg.Messages[0], "myqueue", func() { close(done) })
	}()

	// Wait for processing to complete
	<-done

	time.Sleep(50 * time.Millisecond)

	t.Logf("recorded logs:")
	for i, entry := range recorded.All() {
		t.Logf("  [%d] %s: %s", i, entry.Level, entry.Message)
	}

	// Check for the "reaching maximum visibility window" log message
	found := false
	for _, entry := range recorded.All() {
		if strings.Contains(entry.Message, "reaching maximum visibility window, extending to max and stopping") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'reaching maximum visibility window' log message to be present")
}

func TestVisibilityExtensionContextCancellation(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	ctx, cancel := context.WithCancel(context.Background())
	fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

	objectSet := map[string]map[string]string{
		"mybucket": {
			"mykey1": "line1\nline2\nline3",
		},
	}
	fakeAWS.CreateObjects(t, objectSet)

	// Set up zap observer
	core, recorded := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	set := componenttest.NewNopTelemetrySettings()
	set.Logger = logger

	sink := new(consumertest.LogsSink)
	visibilityExtensionInterval := 1 * time.Millisecond
	visibilityTimeout := 300 * time.Second
	b, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)
	w := worker.New(set, sink, fakeAWS, 4096, 1000, visibilityExtensionInterval, visibilityTimeout, 6*time.Hour, worker.WithTelemetryBuilder(b))

	msg, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.NoError(t, err)
	require.Len(t, msg.Messages, 1)

	done := make(chan struct{})
	go func() {
		w.ProcessMessage(ctx, msg.Messages[0], "myqueue", func() { close(done) })
	}()

	// Cancel context after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for processing to complete
	<-done

	// Check for context cancellation log message
	found := false
	for _, entry := range recorded.All() {
		if entry.Message == "visibility extension stopped due to context cancellation" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'visibility extension stopped due to context cancellation' log message to be present")
}

func TestVisibilityExtensionErrorHandling(t *testing.T) {
	ctx := context.Background()
	mockSQS := &mocks.MockSQSClient{}
	mockS3 := &mocks.MockS3Client{}
	mockClient := &mocks.MockClient{}
	mockClient.EXPECT().SQS().Return(mockSQS)
	mockClient.EXPECT().S3().Return(mockS3)

	// Provide a valid S3 event message
	validS3Event := `{"Records":[{"eventName":"s3:ObjectCreated:Put","s3":{"bucket":{"name":"mybucket"},"object":{"key":"mykey1","size":15}}}]}`

	mockSQS.EXPECT().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput)).Return(&sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{
				Body:          aws.String(validS3Event),
				MessageId:     aws.String("123"),
				ReceiptHandle: aws.String("receipt-handle"),
			},
		},
	}, nil)

	// Mock S3 GetObject to return content after a delay
	mockS3.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
		// Add a delay to simulate processing time and trigger visibility extension
		time.Sleep(10 * time.Millisecond)
		return &s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader("line1\nline2\nline3")),
		}, nil
	})

	mockSQS.EXPECT().DeleteMessage(mock.Anything, mock.Anything).Return(&sqs.DeleteMessageOutput{}, nil)
	mockSQS.EXPECT().ChangeMessageVisibility(mock.Anything, mock.Anything).Return(&sqs.ChangeMessageVisibilityOutput{}, errors.New("visibility extension error"))

	// Set up zap observer
	core, recorded := observer.New(zap.ErrorLevel)
	logger := zap.New(core)
	set := componenttest.NewNopTelemetrySettings()
	set.Logger = logger

	sink := new(consumertest.LogsSink)
	visibilityExtensionInterval := 1 * time.Millisecond
	visibilityTimeout := 300 * time.Second
	b, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)
	w := worker.New(set, sink, mockClient, 4096, 1000, visibilityExtensionInterval, visibilityTimeout, 6*time.Hour, worker.WithTelemetryBuilder(b))

	msg, err := mockClient.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
	require.NoError(t, err)
	require.Len(t, msg.Messages, 1)

	// Process the message - this will take time due to S3 operations
	done := make(chan struct{})
	go func() {
		w.ProcessMessage(ctx, msg.Messages[0], "myqueue", func() { close(done) })
	}()
	// Wait for processing to complete
	<-done

	// Add a small delay to allow visibility extension goroutine to finish logging
	time.Sleep(50 * time.Millisecond)

	t.Logf("recorded logs:")
	for i, entry := range recorded.All() {
		t.Logf("  [%d] %s: %s", i, entry.Level, entry.Message)
	}

	// Check for error log messages
	errorMessages := []string{
		"failed to extend message visibility",
	}

	for _, expectedMsg := range errorMessages {
		found := false
		for _, entry := range recorded.All() {
			if strings.Contains(entry.Message, expectedMsg) {
				found = true
				break
			}
		}
		assert.True(t, found, "expected error message containing '%s' to be present", expectedMsg)
	}
}

func TestProcessMessageWithFilters(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	maxLogSize := 4096
	maxLogsEmitted := 1000
	visibilityExtensionInterval := 100 * time.Millisecond

	testCases := []struct {
		name             string
		objectSets       []map[string]map[string]string
		expectLines      int
		bucketNameFilter *regexp.Regexp
		objectKeyFilter  *regexp.Regexp
	}{
		{
			name: "single object - bucket filter",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
			expectLines:      1,
			bucketNameFilter: regexp.MustCompile("myb.*et"),
		},
		{
			name: "single object - key filter",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
			expectLines:     1,
			objectKeyFilter: regexp.MustCompile(".*key1"),
		},
		{
			name: "single object - bucket filter, no match",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
			expectLines:      0,
			bucketNameFilter: regexp.MustCompile("^.*[xyz]+$"),
		},
		{
			name: "single object - object key filter, no match",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
			expectLines:     0,
			objectKeyFilter: regexp.MustCompile("^.*[xyz]+$"),
		},
		{
			name: "single object - both filters, no match",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
			expectLines:      0,
			bucketNameFilter: regexp.MustCompile("^.*[xyz]+$"),
			objectKeyFilter:  regexp.MustCompile("^.*[xyz]+$"),
		},
		{
			name:             "parses as JSON and creates 4 log lines from a JSON array in Records",
			objectSets:       logsFromFile(t, "testdata/logs_array_in_records.json"),
			expectLines:      4,
			bucketNameFilter: regexp.MustCompile("bucket"),
		},
		{
			name:             "parses as JSON, bucket filter, no match",
			objectSets:       logsFromFile(t, "testdata/logs_array_in_records.json"),
			expectLines:      0,
			bucketNameFilter: regexp.MustCompile("^.*[xz]+$"),
		},
		{
			name:        "attempts to parse as JSON, but fails and creates 294 log lines from text",
			objectSets:  logsFromFile(t, "testdata/logs_array_in_records_after_limit.json"),
			expectLines: 294,
		},
		{
			name:             "attempts to parse as JSON, but fails and creates 294 log lines from text, bucket filter",
			objectSets:       logsFromFile(t, "testdata/logs_array_in_records_after_limit.json"),
			expectLines:      294,
			bucketNameFilter: regexp.MustCompile("^mybucket$"),
		},
		{
			name:            "attempts to parse as JSON, but fails and creates 294 log lines from text, object key filter, no match",
			objectSets:      logsFromFile(t, "testdata/logs_array_in_records_after_limit.json"),
			expectLines:     0,
			objectKeyFilter: regexp.MustCompile("^.*[xyz]+$"),
		},
		{
			name:            "attempts to parse as JSON, but fails and creates 294 log lines from text, object key filters",
			objectSets:      logsFromFile(t, "testdata/logs_array_in_records_after_limit.json"),
			expectLines:     294,
			objectKeyFilter: regexp.MustCompile("testdata/logs_array_in_records_after_limit.json"),
		},

		{
			name:             "does not attempt to parse as JSON and creates 4 log lines from text, bucket filter",
			objectSets:       logsFromFile(t, "testdata/json_lines.txt"),
			expectLines:      4,
			bucketNameFilter: regexp.MustCompile("^mybucket$"),
		},
		{
			name:             "does not attempt to parse as JSON and creates 4 log lines from text, bucket filter, no match",
			objectSets:       logsFromFile(t, "testdata/json_lines.txt"),
			expectLines:      0,
			bucketNameFilter: regexp.MustCompile("^.*[xyz]+$"),
		},
		{
			name:            "does not attempt to parse as JSON and creates 4 log lines from text, object key filter",
			objectSets:      logsFromFile(t, "testdata/json_lines.txt"),
			expectLines:     4,
			objectKeyFilter: regexp.MustCompile("testdata/json_lines.txt"),
		},
		{
			name:            "does not attempt to parse as JSON and creates 4 log lines from text, object key filter, no match",
			objectSets:      logsFromFile(t, "testdata/json_lines.txt"),
			expectLines:     0,
			objectKeyFilter: regexp.MustCompile("^.*[xyz]+$"),
		},
		{
			name:        "parses as avro ocf and creates 10 log lines from avro ocf",
			objectSets:  logsFromFile(t, "testdata/sample_logs.avro"),
			expectLines: 10,
		},
		{
			name:        "parses as avro ocf and creates 1000 log lines from gzipped avro ocf",
			objectSets:  logsFromFile(t, "testdata/sample_logs.avro.gz"),
			expectLines: 1000,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

			var totalObjects int
			for _, objectSet := range testCase.objectSets {
				for _, bucket := range objectSet {
					totalObjects += len(bucket)
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			sink := new(consumertest.LogsSink)

			set := componenttest.NewNopTelemetrySettings()

			b, err := metadata.NewTelemetryBuilder(set)
			require.NoError(t, err)

			opts := []worker.Option{worker.WithTelemetryBuilder(b)}
			if testCase.bucketNameFilter != nil {
				opts = append(opts, worker.WithBucketNameFilter(testCase.bucketNameFilter))
			}
			if testCase.objectKeyFilter != nil {
				opts = append(opts, worker.WithObjectKeyFilter(testCase.objectKeyFilter))
			}
			w := worker.New(set, sink, fakeAWS, maxLogSize, maxLogsEmitted, visibilityExtensionInterval, 300*time.Second, 6*time.Hour, opts...)

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

			var numRecords int
			for _, logs := range sink.AllLogs() {
				numRecords += logs.LogRecordCount()
			}
			require.Equal(t, testCase.expectLines, numRecords)

			_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.ErrorIs(t, err, fake.ErrEmptyQueue)
		})
	}
}
