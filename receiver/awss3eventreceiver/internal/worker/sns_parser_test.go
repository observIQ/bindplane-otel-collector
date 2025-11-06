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
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
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

// Benchmark helper functions to generate representative datasets

// smallSNSMessage creates an SNS notification with a single S3 record (minimal fields)
func smallSNSMessage() string {
	eventTime := time.Date(2025, 3, 1, 14, 23, 10, 616000000, time.UTC)
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

	s3EventJSON, _ := json.Marshal(s3Event)
	escapedS3Event, _ := json.Marshal(string(s3EventJSON))

	return fmt.Sprintf(`{
		"Type": "Notification",
		"MessageId": "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
		"TopicArn": "arn:aws:sns:us-east-1:123456789012:MyTopic",
		"Subject": "Amazon S3 Notification",
		"Message": %s,
		"Timestamp": "2025-03-01T14:23:11.796Z"
	}`, string(escapedS3Event))
}

// mediumSNSMessage creates an SNS notification with 10 S3 records (typical fields)
func mediumSNSMessage() string {
	eventTime := time.Date(2025, 3, 1, 14, 23, 10, 616000000, time.UTC)
	records := make([]events.S3EventRecord, 10)

	for i := 0; i < 10; i++ {
		records[i] = events.S3EventRecord{
			EventVersion: "2.1",
			EventSource:  "aws:s3",
			AWSRegion:    "us-east-1",
			EventTime:    eventTime.Add(time.Duration(i) * time.Second),
			EventName:    "ObjectCreated:Put",
			PrincipalID: events.S3UserIdentity{
				PrincipalID: fmt.Sprintf("AWS:AIDAXXXXXXXXXXXXXXXX%d", i),
			},
			RequestParameters: events.S3RequestParameters{
				SourceIPAddress: fmt.Sprintf("192.168.1.%d", i),
			},
			ResponseElements: map[string]string{
				"x-amz-request-id": fmt.Sprintf("RMANNDEXSQ7BP0GZ%d", i),
				"x-amz-id-2":       fmt.Sprintf("oelm3nELXCwKs59KcgUlPXTSU30ksr/9uoEXLC9uz7+CwARHPGdO4f/zUR5FwJXgqW9YVmTfE09wsDS9+AZQL6O13jB5CX5RdMBHhQ9zxMo=%d", i),
			},
			S3: events.S3Entity{
				SchemaVersion:   "1.0",
				ConfigurationID: fmt.Sprintf("OTgyMXU4MjMtN2JiZC00XDMzLWFjN2UtYmJjYWYyNTA0Nzlh%d", i),
				Bucket: events.S3Bucket{
					Name: fmt.Sprintf("s3eventreceiver-dev-bucket-%d", i),
					OwnerIdentity: events.S3UserIdentity{
						PrincipalID: "A2Q36XOLY4VUGS",
					},
					Arn: fmt.Sprintf("arn:aws:s3:::s3eventreceiver-dev-bucket-%d", i),
				},
				Object: events.S3Object{
					Key:       fmt.Sprintf("year%%3D2025/month%%3D05/day%%3D01/hour%%3D10/minute%%3D32/logs_%d.json", i),
					Size:      int64(1000 + i*100),
					ETag:      fmt.Sprintf("a021c14602e4aebf8d8279e885b4bb7c%d", i),
					Sequencer: fmt.Sprintf("0067X3184E94X0E1A5%d", i),
				},
			},
		}
	}

	s3Event := events.S3Event{Records: records}
	s3EventJSON, _ := json.Marshal(s3Event)
	escapedS3Event, _ := json.Marshal(string(s3EventJSON))

	return fmt.Sprintf(`{
		"Type": "Notification",
		"MessageId": "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
		"TopicArn": "arn:aws:sns:us-east-1:123456789012:MyTopic",
		"Subject": "Amazon S3 Notification",
		"Message": %s,
		"Timestamp": "2025-03-01T14:23:11.796Z",
		"SignatureVersion": "1",
		"Signature": "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khHjLiCwHw7M5KIGvLiYlpctP2XEu30EGjNpK1k8h1d7fk2Dbb6XgqjWa6cIE2/KYEvs6RaQbGDnRnyR9Xh6EYLzuvXb7e7V5WGJKaO7xQnJN1xJlpKr1EIlBKJgp3cEoJBqw7/QQhKYYVV5ZjpKL5CHk5Y2TU8VKmKv0FdmRlnZqVp5CZfDxKOlnKvqEPpMFdgg==",
		"SigningCertURL": "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-01d088a6f77103d0fe307c0069e40ed6.pem",
		"UnsubscribeURL": "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:123456789012:MyTopic:2bcfbf39-05c3-41de-beaa-fcfcc21c8f55"
	}`, string(escapedS3Event))
}

