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

package awss3eventreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/internal/aws/backoff"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
)

type logsReceiver struct {
	id        component.ID
	cfg       *Config
	telemetry component.TelemetrySettings
	sqsClient client.SQSClient
	next      consumer.Logs
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

func newLogsReceiver(id component.ID, tel component.TelemetrySettings, cfg *Config, next consumer.Logs) (component.Component, error) {
	region, err := client.ParseRegionFromSQSURL(cfg.SQSQueueURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract region from SQS URL: %w", err)
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	return &logsReceiver{
		id:        id,
		cfg:       cfg,
		telemetry: tel,
		next:      next,
		sqsClient: client.NewClient(awsConfig).SQS(),
		workerPool: sync.Pool{
			New: func() any {
				return worker.New(tel, awsConfig, next, cfg.MaxLogSize, cfg.MaxLogsEmitted, cfg.VisibilityTimeout, cfg.VisibilityExtensionInterval, cfg.MaxVisibilityWindow)
			},
		},
	}, nil
}

func (r *logsReceiver) Start(_ context.Context, _ component.Host) error {
	// Context passed to Start is not long running, so we can use a background context
	ctx := context.Background()
	r.startOnce.Do(func() {
		// Create message channel
		r.msgChan = make(chan workerMessage, r.cfg.Workers*2)

		// Start fixed number of workers
		for i := 0; i < r.cfg.Workers; i++ {
			r.workerWg.Add(1)
			go r.runWorker(ctx)
		}

		pollCtx, pollCancel := context.WithCancel(ctx)
		r.pollCancel = pollCancel
		r.pollDone = make(chan struct{})
		go r.poll(pollCtx, func() {
			close(r.pollDone)
		})
	})

	return nil
}

func (r *logsReceiver) runWorker(ctx context.Context) {
	defer r.workerWg.Done()
	w := r.workerPool.Get().(*worker.Worker)

	r.telemetry.Logger.Debug("worker started")

	for {
		select {
		case <-ctx.Done():
			r.telemetry.Logger.Debug("worker stopping due to context cancellation")
			r.workerPool.Put(w)
			return
		case msg, ok := <-r.msgChan:
			if !ok {
				r.telemetry.Logger.Debug("worker stopping due to closed channel")
				r.workerPool.Put(w)
				return
			}
			r.telemetry.Logger.Debug("worker processing message", zap.String("message_id", *msg.msg.MessageId))
			w.ProcessMessage(ctx, msg.msg, msg.queueURL, func() {
				r.telemetry.Logger.Debug("worker finished processing message", zap.String("message_id", *msg.msg.MessageId))
			})
		}
	}
}

func (r *logsReceiver) Shutdown(context.Context) error {
	r.stopOnce.Do(func() {
		if r.pollCancel != nil {
			r.pollCancel()
		}
		if r.pollDone != nil {
			<-r.pollDone
		}
		if r.msgChan != nil {
			close(r.msgChan)
		}
		r.workerWg.Wait()
	})
	return nil
}

func (r *logsReceiver) poll(ctx context.Context, deferThis func()) {
	defer deferThis()

	ticker := time.NewTicker(r.cfg.StandardPollInterval)
	defer ticker.Stop()

	nextInterval := backoff.New(r.telemetry, r.cfg.StandardPollInterval, r.cfg.MaxPollInterval, r.cfg.PollingBackoffFactor)
	for {
		select {
		case <-ctx.Done():
			r.telemetry.Logger.Info("context cancelled, stopping polling")
			return
		case <-ticker.C:
			numMessages := r.receiveMessages(ctx)
			r.telemetry.Logger.Debug(fmt.Sprintf("received %d messages", numMessages))
			ticker.Reset(nextInterval.Update(numMessages))
		}
	}
}

func (r *logsReceiver) receiveMessages(ctx context.Context) int {
	var numMessages int

	queueURL := r.cfg.SQSQueueURL
	params := &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 10,
		VisibilityTimeout:   int32(r.cfg.VisibilityTimeout.Seconds()),
		WaitTimeSeconds:     10, // Use long polling
	}

	resp, err := r.sqsClient.ReceiveMessage(ctx, params)
	if err != nil {
		r.telemetry.Logger.Error("receive messages", zap.Error(err))
		return numMessages
	}

	// loop until we get no messages
	for len(resp.Messages) > 0 {
		r.telemetry.Logger.Debug("messages received", zap.Int("count", len(resp.Messages)), zap.String("first_message_id", *resp.Messages[0].MessageId))

		for _, msg := range resp.Messages {
			select {
			case r.msgChan <- workerMessage{msg: msg, queueURL: queueURL}:
				r.telemetry.Logger.Debug("queued message", zap.String("message_id", *msg.MessageId))
			case <-ctx.Done():
				return numMessages
			}
		}

		r.telemetry.Logger.Debug(fmt.Sprintf("queued %d messages for processing", len(resp.Messages)))

		resp, err = r.sqsClient.ReceiveMessage(ctx, params)
		if err != nil {
			r.telemetry.Logger.Error("receive messages", zap.Error(err))
			return numMessages
		}
	}

	return numMessages
}
