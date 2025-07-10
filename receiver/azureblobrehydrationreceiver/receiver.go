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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver"

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver/internal/azureblob"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
)

// newAzureBlobClient is the function use to create new Azure Blob Clients.
// Meant to be overwritten for tests
var newAzureBlobClient = azureblob.NewAzureBlobClient

type rehydrationReceiver struct {
	logger             *zap.Logger
	id                 component.ID
	cfg                *Config
	azureClient        azureblob.BlobClient
	supportedTelemetry pipeline.Signal
	consumer           rehydration.Consumer
	checkpoint         *rehydration.CheckPoint
	checkpointStore    storageclient.StorageClient

	blobChan chan []*azureblob.BlobInfo
	errChan  chan error
	doneChan chan struct{}

	// mutexes for ensuring a thread safe checkpoint
	mut *sync.Mutex
	wg  *sync.WaitGroup

	lastBlob     *azureblob.BlobInfo
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
	azureClient, err := newAzureBlobClient(cfg.ConnectionString, cfg.BatchSize, cfg.PageSize)
	if err != nil {
		return nil, fmt.Errorf("new Azure client: %w", err)
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
		azureClient:     azureClient,
		checkpointStore: storageclient.NewNopStorage(),
		startingTime:    startingTime,
		endingTime:      endingTime,
		blobChan:        make(chan []*azureblob.BlobInfo),
		errChan:         make(chan error),
		doneChan:        make(chan struct{}),
		mut:             &sync.Mutex{},
		wg:              &sync.WaitGroup{},
	}, nil
}

// Start starts the rehydration receiver
func (r *rehydrationReceiver) Start(ctx context.Context, host component.Host) error {
	r.logAnyDeprecationWarnings()

	if r.cfg.StorageID != nil {
		checkpointStore, err := storageclient.NewStorageClient(ctx, host, *r.cfg.StorageID, r.id, r.supportedTelemetry)
		if err != nil {
			return fmt.Errorf("NewCheckpointStorage: %w", err)
		}
		r.checkpointStore = checkpointStore
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel

	go r.streamRehydrateBlobs(cancelCtx)
	return nil
}

func (r *rehydrationReceiver) logAnyDeprecationWarnings() {
	if r.cfg.PollInterval != 0 {
		r.logger.Warn("poll_interval is no longer recognized and will be removed in a future release. batch_size/page_size should be used instead")
	}

	if r.cfg.PollTimeout != 0 {
		r.logger.Warn("poll_timeout is no longer recognized and will be removed in a future release. batch_size/page_size should be used instead")
	}
}

// Shutdown shuts down the rehydration receiver
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
	r.logger.Info("Shutdown complete")
	return errs
}

func (r *rehydrationReceiver) streamRehydrateBlobs(ctx context.Context) {
	checkpoint := rehydration.NewCheckpoint()
	err := r.checkpointStore.LoadStorageData(ctx, r.checkpointKey(), checkpoint)
	if err != nil {
		r.logger.Warn("Error loading checkpoint, continuing without a previous checkpoint", zap.Error(err))
		checkpoint = rehydration.NewCheckpoint()
	}
	r.checkpoint = checkpoint

	var prefix *string
	if r.cfg.RootFolder != "" {
		prefix = &r.cfg.RootFolder
	}

	startTime := time.Now()
	r.logger.Info("Starting rehydration", zap.Time("startTime", startTime))

	go r.azureClient.StreamBlobs(ctx, r.cfg.Container, prefix, r.errChan, r.blobChan, r.doneChan)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context cancelled, stopping rehydration", zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
			return
		case <-r.doneChan:
			r.logger.Info("Finished rehydrating blobs", zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
			return
		case err := <-r.errChan:
			r.logger.Error("Error streaming blobs, stopping rehydration", zap.Error(err), zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
			return
		case br, ok := <-r.blobChan:
			if !ok {
				r.logger.Info("Finished rehydrating blobs", zap.Int("durationSeconds", int(time.Since(startTime).Seconds())))
				return
			}
			numProcessedBlobs := r.rehydrateBlobs(ctx, br)
			r.logger.Debug("Processed a number of blobs", zap.Int("num_processed_blobs", numProcessedBlobs))
		}
	}
}

