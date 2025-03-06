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
	"io"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// Worker processes S3 event notifications.
// It is responsible for processing messages from the SQS queue and sending them to the next consumer.
// It also handles deleting messages from the SQS queue after they have been processed.
// It is designed to be used in a worker pool.
type Worker struct {
	tel          component.TelemetrySettings
	client       bpaws.Client
	nextConsumer consumer.Logs
	maxLogSize   int
}

// New creates a new Worker
func New(tel component.TelemetrySettings, cfg aws.Config, nextConsumer consumer.Logs, maxLogSize int) *Worker {
	client := bpaws.NewClient(cfg)
	return &Worker{
		tel:          tel,
		client:       client,
		nextConsumer: nextConsumer,
		maxLogSize:   maxLogSize,
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
		return
	}
	w.tel.Logger.Debug("processing notification", zap.Int("event.count", len(notification.Records)))

	ld := plog.NewLogs()
	for _, record := range notification.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key
		w.tel.Logger.Debug("processing record", zap.String("bucket", bucket), zap.String("key", key))

		// Create new resource logs for each record
		rls := ld.ResourceLogs().AppendEmpty()
		rls.Resource().Attributes().PutStr("aws.s3.bucket", bucket)
		rls.Resource().Attributes().PutStr("aws.s3.key", key)
		sls := rls.ScopeLogs().AppendEmpty()

		w.processRecord(ctx, record, sls.LogRecords())
	}

	// TODO allow configuration of whether to delete messages before or after consuming the logs.
	// ConsumeLogs then Delete: If Delete fails, the message may be redelivered.
	// Delete then ConsumeLogs: If the ConsumeLogs fails, the logs will be lost.

	deleteParams := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	}
	_, err = w.client.SQS().DeleteMessage(ctx, deleteParams)
	if err != nil {
		w.tel.Logger.Error("delete message", zap.Error(err), zap.String("message_id", *msg.MessageId))
		return
	}
	w.tel.Logger.Debug("deleted message", zap.String("message_id", *msg.MessageId), zap.Int("event.count", len(notification.Records)))

	if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
		w.tel.Logger.Error("consume logs", zap.Error(err), zap.String("message_id", *msg.MessageId))
	}
}

func (w *Worker) processRecord(ctx context.Context, record events.S3EventRecord, lrs plog.LogRecordSlice) {
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
		return
	}
	defer resp.Body.Close()

	now := time.Now()

	reader := bufio.NewReader(resp.Body)

	for {
		// ReadLine returns line fragments if the line doesn't fit in the buffer
		lineBytes, _, err := reader.ReadLine()

		if err != nil {
			if err == io.EOF {
				break
			}
			w.tel.Logger.Error("reading object content", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
			return
		}

		if len(lineBytes) == 0 {
			continue
		}

		// Create a log record for this line fragment
		lr := lrs.AppendEmpty()
		lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(now))
		lr.SetTimestamp(pcommon.NewTimestampFromTime(record.EventTime))
		lr.Body().SetStr(string(lineBytes))
	}

	w.tel.Logger.Debug("processed S3 object", zap.String("bucket", bucket), zap.String("key", key))
}
