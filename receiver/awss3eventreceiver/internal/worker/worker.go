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

// Package worker provides a worker that processes S3 event notifications.
package worker // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
)

// Worker processes S3 event notifications.
// It is responsible for processing messages from the SQS queue and sending them to the next consumer.
// It also handles deleting messages from the SQS queue after they have been processed.
// It is designed to be used in a worker pool.
type Worker struct {
	logger                      *zap.Logger
	tel                         component.TelemetrySettings
	client                      client.Client
	nextConsumer                consumer.Logs
	maxLogSize                  int
	maxLogsEmitted              int
	visibilityTimeout           time.Duration
	visibilityExtensionInterval time.Duration
	maxVisibilityWindow         time.Duration
}

// New creates a new Worker
func New(tel component.TelemetrySettings, nextConsumer consumer.Logs, client client.Client, maxLogSize int, maxLogsEmitted int, visibilityTimeout time.Duration, visibilityExtensionInterval time.Duration, maxVisibilityWindow time.Duration) *Worker {
	return &Worker{
		logger:                      tel.Logger.With(zap.String("component", "awss3eventreceiver")),
		tel:                         tel,
		client:                      client,
		nextConsumer:                nextConsumer,
		maxLogSize:                  maxLogSize,
		maxLogsEmitted:              maxLogsEmitted,
		visibilityTimeout:           visibilityTimeout,
		visibilityExtensionInterval: visibilityExtensionInterval,
		maxVisibilityWindow:         maxVisibilityWindow,
	}
}

// ProcessMessage processes a message from the SQS queue
// TODO add metric for number of messages processed / deleted / errors, events processed, etc.
func (w *Worker) ProcessMessage(ctx context.Context, msg types.Message, queueURL string, deferThis func()) {
	defer deferThis()

	logger := w.logger.With(zap.String("message_id", *msg.MessageId), zap.String("queue_url", queueURL))

	// Start a goroutine to periodically extend message visibility
	visibilityCtx, cancelVisibility := context.WithCancel(ctx)
	defer cancelVisibility()

	go w.extendMessageVisibility(visibilityCtx, msg, queueURL, logger)

	notification := new(events.S3Event)
	err := json.Unmarshal([]byte(*msg.Body), notification)
	if err != nil {
		w.tel.Logger.Error("unmarshal notification", zap.Error(err))
		// We can delete messages with unmarshaling errors as they'll never succeed
		w.deleteMessage(ctx, msg, queueURL, logger)
		return
	}
	logger.Debug("processing notification", zap.Int("event.count", len(notification.Records)))

	// Filter records to only include s3:ObjectCreated:* events
	var objectCreatedRecords []events.S3EventRecord
	for _, record := range notification.Records {
		recordLogger := logger.With(zap.String("event_name", record.EventName),
			zap.String("bucket", record.S3.Bucket.Name),
			zap.String("key", record.S3.Object.Key))
		// S3 UI shows the prefix as "s3:ObjectCreated:", but the event name is unmarshalled as "ObjectCreated:"
		if strings.Contains(record.EventName, "ObjectCreated:") {
			objectCreatedRecords = append(objectCreatedRecords, record)
		} else {
			recordLogger.Warn("unexpected event: receiver handles only s3:ObjectCreated:* events")
		}
	}

	if len(objectCreatedRecords) == 0 {
		logger.Debug("no s3:ObjectCreated:* events found in notification, skipping")
		w.deleteMessage(ctx, msg, queueURL, logger)
		return
	}

	if len(objectCreatedRecords) > 1 {
		logger.Warn("duplicate logs possible: multiple s3:ObjectCreated:* events found in notification", zap.Int("event.count", len(objectCreatedRecords)))
	}

	for _, record := range objectCreatedRecords {
		recordLogger := logger.With(zap.String("bucket", record.S3.Bucket.Name), zap.String("key", record.S3.Object.Key))
		recordLogger.Debug("processing record")

		if err := w.processRecord(ctx, record, recordLogger); err != nil {
			recordLogger.Error("error processing record, preserving message in SQS for retry", zap.Error(err))
			return
		}
	}
	w.deleteMessage(ctx, msg, queueURL, logger)
}

func (w *Worker) processRecord(ctx context.Context, record events.S3EventRecord, recordLogger *zap.Logger) error {
	err := w.consumeLogsFromS3Object(ctx, record, true, recordLogger)
	if err != nil {
		if errors.Is(err, ErrNotArrayOrKnownObject) {
			// try again without attempting to parse as JSON
			return w.consumeLogsFromS3Object(ctx, record, false, recordLogger)
		}
		return err
	}
	return nil
}

