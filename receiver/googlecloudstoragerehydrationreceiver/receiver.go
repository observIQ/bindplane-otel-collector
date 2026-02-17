// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package googlecloudstoragerehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver"

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/blobconsume"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
)

// newStorageClient is the function used to create new Google Cloud Storage Clients.
// Meant to be overwritten for tests
var newStorageClient = NewStorageClient

type rehydrationReceiver struct {
	logger             *zap.Logger
	id                 component.ID
	cfg                *Config
	storageClient      StorageClient
	supportedTelemetry pipeline.Signal
	consumer           blobconsume.Consumer
	checkpoint         *blobconsume.CheckPoint
	checkpointStore    storageclient.StorageClient

	objectChan chan []*ObjectInfo
	errChan    chan error
	doneChan   chan struct{}

	// mutexes for ensuring a thread safe checkpoint
	mut *sync.Mutex
	wg  *sync.WaitGroup

	lastObject     *ObjectInfo
	lastObjectTime *time.Time

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
	r.consumer = blobconsume.NewMetricsConsumer(nextConsumer)

	return r, nil
}

// newLogsReceiver creates a new logs specific receiver.
func newLogsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Logs) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalLogs
	r.consumer = blobconsume.NewLogsConsumer(nextConsumer)

	return r, nil
}

// newTracesReceiver creates a new traces specific receiver.
func newTracesReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Traces) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalTraces
	r.consumer = blobconsume.NewTracesConsumer(nextConsumer)

	return r, nil
}

// newRehydrationReceiver creates a new rehydration receiver
func newRehydrationReceiver(id component.ID, logger *zap.Logger, cfg *Config) (*rehydrationReceiver, error) {
	storageClient, err := newStorageClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	// We should not get an error for either of these time parsings as we check in config validate.
	// Doing error checking anyways just in case.
	startingTime, err := time.Parse(blobconsume.TimeFormat, cfg.StartingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid starting_time timestamp: %w", err)
	}

	endingTime, err := time.Parse(blobconsume.TimeFormat, cfg.EndingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid ending_time timestamp: %w", err)
	}

	return &rehydrationReceiver{
		logger:          logger,
		id:              id,
		cfg:             cfg,
		storageClient:   storageClient,
		checkpointStore: storageclient.NewNopStorage(),
		startingTime:    startingTime,
		endingTime:      endingTime,
		objectChan:      make(chan []*ObjectInfo),
		errChan:         make(chan error),
		doneChan:        make(chan struct{}),
		mut:             &sync.Mutex{},
		wg:              &sync.WaitGroup{},
	}, nil
}

// Start starts the receiver
func (r *rehydrationReceiver) Start(ctx context.Context, host component.Host) error {
	if r.cfg.StorageID != nil {
		checkpointStore, err := storageclient.NewStorageClient(ctx, host, *r.cfg.StorageID, r.id, r.supportedTelemetry)
		if err != nil {
			return fmt.Errorf("NewCheckpointStorage: %w", err)
		}
		r.checkpointStore = checkpointStore
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel

	go r.streamRehydrateObjects(cancelCtx)
	return nil
}

// Shutdown stops the receiver
func (r *rehydrationReceiver) Shutdown(ctx context.Context) error {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// wait for any in-progress operations to finish
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
		return fmt.Errorf("shutdown timeout: %w", shutdownCtx.Err())
	}

	var errs error
	if err := r.makeCheckpoint(shutdownCtx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error while saving checkpoint: %w", err))
	}

	r.mut.Lock()
	defer r.mut.Unlock()

	if err := r.checkpointStore.Close(shutdownCtx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error while closing checkpoint store: %w", err))
	}

	if err := r.storageClient.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error while closing storage client: %w", err))
	}

	r.logger.Info("Shutdown complete")
	return errs
}