func (r *rehydrationReceiver) rehydrateBlobs(ctx context.Context, blobs []*azureblob.BlobInfo) (numProcessedBlobs int) {
	// Go through each blob and parse it's path to determine if we should consume it or not
	r.logger.Debug("Received a batch of blobs, parsing through them to determine if they should be rehydrated", zap.Int("num_blobs", len(blobs)))
	processedBlobCount := atomic.Int64{}
	for _, blob := range blobs {
		select {
		case <-ctx.Done():
			break
		default:
		}

		blobTime, telemetryType, err := rehydration.ParseEntityPath(blob.Name)
		switch {
		case errors.Is(err, rehydration.ErrInvalidEntityPath):
			r.logger.Debug("Skipping Blob, non-matching blob path", zap.String("blob", blob.Name))
		case err != nil:
			r.logger.Error("Error processing blob path", zap.String("blob", blob.Name), zap.Error(err))
		case r.checkpoint.ShouldParse(*blobTime, blob.Name):
			// if the blob is not in the specified time range or not of the telemetry type supported by this receiver
			// then skip consuming it.
			if !rehydration.IsInTimeRange(*blobTime, r.startingTime, r.endingTime) || telemetryType != r.supportedTelemetry {
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
				// Process and consume the blob at the given path
				if err := r.processBlob(ctx, blob); err != nil {
					// If the error is because the context was canceled, then we don't want to log it
					if !errors.Is(err, context.Canceled) {
						r.logger.Error("Error consuming blob", zap.String("blob", blob.Name), zap.Error(err))
					}
					return
				}
				processedBlobCount.Add(1)

				// Delete blob if configured to do so
				if err := r.conditionallyDeleteBlob(ctx, blob); err != nil {
					r.logger.Error("Error while attempting to delete blob", zap.String("blob", blob.Name), zap.Error(err))
				}

				if r.lastBlobTime == nil || r.lastBlobTime.Before(*blobTime) {
					r.mut.Lock()
					r.lastBlob = blob
					r.lastBlobTime = blobTime
					r.mut.Unlock()
				}
			}()
		}
	}

	r.wg.Wait()

	if err := r.makeCheckpoint(ctx); err != nil {
		r.logger.Error("Error while saving checkpoint", zap.Error(err))
	}

	return int(processedBlobCount.Load())
}

// processBlob does the following:
// 1. Downloads the blob
// 2. Decompresses the blob if applicable
// 3. Pass the blob to the consumer
func (r *rehydrationReceiver) processBlob(ctx context.Context, blob *azureblob.BlobInfo) error {
	// Allocate a buffer the size of the blob. If the buffer isn't big enough download errors.
	blobBuffer := make([]byte, blob.Size)

	size, err := r.azureClient.DownloadBlob(ctx, r.cfg.Container, blob.Name, blobBuffer)
	if err != nil {
		return fmt.Errorf("download blob: %w", err)
	}

	// Check file extension see if we need to decompress
	ext := filepath.Ext(blob.Name)
	switch {
	case ext == ".gz":
		blobBuffer, err = rehydration.GzipDecompress(blobBuffer[:size])
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
	case ext == ".json":
		// Do nothing for json files
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if err := r.consumer.Consume(ctx, blobBuffer); err != nil {
		return fmt.Errorf("consume: %w", err)
	}
	return nil
}

// checkpointStorageKey the key used for storing the checkpoint
const checkpointStorageKey = "azure_blob_checkpoint"

// checkpointKey returns the key used for storing the checkpoint
func (r *rehydrationReceiver) checkpointKey() string {
	return fmt.Sprintf("%s_%s_%s", checkpointStorageKey, r.id, r.supportedTelemetry.String())
}

func (r *rehydrationReceiver) makeCheckpoint(ctx context.Context) error {
	if r.lastBlob == nil || r.lastBlobTime == nil {
		return nil
	}
	r.logger.Debug("Making checkpoint", zap.String("blob", r.lastBlob.Name), zap.Time("time", *r.lastBlobTime))
	r.mut.Lock()
	defer r.mut.Unlock()
	r.checkpoint.UpdateCheckpoint(*r.lastBlobTime, r.lastBlob.Name)
	return r.checkpointStore.SaveStorageData(ctx, r.checkpointKey(), r.checkpoint)
}

func (r *rehydrationReceiver) conditionallyDeleteBlob(ctx context.Context, blob *azureblob.BlobInfo) error {
	if !r.cfg.DeleteOnRead {
		return nil
	}
	return r.azureClient.DeleteBlob(ctx, r.cfg.Container, blob.Name)
}
