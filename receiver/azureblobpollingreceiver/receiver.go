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

package azureblobpollingreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver"

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/rehydration"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver/internal/azureblob"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
)

// newAzureBlobClient is the function use to create new Azure Blob Clients.
// Meant to be overwritten for tests
var newAzureBlobClient = azureblob.NewAzureBlobClient

type pollingReceiver struct {
	logger             *zap.Logger
	id                 component.ID
	cfg                *Config
	azureClient        azureblob.BlobClient
	supportedTelemetry pipeline.Signal
	consumer           rehydration.Consumer
	checkpoint         *PollingCheckPoint
	checkpointStore    storageclient.StorageClient

	pollInterval    time.Duration
	initialLookback time.Duration

	blobChan chan []*azureblob.BlobInfo
	errChan  chan error
	doneChan chan struct{}

	// mutexes for ensuring a thread safe checkpoint
	mut *sync.Mutex
	wg  *sync.WaitGroup

	lastBlob     *azureblob.BlobInfo
	lastBlobTime *time.Time

	filenameRegex *regexp.Regexp
	cancelFunc    context.CancelFunc
}

// newMetricsReceiver creates a new metrics specific receiver.
func newMetricsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Metrics) (*pollingReceiver, error) {
	r, err := newPollingReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalMetrics
	r.consumer = rehydration.NewMetricsConsumer(nextConsumer)

	return r, nil
}

// newLogsReceiver creates a new logs specific receiver.
func newLogsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Logs) (*pollingReceiver, error) {
	r, err := newPollingReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalLogs
	r.consumer = rehydration.NewLogsConsumer(nextConsumer)

	return r, nil
}

// newTracesReceiver creates a new traces specific receiver.
func newTracesReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Traces) (*pollingReceiver, error) {
	r, err := newPollingReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalTraces
	r.consumer = rehydration.NewTracesConsumer(nextConsumer)

	return r, nil
}

// newPollingReceiver creates a new polling receiver
func newPollingReceiver(id component.ID, logger *zap.Logger, cfg *Config) (*pollingReceiver, error) {
	azureClient, err := newAzureBlobClient(cfg.ConnectionString, cfg.BatchSize, cfg.PageSize)
	if err != nil {
		return nil, fmt.Errorf("new Azure client: %w", err)
	}

	// Set initialLookback to pollInterval if not specified
	initialLookback := cfg.InitialLookback
	if initialLookback == 0 {
		initialLookback = cfg.PollInterval
	}

	// Compile filename regex if provided
	var filenameRegex *regexp.Regexp
	if cfg.FilenamePattern != "" {
		filenameRegex, err = regexp.Compile(cfg.FilenamePattern)
		if err != nil {
			return nil, fmt.Errorf("compile filename pattern: %w", err)
		}
	}

	return &pollingReceiver{
		logger:          logger,
		id:              id,
		cfg:             cfg,
		azureClient:     azureClient,
		checkpointStore: storageclient.NewNopStorage(),
		pollInterval:    cfg.PollInterval,
		initialLookback: initialLookback,
		blobChan:        make(chan []*azureblob.BlobInfo),
		errChan:         make(chan error),
		doneChan:        make(chan struct{}),
		mut:             &sync.Mutex{},
		wg:              &sync.WaitGroup{},
		filenameRegex:   filenameRegex,
	}, nil
}

// Start starts the polling receiver
func (r *pollingReceiver) Start(ctx context.Context, host component.Host) error {
	if r.cfg.StorageID != nil {
		checkpointStore, err := storageclient.NewStorageClient(ctx, host, *r.cfg.StorageID, r.id, r.supportedTelemetry)
		if err != nil {
			return fmt.Errorf("NewCheckpointStorage: %w", err)
		}
		r.checkpointStore = checkpointStore
	}

	// Load checkpoint
	checkpoint := NewPollingCheckpoint()
	err := r.checkpointStore.LoadStorageData(ctx, r.checkpointKey(), checkpoint)
	if err != nil {
		r.logger.Warn("Error loading checkpoint, starting fresh", zap.Error(err))
		checkpoint = NewPollingCheckpoint()
	}
	r.checkpoint = checkpoint

	cancelCtx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel

	go r.pollLoop(cancelCtx)
	return nil
}

