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
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type logsReceiver struct {
	id        component.ID
	cfg       *Config
	telemetry component.TelemetrySettings
	sqsClient bpaws.SQSClient
	next      consumer.Logs
	startOnce sync.Once
	stopOnce  sync.Once

	pollCancel context.CancelFunc
	pollDone   chan struct{}
	workerPool sync.Pool
	workerWg   sync.WaitGroup
}

func newLogsReceiver(id component.ID, tel component.TelemetrySettings, cfg *Config, next consumer.Logs) (component.Component, error) {
	region, err := bpaws.ParseRegionFromSQSURL(cfg.SQSQueueURL)
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
		sqsClient: bpaws.NewClient(awsConfig).SQS(),
		workerPool: sync.Pool{
			New: func() interface{} {
				return worker.New(tel, awsConfig, next)
			},
		},
	}, nil
}

func (r *logsReceiver) Start(ctx context.Context, _ component.Host) error {
	r.startOnce.Do(func() {
		pollCtx, pollCancel := context.WithCancel(ctx)
		r.pollCancel = pollCancel
		r.pollDone = make(chan struct{})
		go r.poll(pollCtx, func() {
			close(r.pollDone)
		})
	})

	return nil
}

func (r *logsReceiver) Shutdown(context.Context) error {
	r.stopOnce.Do(func() {
		r.pollCancel()
		<-r.pollDone
		r.workerWg.Wait()
	})
	return nil
}

func (r *logsReceiver) poll(ctx context.Context, deferThis func()) {
	defer deferThis()
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.telemetry.Logger.Info("context cancelled, stopping polling")
			return
		case <-ticker.C:
			err := r.receiveMessages(ctx)
			if err != nil {
				r.telemetry.Logger.Error("polling SQS", zap.Error(err))
			}
		}
	}
}

func (r *logsReceiver) receiveMessages(ctx context.Context) error {
	queueURL := r.cfg.SQSQueueURL
	receiveParams := &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: r.cfg.APIMaxMessages,
		VisibilityTimeout:   int32(r.cfg.VisibilityTimeout.Seconds()),
		WaitTimeSeconds:     10, // Use long polling
	}

	resp, err := r.sqsClient.ReceiveMessage(ctx, receiveParams)
	if err != nil {
		return fmt.Errorf("receive messages: %w", err)
	}

	if len(resp.Messages) == 0 {
		return nil
	}

	r.telemetry.Logger.Debug("messages received", zap.Int("count", len(resp.Messages)))

	for _, msg := range resp.Messages {
		w := r.workerPool.Get().(*worker.Worker)
		r.workerWg.Add(1)
		go w.ProcessMessage(ctx, msg, queueURL, func() {
			r.workerWg.Done()
			r.workerPool.Put(w)
		})
	}

	r.telemetry.Logger.Debug("processed all messages")
	return nil
}