// largeSNSMessage creates an SNS notification with 100 S3 records (all fields populated)
func largeSNSMessage() string {
	eventTime := time.Date(2025, 3, 1, 14, 23, 10, 616000000, time.UTC)
	records := make([]events.S3EventRecord, 100)

	for i := 0; i < 100; i++ {
		// Create longer keys to simulate real-world scenarios with nested paths
		keyPath := strings.Repeat("nested/", i%10) + fmt.Sprintf("year%%3D2025/month%%3D05/day%%3D01/hour%%3D10/minute%%3D32/logs_%d.json", i)

		records[i] = events.S3EventRecord{
			EventVersion: "2.1",
			EventSource:  "aws:s3",
			AWSRegion:    "us-east-1",
			EventTime:    eventTime.Add(time.Duration(i) * time.Second),
			EventName:    "ObjectCreated:Put",
			PrincipalID: events.S3UserIdentity{
				PrincipalID: fmt.Sprintf("AWS:AIDAXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX%d", i),
			},
			RequestParameters: events.S3RequestParameters{
				SourceIPAddress: fmt.Sprintf("192.168.1.%d", i%256),
			},
			ResponseElements: map[string]string{
				"x-amz-request-id": fmt.Sprintf("RMANNDEXSQ7BP0GZ%d", i),
				"x-amz-id-2":       fmt.Sprintf("oelm3nELXCwKs59KcgUlPXTSU30ksr/9uoEXLC9uz7+CwARHPGdO4f/zUR5FwJXgqW9YVmTfE09wsDS9+AZQL6O13jB5CX5RdMBHhQ9zxMo=%d", i),
			},
			S3: events.S3Entity{
				SchemaVersion:   "1.0",
				ConfigurationID: fmt.Sprintf("OTgyMXU4MjMtN2JiZC00XDMzLWFjN2UtYmJjYWYyNTA0Nzlh%d", i),
				Bucket: events.S3Bucket{
					Name: fmt.Sprintf("s3eventreceiver-dev-bucket-%d", i/10),
					OwnerIdentity: events.S3UserIdentity{
						PrincipalID: "A2Q36XOLY4VUGS",
					},
					Arn: fmt.Sprintf("arn:aws:s3:::s3eventreceiver-dev-bucket-%d", i/10),
				},
				Object: events.S3Object{
					Key:       keyPath,
					Size:      int64(10000 + i*1000),
					ETag:      fmt.Sprintf("a021c14602e4aebf8d8279e885b4bb7c%d", i),
					Sequencer: fmt.Sprintf("0067X3184E94X0E1A5%d", i),
				},
			},
		}
	}

	s3Event := events.S3Event{Records: records}
	s3EventJSON, _ := json.Marshal(s3Event)
	escapedS3Event, _ := json.Marshal(string(s3EventJSON))

	return fmt.Sprintf(`{
		"Type": "Notification",
		"MessageId": "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
		"TopicArn": "arn:aws:sns:us-east-1:123456789012:MyTopic",
		"Subject": "Amazon S3 Notification",
		"Message": %s,
		"Timestamp": "2025-03-01T14:23:11.796Z",
		"SignatureVersion": "1",
		"Signature": "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khHjLiCwHw7M5KIGvLiYlpctP2XEu30EGjNpK1k8h1d7fk2Dbb6XgqjWa6cIE2/KYEvs6RaQbGDnRnyR9Xh6EYLzuvXb7e7V5WGJKaO7xQnJN1xJlpKr1EIlBKJgp3cEoJBqw7/QQhKYYVV5ZjpKL5CHk5Y2TU8VKmKv0FdmRlnZqVp5CZfDxKOlnKvqEPpMFdgg==",
		"SigningCertURL": "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-01d088a6f77103d0fe307c0069e40ed6.pem",
		"UnsubscribeURL": "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:123456789012:MyTopic:2bcfbf39-05c3-41de-beaa-fcfcc21c8f55"
	}`, string(escapedS3Event))
}

// BenchmarkParseSNSToS3Event benchmarks the full ParseSNSToS3Event function
// which performs two JSON unmarshals: SNS notification and S3 event
func BenchmarkParseSNSToS3Event(b *testing.B) {
	sizes := []struct {
		name    string
		message func() string
	}{
		{"small", smallSNSMessage},
		{"medium", mediumSNSMessage},
		{"large", largeSNSMessage},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			messageBody := size.message()
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := ParseSNSToS3Event(messageBody)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