func (w *Worker) consumeLogsFromS3Object(ctx context.Context, record events.S3EventRecord, tryJSON bool, recordLogger *zap.Logger) error {
	bucket := record.S3.Bucket.Name
	key := record.S3.Object.Key
	size := record.S3.Object.Size
	opts := []func(o *s3.Options){
		func(o *s3.Options) {
			o.Region = record.AWSRegion
		},
	}

	recordLogger.Debug("reading S3 object", zap.Int64("size", size))

	resp, err := w.client.S3().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, opts...)
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer resp.Body.Close()

	now := time.Now()

	stream := logStream{
		name:            key,
		contentEncoding: resp.ContentEncoding,
		contentType:     resp.ContentType,
		body:            resp.Body,
		maxLogSize:      w.maxLogSize,
		logger:          recordLogger,
		tryJSON:         tryJSON,
	}

	reader, err := stream.BufferedReader(ctx)
	if err != nil {
		return fmt.Errorf("get stream reader: %w", err)
	}

	parser, err := newParser(ctx, stream, reader)
	if err != nil {
		return fmt.Errorf("create parser: %w", err)
	}

	ld := plog.NewLogs()
	rls := ld.ResourceLogs().AppendEmpty()
	rls.Resource().Attributes().PutStr("aws.s3.bucket", bucket)
	rls.Resource().Attributes().PutStr("aws.s3.key", key)
	lrs := rls.ScopeLogs().AppendEmpty().LogRecords()

	batchesConsumedCount := 0

	// Parse logs into a sequence of log records
	logs, err := parser.Parse(ctx)
	if err != nil {
		return fmt.Errorf("parse logs: %w", err)
	}

	for log, err := range logs {
		if err != nil {
			recordLogger.Error("parse log", zap.Error(err))
			continue
		}

		// Create a log record for this line fragment
		lr := lrs.AppendEmpty()
		lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(now))
		lr.SetTimestamp(pcommon.NewTimestampFromTime(record.EventTime))

		err := parser.AppendLogBody(ctx, lr, log)
		if err != nil {
			recordLogger.Error("append log body", zap.Error(err))
			continue
		}

		if ld.LogRecordCount() >= w.maxLogsEmitted {
			if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
				recordLogger.Error("consume logs", zap.Error(err), zap.Int("batches_consumed_count", batchesConsumedCount))
				return fmt.Errorf("consume logs: %w", err)
			}
			batchesConsumedCount++
			recordLogger.Debug("Reached max logs for single batch, starting new batch", zap.Int("batches_consumed_count", batchesConsumedCount))

			ld = plog.NewLogs()
			rls = ld.ResourceLogs().AppendEmpty()
			rls.Resource().Attributes().PutStr("aws.s3.bucket", bucket)
			rls.Resource().Attributes().PutStr("aws.s3.key", key)
			lrs = rls.ScopeLogs().AppendEmpty().LogRecords()
		}
	}

	if ld.LogRecordCount() == 0 {
		return nil
	}

	if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
		recordLogger.Error("consume logs", zap.Error(err), zap.Int("batches_consumed_count", batchesConsumedCount))
		return fmt.Errorf("consume logs: %w", err)
	}
	recordLogger.Debug("processed S3 object", zap.Int("batches_consumed_count", batchesConsumedCount+1))
	return nil
}

func (w *Worker) deleteMessage(ctx context.Context, msg types.Message, queueURL string, recordLogger *zap.Logger) {
	deleteParams := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	}
	_, err := w.client.SQS().DeleteMessage(ctx, deleteParams)
	if err != nil {
		recordLogger.Error("delete message", zap.Error(err))
		return
	}
	recordLogger.Debug("deleted message")
}

