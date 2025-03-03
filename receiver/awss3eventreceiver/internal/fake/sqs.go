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
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
)

var ErrEmptyQueue = errors.New("queue is empty")

var _ bpaws.SQSClient = &sqsClient{}

var fakeSQS = struct {
	mu sync.Mutex

	messages        []types.Message
	deletedMessages []string
}{
	messages:        []types.Message{},
	deletedMessages: []string{},
}

// NewSQSClient creates a new fake SQS client with the provided messages
func NewSQSClient(_ aws.Config) bpaws.SQSClient {
	return &sqsClient{}
}

type sqsClient struct{}

func (f *sqsClient) ReceiveMessage(_ context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	fakeSQS.mu.Lock()
	defer fakeSQS.mu.Unlock()

	if len(fakeSQS.messages) == 0 {
		return nil, ErrEmptyQueue
	}

	messages := fakeSQS.messages
	if params.MaxNumberOfMessages > 0 && int(params.MaxNumberOfMessages) < len(messages) {
		messages = messages[:int(params.MaxNumberOfMessages)]
	}

	copyMessages := make([]types.Message, len(messages))
	copy(copyMessages, messages)
	return &sqs.ReceiveMessageOutput{
		Messages: copyMessages,
	}, nil
}

func (f *sqsClient) DeleteMessage(_ context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	fakeSQS.mu.Lock()
	defer fakeSQS.mu.Unlock()

	for i, msg := range fakeSQS.messages {
		if *msg.ReceiptHandle == *params.ReceiptHandle {
			fakeSQS.messages = append(fakeSQS.messages[:i], fakeSQS.messages[i+1:]...)
			fakeSQS.deletedMessages = append(fakeSQS.deletedMessages, *params.ReceiptHandle)
			break
		}
	}

	return &sqs.DeleteMessageOutput{}, nil
}

func (f *sqsClient) sendMessage(msg types.Message) {
	fakeSQS.mu.Lock()
	defer fakeSQS.mu.Unlock()
	fakeSQS.messages = append(fakeSQS.messages, msg)
}
