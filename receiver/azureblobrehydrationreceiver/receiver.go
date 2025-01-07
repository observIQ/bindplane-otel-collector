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
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
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
	checkpointStore    rehydration.CheckpointStorer

	lastBlob     *azureblob.BlobInfo
	lastBlobTime *time.Time

	startingTime time.Time
	endingTime   time.Time

	started    bool
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
		checkpointStore: rehydration.NewNopStorage(),
		startingTime:    startingTime,
		endingTime:      endingTime,
	}, nil
}

// Start starts the rehydration receiver
func (r *rehydrationReceiver) Start(ctx context.Context, host component.Host) error {
	if r.cfg.StorageID != nil {
		checkpointStore, err := rehydration.NewCheckpointStorage(ctx, host, *r.cfg.StorageID, r.id, r.supportedTelemetry)
		if err != nil {
			return fmt.Errorf("NewCheckpointStorage: %w", err)
		}
		r.checkpointStore = checkpointStore
	}

	r.started = true
	go r.streamRehydrateBlobs(ctx)
	return nil
}

// Shutdown shuts down the rehydration receiver
func (r *rehydrationReceiver) Shutdown(ctx context.Context) error {
	var err error
	// If we have called started then close and wait for goroutine to finish
	if r.cancelFunc != nil {
		r.cancelFunc()
	}
	r.makeCheckpoint(ctx)
	err = errors.Join(err, r.checkpointStore.Close(ctx))
	return err
}

// emptyPollLimit is the number of consecutive empty polling cycles that can
// occur before we stop polling.
const emptyPollLimit = 3

func (r *rehydrationReceiver) streamRehydrateBlobs(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	var marker *string

	checkpoint, err := r.checkpointStore.LoadCheckPoint(ctx, r.checkpointKey())
	if err != nil {
		r.logger.Warn("Error loading checkpoint, continuing without a previous checkpoint", zap.Error(err))
		checkpoint = rehydration.NewCheckpoint()
	}
	r.checkpoint = checkpoint

	var prefix *string
	if r.cfg.RootFolder != "" {
		prefix = &r.cfg.RootFolder
	}

	blobChan := make(chan *azureblob.BlobResults)
	errChan := make(chan error)

	cancelCtx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel
	doneChan := make(chan struct{})

	go r.azureClient.StreamBlobs(cancelCtx, r.cfg.Container, prefix, marker, errChan, blobChan, doneChan)

	for {
		select {
		case <-ctx.Done():
			return
		case <-doneChan:
			return
		case err := <-errChan:
			r.logger.Error("Error streaming blobs", zap.Error(err))
			return
		case br := <-blobChan:
			r.logger.Debug("Received blobs from stream", zap.Int("number_of_blobs", len(br.Blobs)))
			r.rehydrateBlobs(ctx, br.Blobs)
		}
	}

}

func (r *rehydrationReceiver) rehydrateBlobs(ctx context.Context, blobs []*azureblob.BlobInfo) {
	// Go through each blob and parse it's path to determine if we should consume it or not
	numProcessedBlobs := 0
	for _, blob := range blobs {
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

			r.lastBlob = blob
			r.lastBlobTime = blobTime

			// Process and consume the blob at the given path
			if err := r.processBlob(ctx, blob); err != nil {
				r.logger.Error("Error consuming blob", zap.String("blob", blob.Name), zap.Error(err))
				continue
			}

			numProcessedBlobs++
			r.logger.Debug("Processed blob", zap.String("blob", blob.Name), zap.Int("num_processed_blobs", numProcessedBlobs))

			// Delete blob if configured to do so
			if r.cfg.DeleteOnRead {
				if err := r.azureClient.DeleteBlob(ctx, r.cfg.Container, blob.Name); err != nil {
					r.logger.Error("Error while attempting to delete blob", zap.String("blob", blob.Name), zap.Error(err))
				}
			}
		}
	}

	if err := r.makeCheckpoint(ctx); err != nil {
		r.logger.Error("Error while saving checkpoint", zap.Error(err))
	}
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
	r.checkpoint.UpdateCheckpoint(*r.lastBlobTime, r.lastBlob.Name)
	r.checkpointStore.SaveCheckpoint(ctx, r.checkpointKey(), r.checkpoint)
	return nil
}
