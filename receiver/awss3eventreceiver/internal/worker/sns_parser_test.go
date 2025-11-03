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

package worker

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSNSToS3Event(t *testing.T) {
	// Create a sample S3 event
	eventTime, _ := time.Parse(time.RFC3339, "2025-03-01T14:23:10.616Z")
	s3Event := events.S3Event{
		Records: []events.S3EventRecord{
			{
				EventVersion: "2.1",
				EventSource:  "aws:s3",
				AWSRegion:    "us-east-1",
				EventTime:    eventTime,
				EventName:    "ObjectCreated:Put",
				S3: events.S3Entity{
					Bucket: events.S3Bucket{
						Name: "test-bucket",
					},
					Object: events.S3Object{
						Key:  "test-key.txt",
						Size: 123,
					},
				},
			},
		},
	}

	s3EventJSON, err := json.Marshal(s3Event)
	require.NoError(t, err)

	tests := []struct {
		name        string
		messageBody string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Standard SNS format with Message field",
			messageBody: func() string {
				// Properly escape the S3 event JSON for inclusion in SNS message
				escapedS3Event, _ := json.Marshal(string(s3EventJSON))
				return fmt.Sprintf(`{
					"Type": "Notification",
					"MessageId": "test-id",
					"TopicArn": "arn:aws:sns:us-east-1:123456789012:test-topic",
					"Subject": "Amazon S3 Notification",
					"Message": %s,
					"Timestamp": "2025-03-01T14:23:11.000Z"
				}`, string(escapedS3Event))
			}(),
			expectError: false,
		},
		{
			name: "Invalid SNS type",
			messageBody: func() string {
				// Properly escape the S3 event JSON for inclusion in SNS message
				escapedS3Event, _ := json.Marshal(string(s3EventJSON))
				return fmt.Sprintf(`{
					"Type": "SubscriptionConfirmation",
					"MessageId": "test-id",
					"Message": %s
				}`, string(escapedS3Event))
			}(),
			expectError: true,
			errorMsg:    "expected SNS notification type 'Notification', got 'SubscriptionConfirmation'",
		},
		{
			name: "Missing message field",
			messageBody: `{
				"Type": "Notification",
				"MessageId": "test-id",
				"TopicArn": "arn:aws:sns:us-east-1:123456789012:test-topic"
			}`,
			expectError: true,
			errorMsg:    "no message content found in SNS notification field 'Message'",
		},
		{
			name: "Invalid JSON in SNS message",
			messageBody: `{
				"Type": "Notification",
				"MessageId": "test-id",
				"Message": "invalid-json"
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, err := ParseSNSToS3Event(tt.messageBody)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Records, 1)
				assert.Equal(t, "aws:s3", result.Records[0].EventSource)
				assert.Equal(t, "ObjectCreated:Put", result.Records[0].EventName)
				assert.Equal(t, "test-bucket", result.Records[0].S3.Bucket.Name)
				assert.Equal(t, "test-key.txt", result.Records[0].S3.Object.Key)
			}
		})
	}
}

func TestParseStandardSNSMessage(t *testing.T) {
	s3EventJSON := `{"Records":[{"eventName":"ObjectCreated:Put","s3":{"bucket":{"name":"test-bucket"},"object":{"key":"test.txt"}}}]}`

	tests := []struct {
		name        string
		messageBody string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid standard SNS with Message field",
			messageBody: func() string {
				escapedS3Event, _ := json.Marshal(s3EventJSON)
				return fmt.Sprintf(`{
					"Type": "Notification",
					"MessageId": "test-id",
					"Message": %s
				}`, string(escapedS3Event))
			}(),
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			messageBody: `invalid-json`,
			expectError: true,
			errorMsg:    "failed to unmarshal SNS notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseStandardSNSMessage(tt.messageBody)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestParseS3EventFromJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name: "Valid S3 event",
			jsonData: `{
				"Records": [
					{
						"eventName": "ObjectCreated:Put",
						"s3": {
							"bucket": {"name": "test-bucket"},
							"object": {"key": "test.txt"}
						}
					}
				]
			}`,
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			jsonData:    `invalid-json`,
			expectError: true,
		},
		{
			name:        "Empty JSON",
			jsonData:    ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseS3EventFromJSON(tt.jsonData)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// parseStandardSNSMessageBench is a benchmark-specific version that uses the jsonLib abstraction
// to allow testing different JSON libraries via build tags.
func parseStandardSNSMessageBench(messageBody string) (*events.S3Event, error) {
	var snsNotification SNSNotification
	if err := jsonLib.unmarshal([]byte(messageBody), &snsNotification); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SNS notification: %w", err)
	}

	// Validate this is a notification
	if snsNotification.Type != "Notification" {
		return nil, fmt.Errorf("expected SNS notification type 'Notification', got '%s'", snsNotification.Type)
	}

	// Extract the message content based on the configured field
	messageContent := snsNotification.Message
	if messageContent == "" {
		return nil, fmt.Errorf("no message content found in SNS notification field '%s'", StandardSNSMessageField)
	}

	// Parse the S3 event from the message content
	var s3Event events.S3Event
	if err := jsonLib.unmarshal([]byte(messageContent), &s3Event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal S3 event: %w", err)
	}
	return &s3Event, nil
}

// BenchmarkParseStandardSNSMessage benchmarks the parseStandardSNSMessage function
// with various SNS message sizes and structures.
// Use build tags to switch between JSON libraries:
//   - Default: encoding/json
//   - go test -tags=json_goccy: github.com/goccy/go-json
//   - go test -tags=json_iterator: github.com/json-iterator/go
func BenchmarkParseStandardSNSMessage(b *testing.B) {
	libName := GetJSONLibName()
	b.Logf("Benchmarking with JSON library: %s", libName)
	// Minimal S3 event JSON (single record with basic fields)
	minimalS3EventJSON := `{"Records":[{"eventVersion":"2.1","eventSource":"aws:s3","awsRegion":"us-east-1","eventTime":"2025-03-01T14:23:10.616Z","eventName":"ObjectCreated:Put","s3":{"bucket":{"name":"test-bucket"},"object":{"key":"test-key.txt","size":123}}}]}`
	escapedMinimalS3Event, _ := json.Marshal(minimalS3EventJSON)
	minimalSNSMessage := fmt.Sprintf(`{
		"Type": "Notification",
		"MessageId": "test-id",
		"TopicArn": "arn:aws:sns:us-east-1:123456789012:test-topic",
		"Message": %s
	}`, string(escapedMinimalS3Event))

	// Typical S3 event JSON with full fields (similar to real-world data)
	typicalS3EventJSON := `{"Records":[{"eventVersion":"2.1","eventSource":"aws:s3","awsRegion":"us-east-1","eventTime":"2025-03-01T14:23:10.616Z","eventName":"ObjectCreated:Put","userIdentity":{"principalId":"AWS:AIDAXXXXXXXXXXXXXXXXV"},"requestParameters":{"sourceIPAddress":"96.8.135.246"},"responseElements":{"x-amz-request-id":"RMANNDEXSQ7BP0GZ","x-amz-id-2":"oelm3nELXCwKs59KcgUlPXTSU30ksr/9uoEXLC9uz7+CwARHPGdO4f/zUR5FwJXgqW9YVmTfE09wsDS9+AZQL6O13jB5CX5RdMBHhQ9zxMo="},"s3":{"s3SchemaVersion":"1.0","configurationId":"OTgyMXU4MjMtN2JiZC00XDMzLWFjN2UtYmJjYWYyNTA0Nzlh","bucket":{"name":"s3eventreceiver-dev","ownerIdentity":{"principalId":"A2Q36XOLY4VUGS"},"arn":"arn:aws:s3:::s3eventreceiver-dev"},"object":{"key":"test2.txt","size":25,"eTag":"a021c14602e4aebf8d8279e885b4bb7c","sequencer":"0067X3184E94X0E1A5"}}}]}`
	escapedTypicalS3Event, _ := json.Marshal(typicalS3EventJSON)
	typicalSNSMessage := fmt.Sprintf(`{
		"Type": "Notification",
		"MessageId": "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
		"TopicArn": "arn:aws:sns:us-west-2:123456789012:MyTopic",
		"Subject": "Amazon S3 Notification",
		"Message": %s,
		"Timestamp": "2025-03-01T14:23:11.796Z",
		"SignatureVersion": "1",
		"Signature": "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khHjLiCwHw7M5KIGvLiYlpctP2XEu30EGjNpK1k8h1d7fk2Dbb6XgqjWa6cIE2/KYEvs6RaQbGDnRnyR9Xh6EYLzuvXb7e7V5WGJKaO7xQnJN1xJlpKr1EIlBKJgp3cEoJBqw7/QQhKYYVV5ZjpKL5CHk5Y2TU8VKmKv0FdmRlnZqVp5CZfDxKOlnKvqEPpMFdgg==",
		"SigningCertURL": "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-01d088a6f77103d0fe307c0069e40ed6.pem",
		"UnsubscribeURL": "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:2bcfbf39-05c3-41de-beaa-fcfcc21c8f55"
	}`, string(escapedTypicalS3Event))

	// Build a large S3 event JSON with 100 records
	largeS3Records := make([]string, 100)
	for i := 0; i < 100; i++ {
		largeS3Records[i] = fmt.Sprintf(`{"eventVersion":"2.1","eventSource":"aws:s3","awsRegion":"us-east-1","eventTime":"2025-03-01T14:23:10.616Z","eventName":"ObjectCreated:Put","s3":{"bucket":{"name":"test-bucket-%d","arn":"arn:aws:s3:::test-bucket-%d"},"object":{"key":"test-key-%d.txt","size":%d,"eTag":"etag-%d"}}}`, i, i, i, 1000+i, i)
	}
	recordsStr := ""
	for i, record := range largeS3Records {
		if i > 0 {
			recordsStr += ","
		}
		recordsStr += record
	}
	largeS3EventJSON := fmt.Sprintf(`{"Records":[%s]}`, recordsStr)
	escapedLargeS3Event, _ := json.Marshal(largeS3EventJSON)
	largeSNSMessage := fmt.Sprintf(`{
		"Type": "Notification",
		"MessageId": "large-test-id",
		"TopicArn": "arn:aws:sns:us-east-1:123456789012:test-topic",
		"Message": %s
	}`, string(escapedLargeS3Event))

	// Try to load real test data if available
	var realWorldSNSMessage string
	if data, err := os.ReadFile("testdata/sns_notification_with_s3.json"); err == nil {
		realWorldSNSMessage = string(data)
	}

	benchmarks := []struct {
		name        string
		messageBody string
	}{
		{
			name:        "Minimal",
			messageBody: minimalSNSMessage,
		},
		{
			name:        "Typical",
			messageBody: typicalSNSMessage,
		},
		{
			name:        "Large-100Records",
			messageBody: largeSNSMessage,
		},
	}

	// Add real-world benchmark if test data is available
	if realWorldSNSMessage != "" {
		benchmarks = append(benchmarks, struct {
			name        string
			messageBody string
		}{
			name:        "RealWorld",
			messageBody: realWorldSNSMessage,
		})
	}

	for _, bm := range benchmarks {
		// Include JSON library name in benchmark subtest name
		testName := fmt.Sprintf("%s/%s", libName, bm.name)
		b.Run(testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := parseStandardSNSMessageBench(bm.messageBody)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
