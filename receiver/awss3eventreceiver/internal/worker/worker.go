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
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/constants"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/metadata"
)

// AWS error codes for DLQ condition detection
const (
	AWSErrorCodeAccessDenied = "AccessDenied"
	AWSErrorCodeForbidden    = "Forbidden"
	AWSErrorCodeNoSuchKey    = "NoSuchKey"
)

// parseFunc defines the signature for parsing notification messages into S3 events
type parseFunc func(messageBody string) (*events.S3Event, error)

// getParseFuncForNotificationType returns the appropriate parse function for the given notification type
func (w *Worker) getParseFuncForNotificationType(notificationType string) parseFunc {
	switch notificationType {
	case constants.NotificationTypeSNS:
		return ParseSNSToS3Event
	case constants.NotificationTypeS3:
		fallthrough
	default:
		return parseS3Event
	}
}

// parseS3Event parses a direct S3 event notification
func parseS3Event(messageBody string) (*events.S3Event, error) {
	notification := new(events.S3Event)
	err := json.Unmarshal([]byte(messageBody), notification)
	return notification, err
}

// isDLQConditionError checks if an error should trigger DLQ behavior and returns the specific error type
func isDLQConditionError(err error) error {
	if isAccessDeniedError(err) {
		return &DLQError{Type: "iam_permission_denied", Err: err}
	}
	if isNoSuchKeyError(err) {
		return &DLQError{Type: "file_not_found", Err: err}
	}
	if isUnsupportedFileTypeError(err) {
		return &DLQError{Type: "unsupported_file_type", Err: err}
	}
	return nil
}

// isAccessDeniedError checks if the error is an IAM permission (AccessDenied) error
func isAccessDeniedError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == AWSErrorCodeAccessDenied || apiErr.ErrorCode() == AWSErrorCodeForbidden
	}
	// Also check for string-based errors
	errStr := err.Error()
	return strings.Contains(errStr, AWSErrorCodeAccessDenied) || strings.Contains(errStr, AWSErrorCodeForbidden)
}

// isNoSuchKeyError checks if the error is a file not found (NoSuchKey) error
func isNoSuchKeyError(err error) bool {
	var noSuchKeyErr *s3types.NoSuchKey
	if errors.As(err, &noSuchKeyErr) {
		return true
	}
	// Also check for string-based errors
	errStr := err.Error()
	return strings.Contains(errStr, AWSErrorCodeNoSuchKey)
}

// isUnsupportedFileTypeError checks if the error indicates an unsupported file type
func isUnsupportedFileTypeError(err error) bool {
	return errors.Is(err, ErrNotArrayOrKnownObject)
}

// DLQError represents an error that should trigger DLQ behavior
type DLQError struct {
	Type string
	Err  error
}

func (e *DLQError) Error() string {
	return e.Err.Error()
}

func (e *DLQError) Unwrap() error {
	return e.Err
}

// Worker processes S3 event notifications.
// It is responsible for processing messages from the SQS queue and sending them to the next consumer.
// It also handles deleting messages from the SQS queue after they have been processed.
// It is designed to be used in a worker pool.
type Worker struct {
	logger                      *zap.Logger
	tel                         component.TelemetrySettings
	client                      client.Client
	nextConsumer                consumer.Logs
	offsetStorage               storageclient.StorageClient
	maxLogSize                  int
	maxLogsEmitted              int
	visibilityTimeout           time.Duration
	visibilityExtensionInterval time.Duration
	maxVisibilityWindow         time.Duration
	metrics                     *metadata.TelemetryBuilder
	bucketNameFilter            *regexp.Regexp
	objectKeyFilter             *regexp.Regexp
	notificationType            string
	parseFunc                   parseFunc
}

// Option is a functional option for configuring the Worker
type Option func(*Worker)

// WithBucketNameFilter sets the bucket name filter regex
func WithBucketNameFilter(filter *regexp.Regexp) Option {
	return func(w *Worker) {
		w.bucketNameFilter = filter
	}
}

