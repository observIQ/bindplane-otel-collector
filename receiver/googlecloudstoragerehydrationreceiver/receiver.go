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
	"sync"
	"time"

	"sync/atomic"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/xextension/storage"
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
	consumer           rehydration.Consumer
	checkpoint         *rehydration.CheckPoint
	checkpointStore    rehydration.CheckpointStorer

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
	startingTime, err := time.Parse(time.RFC3339, cfg.StartingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid starting_time: %w", err)
	}

	endingTime, err := time.Parse(time.RFC3339, cfg.EndingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid ending_time: %w", err)
	}

	storageClient, err := newStorageClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &rehydrationReceiver{
		logger:        logger,
		id:            id,
		cfg:           cfg,
		storageClient: storageClient,
		objectChan:    make(chan []*ObjectInfo, cfg.BatchSize),
		errChan:       make(chan error, 1),
		doneChan:      make(chan struct{}),
		mut:           &sync.Mutex{},
		wg:            &sync.WaitGroup{},
		startingTime:  startingTime,
		endingTime:    endingTime,
	}, nil
}

// Start starts the receiver
func (r *rehydrationReceiver) Start(ctx context.Context, host component.Host) error {
	ctx, r.cancelFunc = context.WithCancel(ctx)

	// Get the storage extension
	extension, ok := host.GetExtensions()[*r.cfg.StorageID]
	if !ok {
		return fmt.Errorf("storage extension '%s' not found", r.cfg.StorageID)
	}

	_, ok = extension.(storage.Extension)
	if !ok {
		return fmt.Errorf("non-storage extension '%s' found", r.cfg.StorageID)
	}

	// Create checkpoint store
	checkpointStore, err := rehydration.NewCheckpointStorage(ctx, host, *r.cfg.StorageID, r.id, r.supportedTelemetry)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint storage: %w", err)
	}
	r.checkpointStore = checkpointStore

	// Load checkpoint if it exists
	checkpoint, err := r.checkpointStore.LoadCheckPoint(ctx, r.checkpointKey())
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			return fmt.Errorf("failed to load checkpoint: %w", err)
		}
		checkpoint = rehydration.NewCheckpoint()
	}

	r.checkpoint = checkpoint
	r.lastObjectTime = &r.startingTime

	// Start the rehydration process
	go r.streamRehydrateBlobs(ctx)

	return nil
}

// Shutdown stops the receiver
func (r *rehydrationReceiver) Shutdown(ctx context.Context) error {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	// Wait for all goroutines to finish
	r.wg.Wait()

	// Close channels
	close(r.doneChan)
	close(r.objectChan)
	close(r.errChan)

	// Check for any errors
	select {
	case err := <-r.errChan:
		return fmt.Errorf("error during shutdown: %w", err)
	default:
		return nil
	}
}

// streamRehydrateBlobs streams blobs from the storage service
func (r *rehydrationReceiver) streamRehydrateBlobs(ctx context.Context) {
	r.wg.Add(1)
	defer r.wg.Done()

	// Start streaming blobs
	go r.storageClient.StreamObjects(ctx, r.errChan, r.objectChan, r.doneChan)

	// Process blobs as they arrive
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.doneChan:
			return
		case batch := <-r.objectChan:
			r.rehydrateObjects(ctx, batch)
		case err := <-r.errChan:
			r.logger.Error("Error streaming objects", zap.Error(err))
			return
		}
	}
}

// rehydrateObjects processes a batch of objects
func (r *rehydrationReceiver) rehydrateObjects(ctx context.Context, objects []*ObjectInfo) (numProcessedObjects int) {
	// Go through each object and parse its path to determine if we should consume it or not
	r.logger.Debug("Received a batch of objects, parsing through them to determine if they should be rehydrated", zap.Int("num_objects", len(objects)))
	processedObjectCount := atomic.Int64{}
	for _, object := range objects {
		select {
		case <-ctx.Done():
			break
		default:
		}

		objectTime, telemetryType, err := rehydration.ParseEntityPath(object.Name)
		switch {
		case errors.Is(err, rehydration.ErrInvalidEntityPath):
			r.logger.Debug("Skipping Object, non-matching object path", zap.String("object", object.Name))
		case err != nil:
			r.logger.Error("Error processing object path", zap.String("object", object.Name), zap.Error(err))
		case r.checkpoint.ShouldParse(*objectTime, object.Name):
			// if the object is not in the specified time range or not of the telemetry type supported by this receiver
			// then skip consuming it.
			if !rehydration.IsInTimeRange(*objectTime, r.startingTime, r.endingTime) || telemetryType != r.supportedTelemetry {
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

// conditionallyDeleteObject deletes the object if DeleteOnRead is enabled
func (r *rehydrationReceiver) conditionallyDeleteObject(ctx context.Context, object *ObjectInfo) error {
	if !r.cfg.DeleteOnRead {
		return nil
	}
	return r.storageClient.DeleteObject(ctx, object.Name)
}

// processObject processes a single object
func (r *rehydrationReceiver) processObject(ctx context.Context, object *ObjectInfo) error {
	// Create a buffer for the object data
	buf := make([]byte, object.Size)

	// Download the object into the buffer
	bytesRead, err := r.storageClient.DownloadObject(ctx, object.Name, buf)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	if bytesRead != object.Size {
		return fmt.Errorf("expected to read %d bytes but read %d", object.Size, bytesRead)
	}

	// Process the data
	if err := r.consumer.Consume(ctx, buf); err != nil {
		return fmt.Errorf("failed to consume data: %w", err)
	}

	// Delete the blob if configured
	if r.cfg.DeleteOnRead {
		if err := r.storageClient.DeleteObject(ctx, object.Name); err != nil {
			return fmt.Errorf("failed to delete object: %w", err)
		}
	}

	// Update checkpoint
	r.mut.Lock()
	r.lastObject = object
	now := time.Now()
	r.lastObjectTime = &now
	r.mut.Unlock()

	return nil
}

// checkpointKey returns the key for the checkpoint
func (r *rehydrationReceiver) checkpointKey() string {
	return fmt.Sprintf("%s-%s", r.id.String(), r.supportedTelemetry.String())
}

// makeCheckpoint creates a checkpoint
func (r *rehydrationReceiver) makeCheckpoint(ctx context.Context) error {
	r.mut.Lock()
	defer r.mut.Unlock()

	if r.lastObjectTime == nil {
		return nil
	}

	checkpoint := rehydration.NewCheckpoint()
	checkpoint.UpdateCheckpoint(*r.lastObjectTime, r.lastObject.Name)

	if err := r.checkpointStore.SaveCheckpoint(ctx, r.checkpointKey(), checkpoint); err != nil {
		return fmt.Errorf("failed to store checkpoint: %w", err)
	}

	return nil
} 