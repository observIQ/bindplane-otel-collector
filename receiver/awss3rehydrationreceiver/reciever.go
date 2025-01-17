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

package awss3rehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/awss3rehydrationreceiver"

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3rehydrationreceiver/internal/aws"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
)

// newAWSS3Client is the function used to create new AWS S3 clients.
// Meant to be overwritten for tests
var newAWSS3Client = aws.NewAWSClient

// checkpointStorageKey is the key used for storing the checkpoint
const checkpointStorageKey = "aws_s3_checkpoint"

type rehydrationReceiver struct {
	logger             *zap.Logger
	id                 component.ID
	cfg                *Config
	awsClient          aws.S3Client
	supportedTelemetry pipeline.Signal
	consumer           rehydration.Consumer
	wg                 *sync.WaitGroup
	doneChan           chan struct{}
	objectChan         chan *aws.ObjectResults
	errChan            chan error

	checkpoint      *rehydration.CheckPoint
	checkpointStore rehydration.CheckpointStorer
	checkpointKey   string
	checkpointMutex *sync.Mutex

	lastObjectTime *time.Time
	lastObjectName string

	startingTime time.Time
	endingTime   time.Time

	cancelFunc context.CancelFunc
}

// newMetricsReceiver creates a new metrics specific receiver.
func newMetricsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Metrics) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalMetrics
	r.consumer = rehydration.NewMetricsConsumer(nextConsumer)

	return r, nil
}

// newLogsReceiver creates a new logs specific receiver.
func newLogsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Logs) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalLogs
	r.consumer = rehydration.NewLogsConsumer(nextConsumer)

	return r, nil
}

// newTracesReceiver creates a new traces specific receiver.
func newTracesReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Traces) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalTraces
	r.consumer = rehydration.NewTracesConsumer(nextConsumer)

	return r, nil
}

// newRehydrationReceiver creates a new rehydration receiver
func newRehydrationReceiver(id component.ID, logger *zap.Logger, cfg *Config) (*rehydrationReceiver, error) {
	awsClient, err := newAWSS3Client(logger, cfg.Region, cfg.PollSize, cfg.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("new aws s3 client: %w", err)
	}

	// We should not get an error for either of these time parsings as we check in config validate.
	// Doing error checking anyways just in case.
	startingTime, err := time.Parse(rehydration.TimeFormat, cfg.StartingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid starting_time timestamp: %w", err)
	}

	endingTime, err := time.Parse(rehydration.TimeFormat, cfg.EndingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid ending_time timestamp: %w", err)
	}

	return &rehydrationReceiver{
		logger:          logger,
		id:              id,
		cfg:             cfg,
		awsClient:       awsClient,
		checkpointStore: rehydration.NewNopStorage(),
		startingTime:    startingTime,
		endingTime:      endingTime,
		doneChan:        make(chan struct{}),
		wg:              &sync.WaitGroup{},
		checkpointMutex: &sync.Mutex{},
		objectChan:      make(chan *aws.ObjectResults),
		errChan:         make(chan error),
	}, nil
}

// Start starts the rehydration receiver
func (r *rehydrationReceiver) Start(ctx context.Context, host component.Host) error {
	r.logDeprecationWarnings()

	if err := r.initCheckpoint(ctx, host); err != nil {
		return fmt.Errorf("init checkpoint: %w", err)
	}

	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	r.cancelFunc = cancelFunc

	go r.stream(cancelCtx)
	return nil
}

// Shutdown shuts down the rehydration receiver
func (r *rehydrationReceiver) Shutdown(ctx context.Context) error {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}
	close(r.doneChan)

	// wait for any in-progress object rehydrations to finish
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case <-done:
	case <-shutdownCtx.Done():
		return fmt.Errorf("shutdown timeout: %w", shutdownCtx.Err())
	}

	var errs error
	if err := r.handleCheckpoint(ctx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("handle checkpoint: %w", err))
	}
	if err := r.checkpointStore.Close(ctx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("close checkpoint store: %w", err))
	}
	return errs
}

// logDeprecationWarnings logs errors about deprecated parameters and should be removed when deprecation is completed.
func (r *rehydrationReceiver) logDeprecationWarnings() {
	if r.cfg.PollInterval != 0 {
		r.logger.Warn("poll_interval is no longer recognized and will be removed in a future release. poll_size/batch_size should be used instead")
	}
	if r.cfg.PollTimeout != 0 {
		r.logger.Warn("poll_timeout is no longer recognized and will be removed in a future release. poll_size/batch_size should be used instead")
	}
}

// initCheckpoint initializes a checkpoint store in an extension if configured & retrieves first checkpoint
func (r *rehydrationReceiver) initCheckpoint(ctx context.Context, host component.Host) error {
	// init checkpoint storage using storage ext if configured
	if r.cfg.StorageID != nil {
		checkpointStore, err := rehydration.NewCheckpointStorage(ctx, host, *r.cfg.StorageID, r.id, r.supportedTelemetry)
		if err != nil {
			return fmt.Errorf("new rehydration checkpoint storage: %w", err)
		}
		r.checkpointStore = checkpointStore
	}

	// create static checkpoint key used for storing
	r.checkpointKey = fmt.Sprintf("%s_%s_%s", checkpointStorageKey, r.id, r.supportedTelemetry.String())

	// load the previous checkpoint. If not exist should return zero value for time
	checkpoint, err := r.checkpointStore.LoadCheckPoint(ctx, r.checkpointKey)
	if err != nil {
		r.logger.Warn("Error loading checkpoint, continuing without a previous checkpoint", zap.Error(err))
		checkpoint = rehydration.NewCheckpoint()
	}
	r.checkpoint = checkpoint

	return nil
}

