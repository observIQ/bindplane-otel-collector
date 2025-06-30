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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/stretchr/testify/require"
)

// ErrEmptyQueue is an error returned when a queue is empty
var ErrEmptyQueue = errors.New("queue is empty")

var _ client.SQSClient = &sqsClient{}

var fakeSQS = struct {
	mu           sync.Mutex
	messageCount int

	messages          []types.Message
	invisibleMessages map[string]invisibleMessage
	deletedMessages   []string
}{
	messages:          []types.Message{},
	invisibleMessages: make(map[string]invisibleMessage),
	deletedMessages:   []string{},
}

// invisibleMessage tracks a message that is currently invisible with its visibility timeout
type invisibleMessage struct {
	message           types.Message
	visibilityTimeout time.Time
}

// NewSQSClient creates a new fake SQS client
// If t is provided, automatically registers message leak checking for test cleanup
func NewSQSClient(t *testing.T) client.SQSClient {
	// Register leak check if testing.T was provided

	t.Cleanup(func() {
		fakeSQS.mu.Lock()
		defer fakeSQS.mu.Unlock()

		require.Empty(t, fakeSQS.messages)
		require.Empty(t, fakeSQS.invisibleMessages)
	})

	return &sqsClient{}
}

type sqsClient struct{}

func (f *sqsClient) ReceiveMessage(_ context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	fakeSQS.mu.Lock()
	defer fakeSQS.mu.Unlock()

	// Check for expired visibility timeouts and move messages back to visible queue
	f.checkExpiredVisibilityTimeouts()

	if len(fakeSQS.messages) == 0 {
		return nil, ErrEmptyQueue
	}

	numMessages := len(fakeSQS.messages)
	if params.MaxNumberOfMessages > 0 && int(params.MaxNumberOfMessages) < numMessages {
		numMessages = int(params.MaxNumberOfMessages)
	}

	messages := fakeSQS.messages[:numMessages]
	fakeSQS.messages = fakeSQS.messages[numMessages:]

	// Calculate visibility timeout based on the parameter or use a default
	visibilityTimeout := time.Duration(30) * time.Second // Default 30 seconds
	if params.VisibilityTimeout > 0 {
		visibilityTimeout = time.Duration(params.VisibilityTimeout) * time.Second
	}

	// Move messages to invisible state with timeout
	for _, msg := range messages {
		fakeSQS.invisibleMessages[*msg.ReceiptHandle] = invisibleMessage{
			message:           msg,
			visibilityTimeout: time.Now().Add(visibilityTimeout),
		}
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

	if _, exists := fakeSQS.invisibleMessages[*params.ReceiptHandle]; !exists {
		return nil, fmt.Errorf("attempt to delete message that wasn't received: %s", *params.ReceiptHandle)
	}

	delete(fakeSQS.invisibleMessages, *params.ReceiptHandle)
	fakeSQS.deletedMessages = append(fakeSQS.deletedMessages, *params.ReceiptHandle)

	return &sqs.DeleteMessageOutput{}, nil
}

func (f *sqsClient) ChangeMessageVisibility(_ context.Context, params *sqs.ChangeMessageVisibilityInput, _ ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error) {
	fakeSQS.mu.Lock()
	defer fakeSQS.mu.Unlock()

	invisibleMsg, exists := fakeSQS.invisibleMessages[*params.ReceiptHandle]
	if !exists {
		return nil, fmt.Errorf("attempt to change visibility of message that wasn't received: %s", *params.ReceiptHandle)
	}

	// Calculate new visibility timeout
	visibilityTimeout := time.Duration(params.VisibilityTimeout) * time.Second
	newTimeout := time.Now().Add(visibilityTimeout)

	// Update the visibility timeout for this message
	invisibleMsg.visibilityTimeout = newTimeout
	fakeSQS.invisibleMessages[*params.ReceiptHandle] = invisibleMsg

	return &sqs.ChangeMessageVisibilityOutput{}, nil
}

// checkExpiredVisibilityTimeouts moves messages with expired visibility timeouts back to the visible queue
func (f *sqsClient) checkExpiredVisibilityTimeouts() {
	now := time.Now()
	for receiptHandle, invisibleMsg := range fakeSQS.invisibleMessages {
		if now.After(invisibleMsg.visibilityTimeout) {
			// Move message back to visible queue
			fakeSQS.messages = append(fakeSQS.messages, invisibleMsg.message)
			delete(fakeSQS.invisibleMessages, receiptHandle)
		}
	}
}

func (f *sqsClient) sendMessage(body []byte) {
	fakeSQS.mu.Lock()
	defer fakeSQS.mu.Unlock()
	fakeSQS.messages = append(fakeSQS.messages, types.Message{
		MessageId:     aws.String(fmt.Sprintf("messageId-%d", fakeSQS.messageCount)),
		Body:          aws.String(string(body)),
		ReceiptHandle: aws.String(fmt.Sprintf("receiptHandle-%d", fakeSQS.messageCount)),
	})
	fakeSQS.messageCount++
}