// WithObjectKeyFilter sets the object key filter regex
func WithObjectKeyFilter(filter *regexp.Regexp) Option {
	return func(w *Worker) {
		w.objectKeyFilter = filter
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

// WithNotificationType sets the notification type
func WithNotificationType(notificationType string) Option {
	return func(w *Worker) {
		w.notificationType = notificationType
	}
}

// New creates a new Worker
func New(tel component.TelemetrySettings, nextConsumer consumer.Logs, client client.Client, maxLogSize int, maxLogsEmitted int, visibilityTimeout time.Duration, visibilityExtensionInterval time.Duration, maxVisibilityWindow time.Duration, opts ...Option) *Worker {
	w := &Worker{
		logger:                      tel.Logger.With(zap.String("component", "awss3eventreceiver")),
		tel:                         tel,
		client:                      client,
		nextConsumer:                nextConsumer,
		offsetStorage:               storageclient.NewNopStorage(),
		maxLogSize:                  maxLogSize,
		maxLogsEmitted:              maxLogsEmitted,
		visibilityTimeout:           visibilityTimeout,
		visibilityExtensionInterval: visibilityExtensionInterval,
		maxVisibilityWindow:         maxVisibilityWindow,
		notificationType:            constants.NotificationTypeS3, // Default to S3 notification type
	}

	for _, opt := range opts {
		opt(w)
	}

	// Set the parse function based on the notification type
	w.parseFunc = w.getParseFuncForNotificationType(w.notificationType)

	return w
}

// SetOffsetStorage sets the offset storage client
func (w *Worker) SetOffsetStorage(offsetStorage storageclient.StorageClient) {
	w.offsetStorage = offsetStorage
}

// ProcessMessage processes a message from the SQS queue
func (w *Worker) ProcessMessage(ctx context.Context, msg types.Message, queueURL string, deferThis func()) {
	defer deferThis()

	logger := w.logger.With(zap.String("message_id", *msg.MessageId), zap.String("queue_url", queueURL))

	// Start a goroutine to periodically extend message visibility
	visibilityCtx, cancelVisibility := context.WithCancel(ctx)
	defer cancelVisibility()

	go w.extendMessageVisibility(visibilityCtx, msg, queueURL, logger)
	// Parse the message using the configured parse function
	logger.Debug("parsing message", zap.String("message_id", *msg.MessageId), zap.String("notification_type", w.notificationType))
	notification, err := w.parseFunc(*msg.Body)
	if err != nil {
		logger.Error("unmarshal notification", zap.Error(err))
		w.metrics.S3eventFailures.Add(ctx, 1)
		// We can delete messages with unmarshaling errors as they'll never succeed
		w.deleteMessage(ctx, msg, queueURL, []string{}, logger)
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
		if !strings.Contains(record.EventName, "ObjectCreated:") {
			recordLogger.Warn("unexpected event: receiver handles only s3:ObjectCreated:* events",
				zap.String("event_name", record.EventName),
				zap.String("bucket", record.S3.Bucket.Name),
				zap.String("key", record.S3.Object.Key))
			continue
		}

		if w.bucketNameFilter != nil && !w.bucketNameFilter.MatchString(record.S3.Bucket.Name) {
			recordLogger.Debug("skipping record due to bucket name filter", zap.String("bucket", record.S3.Bucket.Name))
			continue
		}
		if w.objectKeyFilter != nil && !w.objectKeyFilter.MatchString(record.S3.Object.Key) {
			recordLogger.Debug("skipping record due to object key filter", zap.String("key", record.S3.Object.Key))
			continue
		}
		objectCreatedRecords = append(objectCreatedRecords, record)
	}

	if len(objectCreatedRecords) == 0 {
		logger.Debug("no s3:ObjectCreated:* events passed filters, skipping", zap.String("message_id", *msg.MessageId))
		w.deleteMessage(ctx, msg, queueURL, []string{}, logger)
		return
	}

	if len(objectCreatedRecords) > 1 {
		logger.Warn("duplicate logs possible: multiple s3:ObjectCreated:* events found in notification", zap.Int("event.count", len(objectCreatedRecords)))
	}

	var keys []string
	for _, record := range objectCreatedRecords {
		recordLogger := logger.With(zap.String("bucket", record.S3.Bucket.Name), zap.String("key", record.S3.Object.Key))
		recordLogger.Debug("processing record")

		if err := w.processRecord(ctx, record, recordLogger); err != nil {
			w.handleProcessingError(ctx, msg, queueURL, err, recordLogger)
			return
		}
		keys = append(keys, record.S3.Object.Key)
		w.metrics.S3eventObjectsHandled.Add(ctx, 1)
	}
	w.deleteMessage(ctx, msg, queueURL, keys, logger)
}

func (w *Worker) processRecord(ctx context.Context, record events.S3EventRecord, recordLogger *zap.Logger) error {
	err := w.consumeLogsFromS3Object(ctx, record, true, recordLogger)
	if err != nil {
		if errors.Is(err, ErrNotArrayOrKnownObject) {
			// try again without attempting to parse as JSON
			recordLogger.Debug("parsing as JSON failed, trying again with line parsing")
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

	stream := LogStream{
		Name:            key,
		ContentEncoding: resp.ContentEncoding,
		ContentType:     resp.ContentType,
		Body:            resp.Body,
		MaxLogSize:      w.maxLogSize,
		Logger:          recordLogger,
		TryDecoding:     tryJSON,
	}

	// Create the offset storage key for this object
	offsetStorageKey := fmt.Sprintf("%s_%s", OffsetStorageKey, key)

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
	logs, err := parser.Parse(ctx, startOffset)
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
			w.metrics.S3eventBatchSize.Record(ctx, int64(ld.LogRecordCount()))

			batchesConsumedCount++
			recordLogger.Debug("Reached max logs for single batch, starting new batch", zap.Int("batches_consumed_count", batchesConsumedCount))

			// Save the offset to storage
			if err := w.offsetStorage.SaveStorageData(ctx, offsetStorageKey, NewOffset(parser.Offset())); err != nil {
				recordLogger.Error("Failed to save offset", zap.Error(err), zap.String("offset_storage_key", offsetStorageKey), zap.Int64("offset", parser.Offset()))
			}

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
	w.metrics.S3eventBatchSize.Record(ctx, int64(ld.LogRecordCount()))

	if err := w.nextConsumer.ConsumeLogs(ctx, ld); err != nil {
		recordLogger.Error("consume logs", zap.Error(err), zap.Int("batches_consumed_count", batchesConsumedCount))
		return fmt.Errorf("consume logs: %w", err)
	}
	recordLogger.Debug("processed S3 object", zap.Int("batches_consumed_count", batchesConsumedCount+1))

	// Save the offset to storage
	if err := w.offsetStorage.SaveStorageData(ctx, offsetStorageKey, NewOffset(parser.Offset())); err != nil {
		recordLogger.Error("Failed to save offset", zap.Error(err), zap.String("offset_storage_key", offsetStorageKey), zap.Int64("offset", parser.Offset()))
	}

	return nil
}

func (w *Worker) deleteMessage(ctx context.Context, msg types.Message, queueURL string, keys []string, recordLogger *zap.Logger) {
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

	// Delete the offsets for the keys that were processed
	for _, key := range keys {
		offsetStorageKey := fmt.Sprintf("%s_%s", OffsetStorageKey, key)
		if err := w.offsetStorage.DeleteStorageData(ctx, offsetStorageKey); err != nil {
			recordLogger.Error("Failed to delete offset", zap.Error(err), zap.String("offset_storage_key", offsetStorageKey))
		}
	}
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
	return timeout * 50 / 100 // 50% of the timeout
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
}

func (w *Worker) extendVisibility(ctx context.Context, msg types.Message, queueURL string, timeout time.Duration, logger *zap.Logger) error {
	changeParams := &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     msg.ReceiptHandle,
		VisibilityTimeout: int32(timeout.Seconds()),
	}
	logger.Debug("extending message visibility", zap.Duration("timeout", timeout))
	_, err := w.client.SQS().ChangeMessageVisibility(ctx, changeParams)
	return err
}

// recordDLQMetrics records metrics for DLQ conditions based on the error type.
func (w *Worker) recordDLQMetrics(ctx context.Context, errorType string) {
	if w.metrics == nil {
		return
	}

	switch errorType {
	case "iam_permission_denied":
		w.metrics.S3eventDlqIamErrors.Add(ctx, 1)
	case "file_not_found":
		w.metrics.S3eventDlqFileNotFoundErrors.Add(ctx, 1)
	case "unsupported_file_type":
		w.metrics.S3eventDlqUnsupportedFileErrors.Add(ctx, 1)
	default:
		// General failure metric for unknown errors
		w.metrics.S3eventFailures.Add(ctx, 1)
	}
}

// handleDLQCondition handles messages that should be sent to DLQ by resetting visibility and logging
func (w *Worker) handleDLQCondition(ctx context.Context, msg types.Message, queueURL string, err error, logger *zap.Logger) {
	var errorType string
	if err != nil {
		var dlqErr *DLQError
		if errors.As(err, &dlqErr) {
			errorType = dlqErr.Type
			logger.Error("DLQ condition triggered, resetting visibility for DLQ processing",
				zap.Error(dlqErr.Err),
				zap.String("error_type", errorType))
			w.recordDLQMetrics(ctx, errorType)
		} else {
			// Fallback for other errors
			errorType = "unknown_dlq_error"
			logger.Error("DLQ condition triggered for unknown error, resetting visibility for DLQ processing",
				zap.Error(err),
				zap.String("error_type", errorType))
			w.recordDLQMetrics(ctx, errorType)
		}
	}

	if err := w.resetVisibilityTimeout(ctx, msg, queueURL, logger); err != nil {
		logger.Error("failed to reset visibility timeout for DLQ condition", zap.Error(err))
	}
}

// handleProcessingError handles errors from processing records, determining if they should trigger DLQ behavior
func (w *Worker) handleProcessingError(ctx context.Context, msg types.Message, queueURL string, err error, logger *zap.Logger) {
	if dlqErr := isDLQConditionError(err); dlqErr != nil {
		w.handleDLQCondition(ctx, msg, queueURL, dlqErr, logger)
		return
	}
	logger.Error("error processing record, preserving message in SQS for retry", zap.Error(err))
	w.metrics.S3eventFailures.Add(ctx, 1)
}

// resetVisibilityTimeout resets the message visibility timeout to 0, making it immediately available for DLQ processing
func (w *Worker) resetVisibilityTimeout(ctx context.Context, msg types.Message, queueURL string, logger *zap.Logger) error {
	changeParams := &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     msg.ReceiptHandle,
		VisibilityTimeout: 0, // Reset to 0 to make message immediately available
	}
	logger.Debug("resetting message visibility timeout for DLQ processing")
	_, err := w.client.SQS().ChangeMessageVisibility(ctx, changeParams)
	return err
}
