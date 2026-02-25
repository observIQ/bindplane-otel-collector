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

// Package worker provides a worker that processes GCS event notifications from Pub/Sub.
package worker // import "github.com/observiq/bindplane-otel-collector/receiver/gcspubeventreceiver/internal/worker"

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
	"google.golang.org/api/googleapi"

	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubeventreceiver/internal/metadata"
)

// GCS Pub/Sub notification event types
const (
	EventTypeObjectFinalize = "OBJECT_FINALIZE"
)

// GCS Pub/Sub message attribute keys
const (
	AttrBucketID  = "bucketId"
	AttrObjectID  = "objectId"
	AttrEventType = "eventType"
)

// Worker processes GCS event notifications from Pub/Sub messages.
type Worker struct {
	logger           *zap.Logger
	tel              component.TelemetrySettings
	storageClient    *storage.Client
	nextConsumer     consumer.Logs
	offsetStorage    storageclient.StorageClient
	maxLogSize       int
	maxLogsEmitted   int
	metrics          *metadata.TelemetryBuilder
	bucketNameFilter *regexp.Regexp
	objectKeyFilter  *regexp.Regexp
	obsrecv          *receiverhelper.ObsReport
}

// Option is a functional option for configuring the Worker
type Option func(*Worker)

// WithBucketNameFilter sets the bucket name filter regex
func WithBucketNameFilter(filter *regexp.Regexp) Option {
	return func(w *Worker) {
		if filter != nil {
			w.bucketNameFilter = filter
		}
	}
}

// WithObjectKeyFilter sets the object key filter regex
func WithObjectKeyFilter(filter *regexp.Regexp) Option {
	return func(w *Worker) {
		if filter != nil {
			w.objectKeyFilter = filter
		}
	}
}

// WithTelemetryBuilder sets the telemetry builder
func WithTelemetryBuilder(tb *metadata.TelemetryBuilder) Option {
	return func(w *Worker) {
		if tb != nil {
			w.metrics = tb
		}
	}
}

