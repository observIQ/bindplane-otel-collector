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

package worker // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

// SNSNotification represents the structure of an SNS notification
type SNSNotification struct {
	Type      string `json:"Type"`
	MessageID string `json:"MessageId"`
	TopicArn  string `json:"TopicArn"`
	Subject   string `json:"Subject"`
	Message   string `json:"Message"`
	Timestamp string `json:"Timestamp"`
	// Additional fields like Signature, SigningCertURL, UnsubscribeURL can be added if needed
}

const (
	// StandardSNSMessageField is the name of the field that contains the S3 event in a standard SNS notification
	StandardSNSMessageField = "Message"
)

// ParseSNSToS3Event parses an SNS notification containing an S3 event
func ParseSNSToS3Event(messageBody string) (*events.S3Event, error) {
	// Standard SNS format - extract the Message field
	return parseStandardSNSMessage(messageBody)
}

// parseStandardSNSMessage parses a standard SNS notification and extracts the S3 event
func parseStandardSNSMessage(messageBody string) (*events.S3Event, error) {
	var snsNotification SNSNotification
	if err := json.Unmarshal([]byte(messageBody), &snsNotification); err != nil {
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
	return parseS3EventFromJSON(messageContent)
}

// parseS3EventFromJSON parses an S3 event from JSON string
func parseS3EventFromJSON(jsonData string) (*events.S3Event, error) {
	var s3Event events.S3Event
	if err := json.Unmarshal([]byte(jsonData), &s3Event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal S3 event: %w", err)
	}
	return &s3Event, nil
}