// Shutdown shuts down the polling receiver
func (r *pollingReceiver) Shutdown(ctx context.Context) error {
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

// getTelemetryType returns the telemetry type for the receiver
// It first checks if an explicit telemetry_type is configured,
// otherwise falls back to the pipeline type the receiver is configured in
func (r *pollingReceiver) getTelemetryType() pipeline.Signal {
	if r.cfg.TelemetryType != "" {
		switch r.cfg.TelemetryType {
		case "logs":
			return pipeline.SignalLogs
		case "metrics":
			return pipeline.SignalMetrics
		case "traces":
			return pipeline.SignalTraces
		}
	}
	return r.supportedTelemetry
}

// pollLoop continuously polls for new blobs at the configured interval
func (r *pollingReceiver) pollLoop(ctx context.Context) {
	r.logger.Info("Starting continuous polling", zap.Duration("poll_interval", r.pollInterval))

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	// Run first poll immediately
	r.runPoll(ctx)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context cancelled, stopping polling")
			return
		case <-ticker.C:
			r.runPoll(ctx)
		}
	}
}

// runPoll executes a single poll operation with dynamic time window
func (r *pollingReceiver) runPoll(ctx context.Context) {
	now := time.Now().UTC()
	
	// Calculate time window
	var startingTime, endingTime time.Time
	if r.checkpoint.LastPollTime.IsZero() {
		// First poll - use initial lookback
		startingTime = now.Add(-r.initialLookback)
		r.logger.Info("First poll, using initial lookback", 
			zap.Time("starting_time", startingTime), 
			zap.Time("ending_time", now),
			zap.Duration("lookback", r.initialLookback))
	} else {
		// Subsequent polls - use last poll time
		startingTime = r.checkpoint.LastPollTime
		r.logger.Debug("Polling with dynamic window", 
			zap.Time("starting_time", startingTime), 
			zap.Time("ending_time", now))
	}
	endingTime = now

	// Reset lastBlob tracking for this poll
	r.lastBlob = nil
	r.lastBlobTime = nil

	var prefix *string
	if r.cfg.RootFolder != "" {
		prefix = &r.cfg.RootFolder
	}

	pollStartTime := time.Now()
	r.logger.Info("Starting poll", zap.Time("poll_time", pollStartTime))

	// Stream blobs in a goroutine
	go r.azureClient.StreamBlobs(ctx, r.cfg.Container, prefix, r.errChan, r.blobChan, r.doneChan)

	totalProcessed := 0
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context cancelled during poll")
			return
		case <-r.doneChan:
			r.logger.Info("Poll completed", 
				zap.Int("total_processed", totalProcessed),
				zap.Int("duration_seconds", int(time.Since(pollStartTime).Seconds())))
			
			// Update checkpoint with poll time
			r.mut.Lock()
			r.checkpoint.UpdatePollTime(endingTime)
			r.mut.Unlock()
			
			if err := r.makeCheckpoint(ctx); err != nil {
				r.logger.Error("Error saving checkpoint after poll", zap.Error(err))
			}
			return
		case err := <-r.errChan:
			r.logger.Error("Error during poll", zap.Error(err))
			return
		case br, ok := <-r.blobChan:
			if !ok {
				r.logger.Info("Poll completed", 
					zap.Int("total_processed", totalProcessed),
					zap.Int("duration_seconds", int(time.Since(pollStartTime).Seconds())))
				
				// Update checkpoint with poll time
				r.mut.Lock()
				r.checkpoint.UpdatePollTime(endingTime)
				r.mut.Unlock()
				
				if err := r.makeCheckpoint(ctx); err != nil {
					r.logger.Error("Error saving checkpoint after poll", zap.Error(err))
				}
				return
			}
			numProcessed := r.processBlobs(ctx, br, startingTime, endingTime)
			totalProcessed += numProcessed
			r.logger.Debug("Processed batch of blobs", zap.Int("num_processed", numProcessed))
		}
	}
}