// New creates a new Worker
func New(tel component.TelemetrySettings, nextConsumer consumer.Logs, storageClient *storage.Client, obsrecv *receiverhelper.ObsReport, maxLogSize int, maxLogsEmitted int, opts ...Option) *Worker {
	w := &Worker{
		logger:         tel.Logger.With(zap.String("component", "gcspubeventreceiver")),
		tel:            tel,
		storageClient:  storageClient,
		nextConsumer:   nextConsumer,
		offsetStorage:  storageclient.NewNopStorage(),
		obsrecv:        obsrecv,
		maxLogSize:     maxLogSize,
		maxLogsEmitted: maxLogsEmitted,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// SetOffsetStorage sets the offset storage client
func (w *Worker) SetOffsetStorage(offsetStorage storageclient.StorageClient) {
	w.offsetStorage = offsetStorage
}

// ProcessMessage processes a Pub/Sub message containing a GCS event notification
func (w *Worker) ProcessMessage(ctx context.Context, msg *pubsub.Message) {
	logger := w.logger.With(zap.String("message_id", msg.ID))

	// Parse event attributes from the Pub/Sub message
	eventType := msg.Attributes[AttrEventType]
	bucketID := msg.Attributes[AttrBucketID]
	objectID := msg.Attributes[AttrObjectID]

	logger = logger.With(
		zap.String("event_type", eventType),
		zap.String("bucket", bucketID),
		zap.String("object", objectID),
	)

	// Filter for OBJECT_FINALIZE events only
	if eventType != EventTypeObjectFinalize {
		logger.Debug("skipping non-OBJECT_FINALIZE event")
		msg.Ack()
		return
	}

	// Validate required attributes
	if bucketID == "" || objectID == "" {
		logger.Warn("message missing required attributes (bucketId, objectId)")
		msg.Ack()
		return
	}

	// Apply bucket name filter
	if w.bucketNameFilter != nil && !w.bucketNameFilter.MatchString(bucketID) {
		logger.Debug("skipping message due to bucket name filter")
		msg.Ack()
		return
	}

	// Apply object key filter
	if w.objectKeyFilter != nil && !w.objectKeyFilter.MatchString(objectID) {
		logger.Debug("skipping message due to object key filter")
		msg.Ack()
		return
	}

	logger.Debug("processing GCS object")

	// Process the record, trying JSON first then falling back to line parsing
	err := w.processRecord(ctx, bucketID, objectID, logger)
	if err != nil {
		w.handleProcessingError(ctx, msg, err, logger)
		return
	}

	w.metrics.GcseventObjectsHandled.Add(ctx, 1)

	// Clean up offset storage for the processed object
	offsetStorageKey := fmt.Sprintf("%s_%s", OffsetStorageKey, objectID)
	if err := w.offsetStorage.DeleteStorageData(ctx, offsetStorageKey); err != nil {
		logger.Error("failed to delete offset", zap.Error(err), zap.String("offset_storage_key", offsetStorageKey))
	}

	msg.Ack()
	logger.Debug("message acked")
}

func (w *Worker) processRecord(ctx context.Context, bucket, object string, recordLogger *zap.Logger) error {
	err := w.consumeLogsFromGCSObject(ctx, bucket, object, true, recordLogger)
	if err != nil {
		if errors.Is(err, ErrNotArrayOrKnownObject) {
			// try again without attempting to parse as JSON
			recordLogger.Debug("parsing as JSON failed, trying again with line parsing")
			return w.consumeLogsFromGCSObject(ctx, bucket, object, false, recordLogger)
		}
		return err
	}
	return nil
}

func (w *Worker) consumeLogsFromGCSObject(ctx context.Context, bucket, object string, tryJSON bool, recordLogger *zap.Logger) error {
	recordLogger.Debug("reading GCS object")

	obj := w.storageClient.Bucket(bucket).Object(object)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer reader.Close()

	now := time.Now()

	// Get content type from object attributes
	var contentType *string
	if reader.Attrs.ContentType != "" {
		ct := reader.Attrs.ContentType
		contentType = &ct
	}
	var contentEncoding *string
	if reader.Attrs.ContentEncoding != "" {
		ce := reader.Attrs.ContentEncoding
		contentEncoding = &ce
	}

	stream := LogStream{
		Name:            object,
		ContentEncoding: contentEncoding,
		ContentType:     contentType,
		Body:            reader,
		MaxLogSize:      w.maxLogSize,
		Logger:          recordLogger,
		TryDecoding:     tryJSON,
	}

	// Create the offset storage key for this object
	offsetStorageKey := fmt.Sprintf("%s_%s", OffsetStorageKey, object)

	// Load the offset from storage
	offset := NewOffset(0)
	err = w.offsetStorage.LoadStorageData(ctx, offsetStorageKey, offset)
	if err != nil {
		return fmt.Errorf("load offset: %w", err)
	}
	startOffset := offset.Offset

	if startOffset == 0 {
		recordLogger.Debug("no offset found, starting from beginning", zap.String("offset_storage_key", offsetStorageKey))
	} else {
		recordLogger.Debug("loaded offset", zap.String("offset_storage_key", offsetStorageKey), zap.Int64("offset", startOffset))
	}

	bufferedReader, err := stream.BufferedReader(ctx)
	if err != nil {
		return fmt.Errorf("get stream reader: %w", err)
	}

	parser, err := newParser(ctx, stream, bufferedReader)
	if err != nil {
		return fmt.Errorf("create parser: %w", err)
	}

	ld := plog.NewLogs()
	rls := ld.ResourceLogs().AppendEmpty()
	rls.Resource().Attributes().PutStr("gcs.bucket", bucket)
	rls.Resource().Attributes().PutStr("gcs.object", object)
	lrs := rls.ScopeLogs().AppendEmpty().LogRecords()

	batchesConsumedCount := 0

	// Parse logs into a sequence of log records
	logs, err := parser.Parse(ctx, startOffset)
	if err != nil {
		return fmt.Errorf("parse logs: %w", err)
	}

	for log, err := range logs {
		if err != nil {
			// Skipping the individual record rather than nacking the whole message, since
			// retrying a malformed record would produce the same error.  The remaining
			// records in the object can still be ingested successfully.
			recordLogger.Error("parse log", zap.Error(err))
			w.metrics.GcseventParseErrors.Add(ctx, 1)
			continue
		}

		// Create a log record for this line fragment
		lr := lrs.AppendEmpty()
		lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(now))
		lr.SetTimestamp(pcommon.NewTimestampFromTime(now))

		err = parser.AppendLogBody(ctx, lr, log)
		if err != nil {
			// Same rationale as above: skip the record rather than failing the whole object.
			recordLogger.Error("append log body", zap.Error(err))
			w.metrics.GcseventParseErrors.Add(ctx, 1)
			continue
		}

		if ld.LogRecordCount() >= w.maxLogsEmitted {
			obsCtx := w.obsrecv.StartLogsOp(ctx)
			if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
				w.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), ld.LogRecordCount(), err)
				recordLogger.Error("consume logs", zap.Error(err), zap.Int("batches_consumed_count", batchesConsumedCount))
				return fmt.Errorf("consume logs: %w", err)
			}
			w.metrics.GcseventBatchSize.Record(ctx, int64(ld.LogRecordCount()))
			w.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), ld.LogRecordCount(), nil)

			batchesConsumedCount++
			recordLogger.Debug("Reached max logs for single batch, starting new batch", zap.Int("batches_consumed_count", batchesConsumedCount))

			// Save the offset to storage
			if err := w.offsetStorage.SaveStorageData(ctx, offsetStorageKey, NewOffset(parser.Offset())); err != nil {
				recordLogger.Error("Failed to save offset", zap.Error(err), zap.String("offset_storage_key", offsetStorageKey), zap.Int64("offset", parser.Offset()))
			}

			ld = plog.NewLogs()
			rls = ld.ResourceLogs().AppendEmpty()
			rls.Resource().Attributes().PutStr("gcs.bucket", bucket)
			rls.Resource().Attributes().PutStr("gcs.object", object)
			lrs = rls.ScopeLogs().AppendEmpty().LogRecords()
		}
	}

	if ld.LogRecordCount() == 0 {
		return nil
	}
	w.metrics.GcseventBatchSize.Record(ctx, int64(ld.LogRecordCount()))

	obsCtx := w.obsrecv.StartLogsOp(ctx)
	if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
		w.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), ld.LogRecordCount(), err)
		recordLogger.Error("consume logs", zap.Error(err), zap.Int("batches_consumed_count", batchesConsumedCount))
		return fmt.Errorf("consume logs: %w", err)
	}
	w.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), ld.LogRecordCount(), nil)
	recordLogger.Debug("processed GCS object", zap.Int("batches_consumed_count", batchesConsumedCount+1))

	// Save the offset to storage
	if err := w.offsetStorage.SaveStorageData(ctx, offsetStorageKey, NewOffset(parser.Offset())); err != nil {
		recordLogger.Error("Failed to save offset", zap.Error(err), zap.String("offset_storage_key", offsetStorageKey), zap.Int64("offset", parser.Offset()))
	}

	return nil
}

