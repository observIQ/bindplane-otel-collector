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