// streamRehydrateObjects streams objects from the storage service
func (r *rehydrationReceiver) streamRehydrateObjects(ctx context.Context) {
	checkpoint := blobconsume.NewCheckpoint()
	err := r.checkpointStore.LoadStorageData(ctx, r.checkpointKey(), checkpoint)
	if err != nil {
		r.logger.Warn("Error loading checkpoint, continuing without a previous checkpoint", zap.Error(err))
		checkpoint = blobconsume.NewCheckpoint()
	}
	r.checkpoint = checkpoint

	startTime := time.Now()
	r.logger.Info("Starting rehydration", zap.Time("startTime", startTime))

	go r.storageClient.StreamObjects(ctx, r.errChan, r.objectChan, r.doneChan)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context cancelled, stopping rehydration", zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
			return
		case <-r.doneChan:
			r.logger.Info("Finished rehydrating objects", zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
			return
		case err := <-r.errChan:
			r.logger.Error("Error streaming objects, stopping rehydration", zap.Error(err), zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
			return
		case batch, ok := <-r.objectChan:
			if !ok {
				r.logger.Info("Finished rehydrating objects", zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
				return
			}
			numProcessedObjects := r.rehydrateObjects(ctx, batch)
			r.logger.Debug("Processed a number of objects", zap.Int("num_processed_objects", numProcessedObjects))
		}
	}
}

// rehydrateObjects processes a batch of objects
func (r *rehydrationReceiver) rehydrateObjects(ctx context.Context, objects []*ObjectInfo) (numProcessedObjects int) {
	// Go through each object and parse its path to determine if we should consume it or not
	r.logger.Debug("Received a batch of objects, parsing through them to determine if they should be rehydrated", zap.Int("num_objects", len(objects)))
	processedObjectCount := atomic.Int64{}
objectLoop:
	for _, object := range objects {
		select {
		case <-ctx.Done():
			break objectLoop
		default:
		}

		objectTime, telemetryType, err := blobconsume.ParseEntityPath(object.Name)
		switch {
		case errors.Is(err, blobconsume.ErrInvalidEntityPath):
			r.logger.Debug("Skipping Object, non-matching object path", zap.String("object", object.Name))
		case err != nil:
			r.logger.Error("Error processing object path", zap.String("object", object.Name), zap.Error(err))
		case r.checkpoint.ShouldParse(*objectTime, object.Name):
			// if the object is not in the specified time range or not of the telemetry type supported by this receiver
			// then skip consuming it.
			if !blobconsume.IsInTimeRange(*objectTime, r.startingTime, r.endingTime) || telemetryType != r.supportedTelemetry {
				continue
			}

			r.wg.Add(1)
			go func() {
				defer r.wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
				}
				// Process and consume the object at the given path
				if err := r.processObject(ctx, object); err != nil {
					// If the error is because the context was canceled, then we don't want to log it
					if !errors.Is(err, context.Canceled) {
						r.logger.Error("Error consuming object", zap.String("object", object.Name), zap.Error(err))
					}
					return
				}
				processedObjectCount.Add(1)

				// Delete object if configured to do so
				if err := r.conditionallyDeleteObject(ctx, object); err != nil {
					r.logger.Error("Error while attempting to delete object", zap.String("object", object.Name), zap.Error(err))
				}

				if r.lastObjectTime == nil || r.lastObjectTime.Before(*objectTime) {
					r.mut.Lock()
					r.lastObject = object
					r.lastObjectTime = objectTime
					r.mut.Unlock()
				}
			}()
		}
	}

	r.wg.Wait()

	if err := r.makeCheckpoint(ctx); err != nil {
		r.logger.Error("Error while saving checkpoint", zap.Error(err))
	}

	return int(processedObjectCount.Load())
}

// processObject processes a single object
func (r *rehydrationReceiver) processObject(ctx context.Context, object *ObjectInfo) error {
	// Create a buffer for the object data
	buf := make([]byte, object.Size)

	// Download the object into the buffer
	bytesRead, err := r.storageClient.DownloadObject(ctx, object.Name, buf)
	if err != nil {
		return fmt.Errorf("download object: %w", err)
	}

	if bytesRead != object.Size {
		return fmt.Errorf("expected to read %d bytes but read %d", object.Size, bytesRead)
	}

	// Check file extension see if we need to decompress
	ext := filepath.Ext(object.Name)
	switch {
	case ext == ".gz":
		buf, err = blobconsume.GzipDecompress(buf[:bytesRead])
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
	case ext == ".json":
		// Do nothing for json files
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if err := r.consumer.Consume(ctx, buf); err != nil {
		return fmt.Errorf("consume: %w", err)
	}
	return nil
}

// checkpointStorageKey the key used for storing the checkpoint
const checkpointStorageKey = "gcs_checkpoint"

// checkpointKey returns the key used for storing the checkpoint
func (r *rehydrationReceiver) checkpointKey() string {
	return fmt.Sprintf("%s_%s_%s", checkpointStorageKey, r.id, r.supportedTelemetry.String())
}

func (r *rehydrationReceiver) makeCheckpoint(ctx context.Context) error {
	if r.lastObject == nil || r.lastObjectTime == nil {
		return nil
	}
	r.logger.Debug("Making checkpoint", zap.String("object", r.lastObject.Name), zap.Time("time", *r.lastObjectTime))
	r.mut.Lock()
	defer r.mut.Unlock()
	r.checkpoint.UpdateCheckpoint(*r.lastObjectTime, r.lastObject.Name)
	return r.checkpointStore.SaveStorageData(ctx, r.checkpointKey(), r.checkpoint)
}

// conditionallyDeleteObject deletes the object if DeleteOnRead is enabled
func (r *rehydrationReceiver) conditionallyDeleteObject(ctx context.Context, object *ObjectInfo) error {
	if !r.cfg.DeleteOnRead {
		return nil
	}
	return r.storageClient.DeleteObject(ctx, object.Name)
}
