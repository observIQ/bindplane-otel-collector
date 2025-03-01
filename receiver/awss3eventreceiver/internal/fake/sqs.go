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

// Package fake provides fake implementations of AWS clients for testing
package fake

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpsqs"
)

// NewSQSClient creates a new fake SQS client with the provided messages
func NewSQSClient(messages []types.Message) func(cfg aws.Config) bpsqs.Client {
	return func(_ aws.Config) bpsqs.Client {
		return &sqsClient{
			Messages:        messages,
			DeletedMessages: []string{},
		}
	}
}

type sqsClient struct {
	Messages        []types.Message
	ReceiveError    error
	DeleteError     error
	DeletedMessages []string
	mu              sync.Mutex
}

func (f *sqsClient) ReceiveMessage(_ context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ReceiveError != nil {
		return nil, f.ReceiveError
	}

	maxMessages := len(f.Messages)
	if params.MaxNumberOfMessages > 0 && int(params.MaxNumberOfMessages) < maxMessages {
		maxMessages = int(params.MaxNumberOfMessages)
	}

	return &sqs.ReceiveMessageOutput{
		Messages: f.Messages[:maxMessages],
	}, nil
}

func (f *sqsClient) DeleteMessage(_ context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.DeleteError != nil {
		return nil, f.DeleteError
	}

	if params.ReceiptHandle != nil {
		f.DeletedMessages = append(f.DeletedMessages, *params.ReceiptHandle)
	}

	return &sqs.DeleteMessageOutput{}, nil
}