func (w *Worker) extendMessageVisibility(ctx context.Context, msg types.Message, queueURL string, logger *zap.Logger) {
	monitor := newVisibilityMonitor(logger, msg, w.visibilityTimeout, w.visibilityExtensionInterval, w.maxVisibilityWindow)
	defer monitor.stop()

	logger.Debug("starting visibility extension monitoring",
		zap.Duration("initial_timeout", monitor.visibilityTimeout),
		zap.Duration("extension_interval", monitor.extensionInterval),
		zap.Duration("max_window", monitor.maxVisibilityEndTime.Sub(monitor.startTime)),
		zap.Time("max_end_time", monitor.maxVisibilityEndTime))

	for {
		select {
		case <-ctx.Done():
			logger.Debug("visibility extension stopped due to context cancellation")
			return
		case <-monitor.nextExtensionTimer():
			if monitor.shouldExtendToMax() {
				w.extendToMaxAndStop(ctx, msg, queueURL, monitor, logger)
				return
			}
			if err := w.extendVisibility(ctx, msg, queueURL, w.visibilityExtensionInterval, logger); err != nil {
				logger.Error("failed to extend message visibility", zap.Error(err), zap.Duration("attempted_timeout", w.visibilityExtensionInterval))
				return
			}
			monitor.scheduleNextExtension(logger)
		}
	}
}

type visibilityMonitor struct {
	logger               *zap.Logger
	msg                  types.Message
	startTime            time.Time
	maxVisibilityEndTime time.Time
	visibilityTimeout    time.Duration
	extensionInterval    time.Duration
	timer                *time.Timer
}

func newVisibilityMonitor(logger *zap.Logger, msg types.Message, visibilityTimeout, extensionInterval, maxVisibilityWindow time.Duration) *visibilityMonitor {
	startTime := time.Now()
	firstExtensionTime := startTime.Add(getSafetyMargin(visibilityTimeout))

	return &visibilityMonitor{
		logger:               logger.With(zap.String("message_id", *msg.MessageId)),
		msg:                  msg,
		startTime:            startTime,
		maxVisibilityEndTime: startTime.Add(maxVisibilityWindow),
		visibilityTimeout:    visibilityTimeout,
		extensionInterval:    extensionInterval,
		timer:                time.NewTimer(time.Until(firstExtensionTime)),
	}
}

func getSafetyMargin(timeout time.Duration) time.Duration {
	return timeout * 80 / 100 // 80% of the timeout
}

func (vm *visibilityMonitor) stop() {
	vm.timer.Stop()
}

func (vm *visibilityMonitor) nextExtensionTimer() <-chan time.Time {
	return vm.timer.C
}

func (vm *visibilityMonitor) shouldExtendToMax() bool {
	return !time.Now().Add(vm.extensionInterval).Before(vm.maxVisibilityEndTime)
}

func (vm *visibilityMonitor) scheduleNextExtension(logger *zap.Logger) {
	now := time.Now()
	nextExtensionTime := now.Add(getSafetyMargin(vm.extensionInterval))
	logger.Debug("resetting visibility extension timer", zap.Duration("extension_interval", vm.extensionInterval), zap.Time("next_extension_time", nextExtensionTime))
	vm.timer.Reset(time.Until(nextExtensionTime))
}

func (vm *visibilityMonitor) getRemainingTime() time.Duration {
	return time.Until(vm.maxVisibilityEndTime)
}

func (vm *visibilityMonitor) getTotalVisibilityTime() time.Duration {
	return time.Since(vm.startTime)
}

func (w *Worker) extendToMaxAndStop(ctx context.Context, msg types.Message, queueURL string, monitor *visibilityMonitor, logger *zap.Logger) {
	remainingTime := monitor.getRemainingTime()

	logger.Info("reaching maximum visibility window, extending to max and stopping",
		zap.Duration("total_visibility_time", monitor.getTotalVisibilityTime()),
		zap.Duration("remaining_time", remainingTime),
		zap.Duration("max_window", monitor.maxVisibilityEndTime.Sub(monitor.startTime)))

	if err := w.extendVisibility(ctx, msg, queueURL, remainingTime, logger); err != nil {
		logger.Error("failed to extend message visibility to max", zap.Error(err), zap.Duration("attempted_timeout", remainingTime))
	}
	logger.Info("extended message visibility to max")
}

func (w *Worker) extendVisibility(ctx context.Context, msg types.Message, queueURL string, timeout time.Duration, logger *zap.Logger) error {
	changeParams := &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     msg.ReceiptHandle,
		VisibilityTimeout: int32(timeout.Seconds()),
	}
	logger.Debug("extending message visibility", zap.Duration("timeout", timeout))
	_, err := w.client.SQS().ChangeMessageVisibility(ctx, changeParams)
	if err != nil {
		logger.Error("failed to extend message visibility", zap.Error(err), zap.Duration("timeout", timeout))
	}
	logger.Debug("extended message visibility", zap.Duration("timeout", timeout))
	return err
}
