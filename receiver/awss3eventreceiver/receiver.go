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
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/internal/aws/backoff"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
)

type logsReceiver struct {
	id        component.ID
	cfg       *Config
	telemetry component.TelemetrySettings
	metrics   *metadata.TelemetryBuilder
	sqsClient client.SQSClient
	next      consumer.Logs
	startOnce sync.Once
	stopOnce  sync.Once

	pollCancel context.CancelFunc
	pollDone   chan struct{}
	workerPool sync.Pool
	workerWg   sync.WaitGroup

	offsetStorage storageclient.StorageClient

	// Channel for distributing messages to worker goroutines
	msgChan chan workerMessage
}

type workerMessage struct {
	msg      types.Message
	queueURL string
}

func newLogsReceiver(id component.ID, tel component.TelemetrySettings, cfg *Config, next consumer.Logs, tb *metadata.TelemetryBuilder) (component.Component, error) {
	region, err := client.ParseRegionFromSQSURL(cfg.SQSQueueURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract region from SQS URL: %w", err)
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	var bucketNameFilter *regexp.Regexp
	if cfg.BucketNameFilter != "" {
		bucketNameFilter, err = regexp.Compile(cfg.BucketNameFilter)
		if err != nil {
			// config validation should have already caught this
			return nil, fmt.Errorf("failed to compile bucket_name_filter regex: %w", err)
		}
	}

	var objectKeyFilter *regexp.Regexp
	if cfg.ObjectKeyFilter != "" {
		objectKeyFilter, err = regexp.Compile(cfg.ObjectKeyFilter)
		if err != nil {
			// config validation should have already caught this
			return nil, fmt.Errorf("failed to compile regex filter: %w", err)
		}
	}

	return &logsReceiver{
		id:        id,
		cfg:       cfg,
		telemetry: tel,
		metrics:   tb,
		next:      next,
		sqsClient: client.NewClient(awsConfig).SQS(),
		workerPool: sync.Pool{
			New: func() any {
				opts := []worker.Option{
					worker.WithTelemetryBuilder(tb),
				}
				if bucketNameFilter != nil {
					opts = append(opts, worker.WithBucketNameFilter(bucketNameFilter))
				}
				if objectKeyFilter != nil {
					opts = append(opts, worker.WithObjectKeyFilter(objectKeyFilter))
				}
				return worker.New(tel, next, client.NewClient(awsConfig), cfg.MaxLogSize, cfg.MaxLogsEmitted, cfg.VisibilityTimeout, cfg.VisibilityExtensionInterval, cfg.MaxVisibilityWindow, opts...)
			},
		},
		offsetStorage: storageclient.NewNopStorage(),
	}, nil
}

func (r *logsReceiver) Start(_ context.Context, host component.Host) error {
	// Context passed to Start is not long running, so we can use a background context
	ctx := context.Background()

	// Create offset storage
	if r.cfg.StorageID != nil {
		offsetStorage, err := storageclient.NewStorageClient(ctx, host, *r.cfg.StorageID, r.id, pipeline.SignalLogs)
		if err != nil {
			return fmt.Errorf("failed to create offset storage: %w", err)
		}
		r.offsetStorage = offsetStorage
	}

	// Start workers on separate goroutines
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

	w.SetOffsetStorage(r.offsetStorage)

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

func (r *logsReceiver) Shutdown(ctx context.Context) error {
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

	// close offset storage once workers are stopped
	if err := r.offsetStorage.Close(ctx); err != nil {
		return fmt.Errorf("failed to close offset storage: %w", err)
	}

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

	// loop until we get no messages
	for {
		resp, err := r.sqsClient.ReceiveMessage(ctx, params)
		if err != nil {
			r.telemetry.Logger.Error("receive messages", zap.Error(err))
			r.metrics.S3eventFailures.Add(ctx, 1)
			return numMessages
		}
		if len(resp.Messages) == 0 {
			return numMessages
		}
		r.telemetry.Logger.Debug("messages received", zap.Int("count", len(resp.Messages)), zap.String("first_message_id", *resp.Messages[0].MessageId))

		for _, msg := range resp.Messages {
			select {
			case r.msgChan <- workerMessage{msg: msg, queueURL: queueURL}:
				r.telemetry.Logger.Debug("queued message", zap.String("message_id", *msg.MessageId))
				numMessages++
			case <-ctx.Done():
				return numMessages
			}
		}

		r.telemetry.Logger.Debug(fmt.Sprintf("queued %d messages for processing", len(resp.Messages)))
	}
}
