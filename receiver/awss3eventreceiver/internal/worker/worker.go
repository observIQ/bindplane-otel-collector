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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	tel            component.TelemetrySettings
	client         client.Client
	nextConsumer   consumer.Logs
	maxLogSize     int
	maxLogsEmitted int
}

// New creates a new Worker
func New(tel component.TelemetrySettings, cfg aws.Config, nextConsumer consumer.Logs, maxLogSize int, maxLogsEmitted int) *Worker {
	client := client.NewClient(cfg)
	return &Worker{
		tel:            tel,
		client:         client,
		nextConsumer:   nextConsumer,
		maxLogSize:     maxLogSize,
		maxLogsEmitted: maxLogsEmitted,
	}
}

// ProcessMessage processes a message from the SQS queue
// TODO add metric for number of messages processed / deleted / errors, events processed, etc.
func (w *Worker) ProcessMessage(ctx context.Context, msg types.Message, queueURL string, deferThis func()) {
	defer deferThis()

	w.tel.Logger.Debug("processing message", zap.String("message_id", *msg.MessageId), zap.String("body", *msg.Body))

	notification := new(events.S3Event)
	err := json.Unmarshal([]byte(*msg.Body), notification)
	if err != nil {
		w.tel.Logger.Error("unmarshal notification", zap.Error(err))
		// We can delete messages with unmarshaling errors as they'll never succeed
		w.deleteMessage(ctx, msg, queueURL)
		return
	}
	w.tel.Logger.Debug("processing notification", zap.Int("event.count", len(notification.Records)))

	// Filter records to only include s3:ObjectCreated:* events
	var objectCreatedRecords []events.S3EventRecord
	for _, record := range notification.Records {
		// S3 UI shows the prefix as "s3:ObjectCreated:", but the event name is unmarshalled as "ObjectCreated:"
		if strings.Contains(record.EventName, "ObjectCreated:") {
			objectCreatedRecords = append(objectCreatedRecords, record)
		} else {
			w.tel.Logger.Warn("unexpected event: receiver handles only s3:ObjectCreated:* events",
				zap.String("event_name", record.EventName),
				zap.String("bucket", record.S3.Bucket.Name),
				zap.String("key", record.S3.Object.Key))
		}
	}

	if len(objectCreatedRecords) == 0 {
		w.tel.Logger.Debug("no s3:ObjectCreated:* events found in notification, skipping", zap.String("message_id", *msg.MessageId))
		w.deleteMessage(ctx, msg, queueURL)
		return
	}

	if len(objectCreatedRecords) > 1 {
		w.tel.Logger.Warn("duplicate logs possible: multiple s3:ObjectCreated:* events found in notification",
			zap.Int("event.count", len(objectCreatedRecords)),
			zap.String("message_id", *msg.MessageId),
		)
	}

	for _, record := range objectCreatedRecords {
		w.tel.Logger.Debug("processing record",
			zap.String("bucket", record.S3.Bucket.Name),
			zap.String("key", record.S3.Object.Key),
		)

		if err := w.processRecord(ctx, record); err != nil {
			w.tel.Logger.Error("error processing record", zap.Error(err), zap.String("bucket", record.S3.Bucket.Name), zap.String("key", record.S3.Object.Key), zap.String("message_id", *msg.MessageId))
			return
		}
	}
	w.deleteMessage(ctx, msg, queueURL)
}

func (w *Worker) processRecord(ctx context.Context, record events.S3EventRecord) error {
	bucket := record.S3.Bucket.Name
	key := record.S3.Object.Key
	size := record.S3.Object.Size

	w.tel.Logger.Debug("reading S3 object", zap.String("bucket", bucket), zap.String("key", key), zap.Int64("size", size))

	resp, err := w.client.S3().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		w.tel.Logger.Error("get object", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
		return err
	}
	defer resp.Body.Close()

	now := time.Now()

	reader := bufio.NewReader(resp.Body)

	ld := plog.NewLogs()
	rls := ld.ResourceLogs().AppendEmpty()
	rls.Resource().Attributes().PutStr("aws.s3.bucket", bucket)
	rls.Resource().Attributes().PutStr("aws.s3.key", key)
	lrs := rls.ScopeLogs().AppendEmpty().LogRecords()

	batchesConsumedCount := 0
	for {
		// ReadLine returns line fragments if the line doesn't fit in the buffer
		lineBytes, _, err := reader.ReadLine()

		if err != nil {
			if err == io.EOF {
				break
			}
			w.tel.Logger.Error("reading object content", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
			return err
		}

		if len(lineBytes) == 0 {
			continue
		}

		// Create a log record for this line fragment
		lr := lrs.AppendEmpty()
		lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(now))
		lr.SetTimestamp(pcommon.NewTimestampFromTime(record.EventTime))
		lr.Body().SetStr(string(lineBytes))

		if ld.LogRecordCount() >= w.maxLogsEmitted {
			if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
				w.tel.Logger.Error("consume logs", zap.Error(err),
					zap.String("bucket", bucket),
					zap.String("key", key),
					zap.Int("batches_consumed_count", batchesConsumedCount),
				)
				return fmt.Errorf("consume logs: %w", err)
			}
			batchesConsumedCount++
			w.tel.Logger.Debug("Reached max logs for single batch, starting new batch", zap.String("bucket", bucket), zap.String("key", key), zap.Int("batches_consumed_count", batchesConsumedCount))

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
		w.tel.Logger.Error("consume logs", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key), zap.Int("batches_consumed_count", batchesConsumedCount))
		return fmt.Errorf("consume logs: %w", err)
	}
	w.tel.Logger.Debug("processed S3 object", zap.String("bucket", bucket), zap.String("key", key), zap.Int("batches_consumed_count", batchesConsumedCount+1))
	return nil
}

func (w *Worker) deleteMessage(ctx context.Context, msg types.Message, queueURL string) {
	deleteParams := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	}
	_, err := w.client.SQS().DeleteMessage(ctx, deleteParams)
	if err != nil {
		w.tel.Logger.Error("delete message", zap.Error(err), zap.String("message_id", *msg.MessageId))
		return
	}
	w.tel.Logger.Debug("deleted message", zap.String("message_id", *msg.MessageId))
}