func (r *pollingReceiver) processBlobs(ctx context.Context, blobs []*azureblob.BlobInfo, startingTime, endingTime time.Time) (numProcessedBlobs int) {
	r.logger.Debug("Received a batch of blobs, parsing through them", zap.Int("num_blobs", len(blobs)))
	processedBlobCount := atomic.Int64{}
	
	for _, blob := range blobs {
		select {
		case <-ctx.Done():
			break
		default:
		}

		// Filter by filename pattern if configured
		if r.filenameRegex != nil {
			// Extract just the filename (not the full path)
			filename := filepath.Base(blob.Name)
			if !r.filenameRegex.MatchString(filename) {
				r.logger.Debug("Skipping blob, filename doesn't match pattern",
					zap.String("blob", blob.Name),
					zap.String("filename", filename),
					zap.String("pattern", r.cfg.FilenamePattern))
				continue
			}
		}

		var blobTime *time.Time
		var telemetryType pipeline.Signal
		var shouldProcess bool

		if r.cfg.UseLastModified {
			// Use LastModified timestamp mode
			if blob.LastModified.IsZero() {
				r.logger.Debug("Skipping blob with zero LastModified", zap.String("blob", blob.Name))
				continue
			}
			blobTime = &blob.LastModified
			telemetryType = r.getTelemetryType()
			shouldProcess = r.checkpoint.ShouldParse(*blobTime, blob.Name) &&
				rehydration.IsInTimeRange(*blobTime, startingTime, endingTime)
		} else if r.cfg.TimePattern != "" {
			// Use custom time pattern mode
			parsedTime, err := ParseTimeFromPattern(blob.Name, r.cfg.TimePattern)
			if err != nil {
				r.logger.Debug("Skipping blob, failed to parse time from pattern",
					zap.String("blob", blob.Name),
					zap.String("pattern", r.cfg.TimePattern),
					zap.Error(err))
				continue
			}
			blobTime = parsedTime
			telemetryType = r.getTelemetryType()
			shouldProcess = r.checkpoint.ShouldParse(*blobTime, blob.Name) &&
				rehydration.IsInTimeRange(*blobTime, startingTime, endingTime)
		} else {
			// Use default structured path parsing mode (year=YYYY/month=MM/...)
			parsedTime, parsedType, err := rehydration.ParseEntityPath(blob.Name)
			switch {
			case errors.Is(err, rehydration.ErrInvalidEntityPath):
				r.logger.Debug("Skipping Blob, non-matching blob path", zap.String("blob", blob.Name))
				continue
			case err != nil:
				r.logger.Error("Error processing blob path", zap.String("blob", blob.Name), zap.Error(err))
				continue
			}
			blobTime = parsedTime
			telemetryType = parsedType
			shouldProcess = r.checkpoint.ShouldParse(*blobTime, blob.Name) &&
				rehydration.IsInTimeRange(*blobTime, startingTime, endingTime) &&
				telemetryType == r.supportedTelemetry
		}

		if shouldProcess {
			r.wg.Add(1)
			go func(blob *azureblob.BlobInfo, blobTime *time.Time) {
				defer r.wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
				}
				
				// Process and consume the blob
				if err := r.processBlob(ctx, blob); err != nil {
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
			}(blob, blobTime)
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
func (r *pollingReceiver) processBlob(ctx context.Context, blob *azureblob.BlobInfo) error {
	// Allocate a buffer the size of the blob
	blobBuffer := make([]byte, blob.Size)

	size, err := r.azureClient.DownloadBlob(ctx, r.cfg.Container, blob.Name, blobBuffer)
	if err != nil {
		return fmt.Errorf("download blob: %w", err)
	}

	// Check file extension to see if we need to decompress
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
const checkpointStorageKey = "azure_blob_polling_checkpoint"

// checkpointKey returns the key used for storing the checkpoint
func (r *pollingReceiver) checkpointKey() string {
	return fmt.Sprintf("%s_%s_%s", checkpointStorageKey, r.id, r.supportedTelemetry.String())
}

func (r *pollingReceiver) makeCheckpoint(ctx context.Context) error {
	if r.lastBlob == nil || r.lastBlobTime == nil {
		return nil
	}
	r.logger.Debug("Making checkpoint", zap.String("blob", r.lastBlob.Name), zap.Time("time", *r.lastBlobTime))
	r.mut.Lock()
	defer r.mut.Unlock()
	r.checkpoint.UpdateCheckpoint(*r.lastBlobTime, r.lastBlob.Name)
	return r.checkpointStore.SaveStorageData(ctx, r.checkpointKey(), r.checkpoint)
}

func (r *pollingReceiver) conditionallyDeleteBlob(ctx context.Context, blob *azureblob.BlobInfo) error {
	if !r.cfg.DeleteOnRead {
		return nil
	}
	return r.azureClient.DeleteBlob(ctx, r.cfg.Container, blob.Name)
}
