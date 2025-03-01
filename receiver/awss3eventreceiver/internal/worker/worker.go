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
	"io"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bps3"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpsqs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
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
	s3client     bps3.Client
	sqsClient    bpsqs.Client
	nextConsumer consumer.Logs
}

// New creates a new Worker
func New(tel component.TelemetrySettings, cfg aws.Config, nextConsumer consumer.Logs) *Worker {
	return &Worker{
		tel:          tel,
		s3client:     s3.NewFromConfig(cfg),
		sqsClient:    sqs.NewFromConfig(cfg),
		nextConsumer: nextConsumer,
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

	// Efficiently aggregate events into a single plog.Logs according to resource attributes
	// Since each resource has one scope, hash by resource attributes but store the scope
	resourceHashes := make(map[[16]byte]plog.ScopeLogs)
	for _, record := range notification.Records {
		resourceHash := pdatautil.Hash(
			pdatautil.WithString(record.S3.Bucket.Name),
			pdatautil.WithString(record.S3.Object.Key),
		)

		sls, ok := resourceHashes[resourceHash]
		if !ok {
			rls := ld.ResourceLogs().AppendEmpty()
			rls.Resource().Attributes().PutStr("aws.s3.bucket", record.S3.Bucket.Name)
			rls.Resource().Attributes().PutStr("aws.s3.key", record.S3.Object.Key)
			sls = rls.ScopeLogs().AppendEmpty()
			resourceHashes[resourceHash] = sls
		}

		lrs := w.processRecord(ctx, record)
		if lrs != nil {
			lrs.MoveAndAppendTo(sls.LogRecords())
		}
	}

	// TODO allow configuration of whether to delete messages before or after consuming the logs.
	// ConsumeLogs then Delete: If Delete fails, the message may be redelivered.
	// Delete then ConsumeLogs: If the ConsumeLogs fails, the logs will be lost.

	deleteParams := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	}
	_, err = w.sqsClient.DeleteMessage(ctx, deleteParams)
	if err != nil {
		w.tel.Logger.Error("delete message", zap.Error(err))
		return
	}
	w.tel.Logger.Debug("deleted message", zap.Int("event.count", len(notification.Records)))

	if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
		w.tel.Logger.Error("consume logs", zap.Error(err))
	}
}

func (w *Worker) processRecord(ctx context.Context, record events.S3EventRecord) *plog.LogRecordSlice {
	w.tel.Logger.Debug("reading S3 object",
		zap.String("bucket", record.S3.Bucket.Name),
		zap.String("key", record.S3.Object.Key),
		zap.Int64("size", record.S3.Object.Size),
	)

	resp, err := w.s3client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(record.S3.Bucket.Name),
		Key:    aws.String(record.S3.Object.Key),
	})
	if err != nil {
		w.tel.Logger.Error("get object", zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	// TODO switch to a streaming reader that tokenizes the object content
	// and sends the log records to the channel in batches
	data := make([]byte, record.S3.Object.Size)
	_, err = io.ReadFull(resp.Body, data)
	if err != nil {
		w.tel.Logger.Error("read object", zap.Error(err))
		return nil
	}

	lrs := plog.NewLogRecordSlice()
	lr := lrs.AppendEmpty()
	lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	// TODO set timestamp from event.Time
	lr.Body().SetStr(string(data))
	return &lrs
}
