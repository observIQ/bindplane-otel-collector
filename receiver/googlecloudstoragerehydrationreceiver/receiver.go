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

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver/internal"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
)

// newStorageClient is the function used to create new Google Cloud Storage Clients.
// Meant to be overwritten for tests
var newStorageClient = internal.NewStorageClient

type rehydrationReceiver struct {
	logger             *zap.Logger
	id                 component.ID
	cfg                *Config
	storageClient      internal.StorageClient
	supportedTelemetry pipeline.Signal
	consumer           rehydration.Consumer
	checkpoint         *rehydration.CheckPoint
	checkpointStore    rehydration.CheckpointStorer

	blobChan chan []*internal.BlobInfo
	errChan  chan error
	doneChan chan struct{}

	// mutexes for ensuring a thread safe checkpoint
	mut *sync.Mutex
	wg  *sync.WaitGroup

	lastBlob     *internal.BlobInfo
	lastBlobTime *time.Time

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
		blobChan:      make(chan []*internal.BlobInfo, cfg.BatchSize),
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
	storage, err := rehydration.GetStorageExtension(ctx, host, r.cfg.StorageID)
	if err != nil {
		return fmt.Errorf("failed to get storage extension: %w", err)
	}

	// Create checkpoint store
	r.checkpointStore = rehydration.NewCheckpointStore(storage, r.id.String())

	// Load checkpoint if it exists
	checkpoint, err := r.checkpointStore.LoadCheckpoint(ctx)
	if err != nil {
		if !errors.Is(err, rehydration.ErrNoCheckpoint) {
			return fmt.Errorf("failed to load checkpoint: %w", err)
		}
		checkpoint = &rehydration.CheckPoint{
			LastProcessedTime: r.startingTime,
		}
	}

	r.checkpoint = checkpoint
	r.lastBlobTime = &r.startingTime

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
	close(r.blobChan)
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

	// Get all blobs within the time range
	blobs, err := r.storageClient.ListBlobs(ctx, r.startingTime, r.endingTime)
	if err != nil {
		r.errChan <- fmt.Errorf("failed to list blobs: %w", err)
		return
	}

	// Process blobs in batches
	for i := 0; i < len(blobs); i += r.cfg.BatchSize {
		end := i + r.cfg.BatchSize
		if end > len(blobs) {
			end = len(blobs)
		}

		batch := blobs[i:end]
		r.blobChan <- batch

		// Wait for the batch to be processed
		select {
		case <-ctx.Done():
			return
		case <-r.doneChan:
			return
		}
	}

	// Close the blob channel when done
	close(r.blobChan)
}

// rehydrateBlobs processes a batch of blobs
func (r *rehydrationReceiver) rehydrateBlobs(ctx context.Context, blobs []*internal.BlobInfo) (numProcessedBlobs int) {
	for _, blob := range blobs {
		if err := r.processBlob(ctx, blob); err != nil {
			r.logger.Error("Failed to process blob", zap.String("name", blob.Name), zap.Error(err))
			continue
		}
		numProcessedBlobs++
	}
	return numProcessedBlobs
}

// processBlob processes a single blob
func (r *rehydrationReceiver) processBlob(ctx context.Context, blob *internal.BlobInfo) error {
	// Download the blob
	data, err := r.storageClient.DownloadBlob(ctx, blob.Name)
	if err != nil {
		return fmt.Errorf("failed to download blob: %w", err)
	}

	// Process the data
	if err := r.consumer.Consume(ctx, data); err != nil {
		return fmt.Errorf("failed to consume data: %w", err)
	}

	// Delete the blob if configured
	if r.cfg.DeleteOnRead {
		if err := r.storageClient.DeleteBlob(ctx, blob.Name); err != nil {
			return fmt.Errorf("failed to delete blob: %w", err)
		}
	}

	// Update checkpoint
	r.mut.Lock()
	r.lastBlob = blob
	r.lastBlobTime = &blob.LastModified
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

	if r.lastBlobTime == nil {
		return nil
	}

	checkpoint := &rehydration.CheckPoint{
		LastProcessedTime: *r.lastBlobTime,
	}

	if err := r.checkpointStore.StoreCheckpoint(ctx, checkpoint); err != nil {
		return fmt.Errorf("failed to store checkpoint: %w", err)
	}

	return nil
} 