// stream uses the S3 client to process batches of objects as they are sent down a channel
func (r *rehydrationReceiver) stream(ctx context.Context) {
	startTime := time.Now()
	r.logger.Info("Starting rehydration", zap.Time("start time", startTime))

	go r.awsClient.StreamObjects(ctx, r.cfg.S3Bucket, r.cfg.S3Prefix, r.objectChan, r.errChan, r.doneChan)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context finished, stopping rehydration", zap.Duration("duration", time.Since(startTime)))
			return
		case <-r.doneChan:
			r.logger.Info("Finished rehydrating objects", zap.Duration("duration", time.Since(startTime)))
			return
		case err := <-r.errChan:
			r.logger.Error("Error streaming objects, stopping rehydration", zap.Error(err), zap.Duration("duration", time.Since(startTime)))
			return
		case o := <-r.objectChan:
			r.logger.Info("Rehydrating new batch of objects", zap.Int("number of objects", len(o.Objects)))
			r.rehydrate(ctx, o.Objects)
		}
	}
}

// rehydrate will check each object and verify if it should be rehydrated, calling a go routine to handle rehydration per object
func (r *rehydrationReceiver) rehydrate(ctx context.Context, objects []*aws.ObjectInfo) {
	// keep track of object names for number processed and deleting
	processedObjectsCount := &atomic.Int64{}
	for _, object := range objects {
		// check context
		select {
		case <-ctx.Done():
			r.logger.Info("Context finished, exiting rehydrate func")
		default:
		}

		// start processing object
		r.logger.Debug("Processing object", zap.String("name", object.Name))
		objectTime, telemetryType, err := rehydration.ParseEntityPath(object.Name)

		switch {
		case errors.Is(err, rehydration.ErrInvalidEntityPath):
			r.logger.Debug("Skipping object - non-matching entity path", zap.String("object", object.Name))
		case err != nil:
			r.logger.Error("Error processing object path", zap.String("object", object.Name), zap.Error(err))
		case r.checkpoint.ShouldParse(*objectTime, object.Name):
			// if object is not in specified time range or not the telemetry type supported by this receiver then skip consuming
			if !rehydration.IsInTimeRange(*objectTime, r.startingTime, r.endingTime) || telemetryType != r.supportedTelemetry {
				r.logger.Debug("Skipping object - not in configured time range or not supported telemetry type", zap.String("object", object.Name))
				continue
			}
			r.wg.Add(1)
			go r.rehydrateObject(ctx, object, objectTime, processedObjectsCount)
		default:
			r.logger.Debug("Skipping object - already parsed or timestamp is too old", zap.String("object", object.Name))
		}
	}

	r.wg.Wait()
	r.logger.Info("Successfully rehydrated objects", zap.Int64("number of objects processed", processedObjectsCount.Load()))
}

// rehydrateObject asynchronously handles rehydrating a given object & deleting it if configured
func (r *rehydrationReceiver) rehydrateObject(ctx context.Context, object *aws.ObjectInfo, objectTime *time.Time, count *atomic.Int64) {
	defer r.wg.Done()

	// check context
	select {
	case <-ctx.Done():
		r.logger.Info("Context finished, exiting rehydrateObject func")
	default:
	}

	// process and consume the object at the given path
	if err := r.processObject(ctx, object); err != nil {
		r.logger.Error("Error processing object", zap.String("object", object.Name), zap.Error(err))
		return
	}
	count.Add(1)
	r.saveCheckpointData(objectTime, object.Name)
	// delete object if configured
	if r.cfg.DeleteOnRead {
		if err := r.awsClient.DeleteObject(ctx, r.cfg.S3Bucket, object.Name); err != nil {
			r.logger.Error("Error while deleting object", zap.String("object", object.Name), zap.Error(err))
		}
		r.logger.Info("Successfully deleted rehydrated object", zap.String("object", object.Name))
	}
}

// processObject does the following:
// 1. Downloads the object
// 2. Decompresses the object if applicable
// 3. Pass the object to the consumer
func (r *rehydrationReceiver) processObject(ctx context.Context, object *aws.ObjectInfo) error {
	objectBuffer := make([]byte, object.Size)

	size, err := r.awsClient.DownloadObject(ctx, r.cfg.S3Bucket, object.Name, objectBuffer)
	if err != nil {
		return fmt.Errorf("download object: %w", err)
	}

	ext := filepath.Ext(object.Name)
	switch {
	case ext == ".gz":
		objectBuffer, err = rehydration.GzipDecompress(objectBuffer[:size])
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
	case ext == ".json":
		// Do nothing for json files
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if err := r.consumer.Consume(ctx, objectBuffer); err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	return nil
}

func (r *rehydrationReceiver) saveCheckpointData(objectTime *time.Time, objectName string) {
	r.checkpointMutex.Lock()
	defer r.checkpointMutex.Unlock()

	if r.lastObjectTime == nil || r.lastObjectTime.Before(*objectTime) {
		r.lastObjectName = objectName
		r.lastObjectTime = objectTime
	}
}

func (r *rehydrationReceiver) handleCheckpoint(ctx context.Context) error {
	r.checkpointMutex.Lock()
	defer r.checkpointMutex.Unlock()

	if r.lastObjectName == "" || r.lastObjectTime == nil {
		return nil
	}

	// update && store checkpoint
	r.checkpoint.UpdateCheckpoint(*r.lastObjectTime, r.lastObjectName)
	if err := r.checkpointStore.SaveCheckpoint(ctx, r.checkpointKey, r.checkpoint); err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}
	return nil
}
