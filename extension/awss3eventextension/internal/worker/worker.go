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
package worker // import "github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/worker"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/aws/event"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// Worker processes S3 event notifications.
// It is responsible for processing messages from the SQS queue and sending them to the next consumer.
// It also handles deleting messages from the SQS queue after they have been processed.
// It is designed to be used in a worker pool.
type Worker struct {
	tel         component.TelemetrySettings
	client      client.Client
	unmarshaler event.Unmarshaler
	directory   string
}

// New creates a new Worker
func New(tel component.TelemetrySettings, cfg aws.Config, unmarshaler event.Unmarshaler, directory string) *Worker {
	client := client.NewClient(cfg)
	return &Worker{
		tel:         tel,
		client:      client,
		unmarshaler: unmarshaler,
		directory:   directory,
	}
}

// ProcessMessage processes a message from the SQS queue
// TODO add metric for number of messages processed / deleted / errors, events processed, etc.
func (w *Worker) ProcessMessage(ctx context.Context, msg types.Message, queueURL string, deferThis func()) {
	defer deferThis()

	w.tel.Logger.Debug("processing message", zap.String("message_id", *msg.MessageId), zap.String("body", *msg.Body))

	objects, err := w.unmarshaler.Unmarshal([]byte(*msg.Body))
	if err != nil {
		w.tel.Logger.Error("unmarshal notification", zap.Error(err))
		// We can delete messages with unmarshaling errors as they'll never succeed
		w.deleteMessage(ctx, msg, queueURL)
		return
	}
	w.tel.Logger.Debug("processing notification", zap.Int("event.count", len(objects)))

	if len(objects) == 0 {
		w.tel.Logger.Debug("no object created events found in notification, skipping", zap.String("message_id", *msg.MessageId))
		w.deleteMessage(ctx, msg, queueURL)
		return
	}

	if len(objects) > 1 {
		w.tel.Logger.Warn("duplicate logs possible: multiple s3:ObjectCreated:* events found in notification",
			zap.Int("event.count", len(objects)),
			zap.String("message_id", *msg.MessageId),
		)
	}

	var noSuchKeyError bool
	for _, object := range objects {
		w.tel.Logger.Debug("processing record",
			zap.String("bucket", object.Bucket),
			zap.String("key", object.Key),
		)

		if err := w.downloadObject(ctx, object); err != nil {
			if strings.Contains(err.Error(), "NoSuchKey") {
				noSuchKeyError = true
				w.tel.Logger.Warn("S3 object not found (404 NoSuchKey), preserving message for retry",
					zap.Error(err),
					zap.String("bucket", object.Bucket),
					zap.String("key", object.Key))
			} else {
				w.tel.Logger.Error("download object", zap.Error(err),
					zap.String("bucket", object.Bucket), zap.String("key", object.Key))
			}
		}
	}
	if noSuchKeyError {
		w.tel.Logger.Info("message preserved for retry due to NoSuchKey error", zap.String("message_id", *msg.MessageId))
	} else {
		w.deleteMessage(ctx, msg, queueURL)
	}
}

func (w *Worker) downloadObject(ctx context.Context, object event.S3Object) error {
	w.tel.Logger.Debug("reading S3 object",
		zap.String("bucket", object.Bucket), zap.String("key", object.Key), zap.Int64("size", object.Size))

	bucketDir := filepath.Join(w.directory, object.Bucket)
	if err := os.MkdirAll(bucketDir, 0700); err != nil {
		return fmt.Errorf("create bucket directory: %w", err)
	}

	filePath := filepath.Join(bucketDir, object.Key)

	filePathDir := filepath.Dir(filePath)
	filePathBase := filepath.Base(filePath)

	// The object could be nested in a directory structure
	if filePathDir != bucketDir {
		if err := os.MkdirAll(filePathDir, 0700); err != nil {
			return fmt.Errorf("create object directory: %w", err)
		}
	}

	// #nosec: G304
	tmpFile, err := os.Create(filepath.Join(filePathDir, filePathBase+".bptmp"))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	downloader := manager.NewDownloader(w.client.S3())
	numBytes, err := downloader.Download(ctx, tmpFile, &s3.GetObjectInput{
		Bucket: aws.String(object.Bucket),
		Key:    aws.String(object.Key),
	})
	if err != nil {
		return fmt.Errorf("download object: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if numBytes != object.Size {
		err := fmt.Errorf("size mismatch: expected %d, actual %d", object.Size, numBytes)
		if rmErr := os.Remove(tmpFile.Name()); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return err
	}

	if err := os.Rename(tmpFile.Name(), filePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	w.tel.Logger.Debug("processed S3 object", zap.String("bucket", object.Bucket), zap.String("key", object.Key))
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