// dlqErrorKind categorizes an error into a DLQ condition bucket.
type dlqErrorKind int

const (
	dlqErrorKindNone dlqErrorKind = iota
	dlqErrorKindFileNotFound
	dlqErrorKindPermissionDenied
	dlqErrorKindUnsupportedFile
)

// dlqConditionKind returns the DLQ error kind for the given error, or
// dlqErrorKindNone if the error does not trigger a DLQ condition.
func dlqConditionKind(err error) dlqErrorKind {
	if err == nil {
		return dlqErrorKindNone
	}
	// GCS returns storage.ErrObjectNotExist when the object is not found.
	if errors.Is(err, storage.ErrObjectNotExist) {
		return dlqErrorKindFileNotFound
	}
	// GCS returns *googleapi.Error with Code 403 for permission denied.
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 403 {
		return dlqErrorKindPermissionDenied
	}
	// Unsupported file type detected during parsing.
	if errors.Is(err, ErrNotArrayOrKnownObject) {
		return dlqErrorKindUnsupportedFile
	}
	return dlqErrorKindNone
}

// isDLQConditionError checks if an error should trigger DLQ behavior.
func isDLQConditionError(err error) bool {
	return dlqConditionKind(err) != dlqErrorKindNone
}

// recordDLQMetrics records metrics for DLQ conditions based on the error type.
func (w *Worker) recordDLQMetrics(ctx context.Context, err error) {
	if w.metrics == nil {
		return
	}

	switch dlqConditionKind(err) {
	case dlqErrorKindFileNotFound:
		w.metrics.GcseventDlqFileNotFoundErrors.Add(ctx, 1)
	case dlqErrorKindPermissionDenied:
		w.metrics.GcseventDlqIamErrors.Add(ctx, 1)
	case dlqErrorKindUnsupportedFile:
		w.metrics.GcseventDlqUnsupportedFileErrors.Add(ctx, 1)
	default:
		w.metrics.GcseventFailures.Add(ctx, 1)
	}
}

// handleProcessingError handles errors from processing records
func (w *Worker) handleProcessingError(ctx context.Context, msg *pubsub.Message, err error, logger *zap.Logger) {
	if isDLQConditionError(err) {
		logger.Error("DLQ condition triggered, nacking message for redelivery/DLQ processing", zap.Error(err))
		w.recordDLQMetrics(ctx, err)
		msg.Nack()
		return
	}
	logger.Error("error processing record, nacking message for retry", zap.Error(err))
	w.metrics.GcseventFailures.Add(ctx, 1)
	msg.Nack()
}
