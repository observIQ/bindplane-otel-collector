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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver"

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/observiq/bindplane-agent/internal/rehydration"
	"github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver/internal/azureblob"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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
	supportedTelemetry component.DataType
	consumer           rehydration.Consumer
	checkpointStore    rehydration.CheckpointStorer

	startingTime time.Time
	endingTime   time.Time

	doneChan   chan struct{}
	started    bool
	ctx        context.Context
	cancelFunc context.CancelCauseFunc
}

// newMetricsReceiver creates a new metrics specific receiver.
func newMetricsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Metrics) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = component.DataTypeMetrics
	r.consumer = rehydration.NewMetricsConsumer(nextConsumer)

	return r, nil
}

// newLogsReceiver creates a new logs specific receiver.
func newLogsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Logs) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = component.DataTypeLogs
	r.consumer = rehydration.NewLogsConsumer(nextConsumer)

	return r, nil
}

// newTracesReceiver creates a new traces specific receiver.
func newTracesReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Traces) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = component.DataTypeTraces
	r.consumer = rehydration.NewTracesConsumer(nextConsumer)

	return r, nil
}

// newRehydrationReceiver creates a new rehydration receiver
func newRehydrationReceiver(id component.ID, logger *zap.Logger, cfg *Config) (*rehydrationReceiver, error) {
	azureClient, err := newAzureBlobClient(cfg.ConnectionString)
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

	ctx, cancel := context.WithCancelCause(context.Background())

	return &rehydrationReceiver{
		logger:          logger,
		id:              id,
		cfg:             cfg,
		azureClient:     azureClient,
		doneChan:        make(chan struct{}),
		checkpointStore: rehydration.NewNopStorage(),
		startingTime:    startingTime,
		endingTime:      endingTime,
		ctx:             ctx,
		cancelFunc:      cancel,
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
	go r.scrape()
	return nil
}

// Shutdown shuts down the rehydration receiver
func (r *rehydrationReceiver) Shutdown(ctx context.Context) error {
	r.cancelFunc(errors.New("shutdown"))
	var err error

	// If we have called started then close and wait for goroutine to finish
	if r.started {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case <-r.doneChan:
		}
	}

	err = errors.Join(err, r.checkpointStore.Close(ctx))

	return err
}

// emptyPollLimit is the number of consecutive empty polling cycles that can
// occur before we stop polling.
const emptyPollLimit = 3

// scrape scrapes the Azure api on interval
func (r *rehydrationReceiver) scrape() {
	defer close(r.doneChan)
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	var marker *string

	// load the previous checkpoint. If not exist should return zero value for time
	checkpoint, err := r.checkpointStore.LoadCheckPoint(r.ctx, r.checkpointKey())
	if err != nil {
		r.logger.Warn("Error loading checkpoint, continuing without a previous checkpoint", zap.Error(err))
		checkpoint = rehydration.NewCheckpoint()
	}

	// Call once before the loop to ensure we do a collection before the first ticker
	numBlobsRehydrated := r.rehydrateBlobs(checkpoint, marker)
	emptyBlobCounter := checkBlobCount(numBlobsRehydrated, 0)

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			// Polling for blobs has egress charges so we want to stop polling
			// after we stop finding blobs.
			if emptyBlobCounter == emptyPollLimit {
				return
			}

			numBlobsRehydrated := r.rehydrateBlobs(checkpoint, marker)
			emptyBlobCounter = checkBlobCount(numBlobsRehydrated, emptyBlobCounter)
		}
	}
}

// rehydrateBlobs pulls blob paths from the UI and if they are within the specified
// time range then the blobs will be downloaded and rehydrated.
// The passed in checkpoint and marker will be updated and should be used in the next iteration.
// The count of blobs processed will be returned
func (r *rehydrationReceiver) rehydrateBlobs(checkpoint *rehydration.CheckPoint, marker *string) (numBlobsRehydrated int) {
	var prefix *string
	if r.cfg.RootFolder != "" {
		prefix = &r.cfg.RootFolder
	}

	ctxTimeout, cancel := context.WithTimeout(r.ctx, r.cfg.PollTimeout)
	defer cancel()

	// get blobs from Azure
	blobs, nextMarker, err := r.azureClient.ListBlobs(ctxTimeout, r.cfg.Container, prefix, marker)
	if err != nil {
		r.logger.Error("Failed to list blobs", zap.Error(err))
		return
	}

	marker = nextMarker

	// Go through each blob and parse it's path to determine if we should consume it or not
	for _, blob := range blobs {
		blobTime, telemetryType, err := rehydration.ParseEntityPath(blob.Name)
		switch {
		case errors.Is(err, rehydration.ErrInvalidEntityPath):
			r.logger.Debug("Skipping Blob, non-matching blob path", zap.String("blob", blob.Name))
		case err != nil:
			r.logger.Error("Error processing blob path", zap.String("blob", blob.Name), zap.Error(err))
		case checkpoint.ShouldParse(*blobTime, blob.Name):
			// if the blob is not in the specified time range or not of the telemetry type supported by this receiver
			// then skip consuming it.
			if !rehydration.IsInTimeRange(*blobTime, r.startingTime, r.endingTime) || telemetryType != r.supportedTelemetry {
				continue
			}

			// Process and consume the blob at the given path
			if err := r.processBlob(blob); err != nil {
				r.logger.Error("Error consuming blob", zap.String("blob", blob.Name), zap.Error(err))
				continue
			}

			numBlobsRehydrated++

			// Update and save the checkpoint with the most recently processed blob
			checkpoint.UpdateCheckpoint(*blobTime, blob.Name)
			if err := r.checkpointStore.SaveCheckpoint(r.ctx, r.checkpointKey(), checkpoint); err != nil {
				r.logger.Error("Error while saving checkpoint", zap.Error(err))
			}

			// Delete blob if configured to do so
			if r.cfg.DeleteOnRead {
				if err := r.azureClient.DeleteBlob(r.ctx, r.cfg.Container, blob.Name); err != nil {
					r.logger.Error("Error while attempting to delete blob", zap.String("blob", blob.Name), zap.Error(err))
				}
			}
		}
	}

	return
}

// processBlob does the following:
// 1. Downloads the blob
// 2. Decompresses the blob if applicable
// 3. Pass the blob to the consumer
func (r *rehydrationReceiver) processBlob(blob *azureblob.BlobInfo) error {
	// Allocate a buffer the size of the blob. If the buffer isn't big enough download errors.
	blobBuffer := make([]byte, blob.Size)

	size, err := r.azureClient.DownloadBlob(r.ctx, r.cfg.Container, blob.Name, blobBuffer)
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

	if err := r.consumer.Consume(r.ctx, blobBuffer); err != nil {
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

// checkBlobCount checks the number of blobs rehydrated and the current state of the
// empty counter. If zero blobs were rehydrated increment the counter.
// If there were blobs rehydrated reset the counter as we want to track consecutive zero sized polls.
func checkBlobCount(numBlobsRehydrated, emptyBlobsCounter int) int {
	switch {
	case emptyBlobsCounter == emptyPollLimit: // If we are at the limit return the limit
		return emptyPollLimit
	case numBlobsRehydrated == 0: // If no blobs were rehydrated then increment the empty blobs counter
		return emptyBlobsCounter + 1
	default: // Default case is numBlobsRehydrated > 0 so reset emptyBlobsCounter to 0
		return 0
	}
}
