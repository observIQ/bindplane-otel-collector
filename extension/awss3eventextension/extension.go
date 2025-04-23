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

package awss3eventextension // import "github.com/observiq/bindplane-otel-collector/extension/awss3eventextension"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/worker"
	"github.com/observiq/bindplane-otel-collector/internal/aws/backoff"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
)

type awsS3EventExtension struct {
	cfg       *Config
	telemetry component.TelemetrySettings
	sqsClient client.SQSClient
	startOnce sync.Once
	stopOnce  sync.Once

	pollCancel context.CancelFunc
	pollDone   chan struct{}
	workerPool sync.Pool
	workerWg   sync.WaitGroup

	// Channel for distributing messages to worker goroutines
	msgChan chan workerMessage
}

type workerMessage struct {
	msg      types.Message
	queueURL string
}

func (e *awsS3EventExtension) Start(_ context.Context, _ component.Host) error {
	// Context passed to Start is not long running, so we can use a background context
	ctx := context.Background()
	e.startOnce.Do(func() {
		e.msgChan = make(chan workerMessage, e.cfg.Workers*2)

		for range e.cfg.Workers {
			e.workerWg.Add(1)
			go e.runWorker(ctx)
		}

		pollCtx, pollCancel := context.WithCancel(ctx)
		e.pollCancel = pollCancel
		e.pollDone = make(chan struct{})
		go e.poll(pollCtx, func() {
			close(e.pollDone)
		})
	})

	return nil
}

func (e *awsS3EventExtension) runWorker(ctx context.Context) {
	defer e.workerWg.Done()
	w := e.workerPool.Get().(*worker.Worker)

	e.telemetry.Logger.Debug("worker started")

	for {
		select {
		case <-ctx.Done():
			e.telemetry.Logger.Debug("worker stopping due to context cancellation")
			e.workerPool.Put(w)
			return
		case msg, ok := <-e.msgChan:
			if !ok {
				e.telemetry.Logger.Debug("worker stopping due to closed channel")
				e.workerPool.Put(w)
				return
			}
			e.telemetry.Logger.Debug("worker processing message", zap.String("message_id", *msg.msg.MessageId))
			w.ProcessMessage(ctx, msg.msg, msg.queueURL, func() {
				e.telemetry.Logger.Debug("worker finished processing message", zap.String("message_id", *msg.msg.MessageId))
			})
		}
	}
}

func (e *awsS3EventExtension) Shutdown(context.Context) error {
	e.stopOnce.Do(func() {
		if e.pollCancel != nil {
			e.pollCancel()
		}
		if e.pollDone != nil {
			<-e.pollDone
		}
		if e.msgChan != nil {
			close(e.msgChan)
		}
		e.workerWg.Wait()
	})
	return nil
}

func (e *awsS3EventExtension) poll(ctx context.Context, deferThis func()) {
	defer deferThis()

	ticker := time.NewTicker(e.cfg.StandardPollInterval)
	defer ticker.Stop()

	nextInterval := backoff.New(e.telemetry, e.cfg.StandardPollInterval, e.cfg.MaxPollInterval, e.cfg.PollingBackoffFactor)
	for {
		select {
		case <-ctx.Done():
			e.telemetry.Logger.Info("context cancelled, stopping polling")
			return
		case <-ticker.C:
			numMessages := e.receiveMessages(ctx)
			e.telemetry.Logger.Debug(fmt.Sprintf("received %d messages", numMessages))
			ticker.Reset(nextInterval.Update(numMessages))
		}
	}
}

func (e *awsS3EventExtension) receiveMessages(ctx context.Context) int {
	var numMessages int

	queueURL := e.cfg.SQSQueueURL
	params := &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 10,
		VisibilityTimeout:   int32(e.cfg.VisibilityTimeout.Seconds()),
		WaitTimeSeconds:     10, // Use long polling
	}

	resp, err := e.sqsClient.ReceiveMessage(ctx, params)
	if err != nil {
		e.telemetry.Logger.Error("receive messages", zap.Error(err))
		return numMessages
	}

	// loop until we get no messages
	for len(resp.Messages) > 0 {
		e.telemetry.Logger.Debug("messages received", zap.Int("count", len(resp.Messages)), zap.String("first_message_id", *resp.Messages[0].MessageId))

		numMessages += len(resp.Messages)
		for _, msg := range resp.Messages {
			select {
			case e.msgChan <- workerMessage{msg: msg, queueURL: queueURL}:
				e.telemetry.Logger.Debug("queued message", zap.String("message_id", *msg.MessageId))
			case <-ctx.Done():
				return numMessages
			}
		}

		e.telemetry.Logger.Debug(fmt.Sprintf("queued %d messages for processing", len(resp.Messages)))

		resp, err = e.sqsClient.ReceiveMessage(ctx, params)
		if err != nil {
			e.telemetry.Logger.Error("receive messages", zap.Error(err))
			return numMessages
		}
	}

	return numMessages
}
