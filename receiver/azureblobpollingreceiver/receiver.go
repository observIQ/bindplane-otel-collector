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

	"github.com/observiq/bindplane-otel-collector/internal/azureblob"
	"github.com/observiq/bindplane-otel-collector/internal/blobconsume"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
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
	consumer           blobconsume.Consumer
	checkpoint         *PollingCheckPoint
	checkpointStore    storageclient.StorageClient

	pollInterval    time.Duration
	initialLookback time.Duration

	// mutexes for ensuring a thread safe checkpoint
	mut *sync.Mutex
	wg  *sync.WaitGroup

	// lastBlob and lastBlobTime should not be modified without locking the mut mutex
	lastBlob     *azureblob.BlobInfo
	lastBlobTime *time.Time

	filenameRegex *regexp.Regexp
	cancelFunc    context.CancelFunc
}

// newMetricsReceiver creates a new metrics specific receiver.
func newMetricsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Metrics) (*pollingReceiver, error) {
	if cfg.BlobFormat != "" && cfg.BlobFormat != BlobFormatOTLP {
		return nil, fmt.Errorf("blob_format %q is not supported for metrics pipelines, only %q is supported", cfg.BlobFormat, BlobFormatOTLP)
	}

	r, err := newPollingReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalMetrics
	r.consumer = blobconsume.NewMetricsConsumer(nextConsumer)

	return r, nil
}

// newLogsReceiver creates a new logs specific receiver.
func newLogsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Logs) (*pollingReceiver, error) {
	r, err := newPollingReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalLogs

	switch cfg.BlobFormat {
	case BlobFormatJSON:
		logger.Info("Using NDJSON blob format consumer")
		r.consumer = blobconsume.NewNDJSONLogsConsumer(nextConsumer, logger)
	case BlobFormatText:
		logger.Info("Using raw text blob format consumer")
		r.consumer = blobconsume.NewRawTextLogsConsumer(nextConsumer)
	case BlobFormatOTLP, "":
		logger.Info("Using OTLP blob format consumer")
		r.consumer = blobconsume.NewLogsConsumer(nextConsumer)
	default:
		return nil, fmt.Errorf("unsupported blob_format %q, must be one of: %s, %s, %s", cfg.BlobFormat, BlobFormatOTLP, BlobFormatJSON, BlobFormatText)
	}

	return r, nil
}

// newTracesReceiver creates a new traces specific receiver.
func newTracesReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Traces) (*pollingReceiver, error) {
	if cfg.BlobFormat != "" && cfg.BlobFormat != BlobFormatOTLP {
		return nil, fmt.Errorf("blob_format %q is not supported for traces pipelines, only %q is supported", cfg.BlobFormat, BlobFormatOTLP)
	}

	r, err := newPollingReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = pipeline.SignalTraces
	r.consumer = blobconsume.NewTracesConsumer(nextConsumer)

	return r, nil
}

// newPollingReceiver creates a new polling receiver
func newPollingReceiver(id component.ID, logger *zap.Logger, cfg *Config) (*pollingReceiver, error) {
	azureClient, err := newAzureBlobClient(cfg.ConnectionString, cfg.BatchSize, cfg.PageSize, logger)
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

	cancelCtx, cancel := context.WithCancel(context.Background())
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

	// Create fresh channels for this poll to avoid closing already-closed channels
	blobChan := make(chan []*azureblob.BlobInfo)
	errChan := make(chan error)
	doneChan := make(chan struct{})

	pollStartTime := time.Now()
	r.logger.Info("Starting poll",
		zap.Time("poll_time", pollStartTime),
		zap.Time("start", startingTime),
		zap.Time("end", endingTime))

	r.pullBlobs(ctx, startingTime, endingTime, doneChan, errChan, blobChan)

	r.processBlobsLoop(ctx, doneChan, errChan, blobChan, pollStartTime, startingTime, endingTime)
}

func (r *pollingReceiver) pullBlobs(ctx context.Context, startingTime, endingTime time.Time, doneChan chan struct{}, errChan chan error, blobChan chan []*azureblob.BlobInfo) {
	// Determine prefixes to poll
	prefixes := r.generatePrefixes(ctx, startingTime, endingTime)

	// Stream blobs in a goroutine
	go func() {
		defer close(doneChan)
		for _, prefix := range prefixes {
			select {
			case <-ctx.Done():
				return
			default:
			}

			r.logger.Debug("Polling with prefix", zap.Stringp("prefix", prefix))

			// StreamBlobs closes the done channel passed to it, so we create a fresh one
			prefixDoneChan := make(chan struct{})
			r.azureClient.StreamBlobs(ctx, r.cfg.Container, prefix, errChan, blobChan, prefixDoneChan)

			// Wait for this step to complete
			select {
			case <-ctx.Done():
				return
			case <-prefixDoneChan:
				// Continue to the next prefix
			}
		}
	}()
}

func (r *pollingReceiver) generatePrefixes(ctx context.Context, startingTime, endingTime time.Time) []*string {
	// Expand glob patterns in root_folder to discover actual directories.
	// Returns nil when no root_folder is configured (scan everything),
	// or an empty slice when a glob matched zero directories (scan nothing).
	rootFolders := r.expandGlobRootFolders(ctx)

	if rootFolders != nil && len(rootFolders) == 0 {
		// Glob matched zero directories — return empty to scan nothing
		return []*string{}
	}

	if r.cfg.UseTimePatternAsPrefix && r.cfg.TimePattern != "" {
		var allPrefixes []*string
		for _, root := range rootFolders {
			prefixes := r.generatePrefixesForRoot(startingTime, endingTime, root)
			allPrefixes = append(allPrefixes, prefixes...)
		}
		if len(allPrefixes) > 0 {
			return allPrefixes
		}
		// we return a nil entry here to indicate that there is no prefix for this poll
		// when there is no prefix, the StreamBlobs call will scan the entire container
		return []*string{nil}
	}

	if len(rootFolders) > 0 {
		prefixes := make([]*string, len(rootFolders))
		for i := range rootFolders {
			rootCopy := rootFolders[i]
			prefixes[i] = &rootCopy
		}
		return prefixes
	}

	// we return a nil entry here to indicate that there is no prefix for this poll
	// when there is no prefix, the StreamBlobs call will scan the entire container
	return []*string{nil}
}

// expandGlobRootFolders expands glob patterns in root_folder by listing directory prefixes
// from Azure. If root_folder contains no glob characters, returns it as-is with no API call.
func (r *pollingReceiver) expandGlobRootFolders(ctx context.Context) []string {
	if r.cfg.RootFolder == "" {
		return nil
	}

	if !isGlobPattern(r.cfg.RootFolder) {
		return []string{r.cfg.RootFolder}
	}

	staticPrefix, globPattern := splitGlobPrefix(r.cfg.RootFolder)

	prefixes, err := r.azureClient.ListPrefixes(ctx, r.cfg.Container, staticPrefix)
	if err != nil {
		r.logger.Warn("Failed to list prefixes for glob expansion, falling back to literal root_folder",
			zap.String("root_folder", r.cfg.RootFolder),
			zap.Error(err))
		return []string{r.cfg.RootFolder}
	}

	var matched []string
	for _, p := range prefixes {
		if matchGlob(globPattern, p) {
			matched = append(matched, p)
		}
	}

	if len(matched) == 0 {
		r.logger.Warn("Glob pattern matched zero directories, no blobs will be scanned",
			zap.String("root_folder", r.cfg.RootFolder),
			zap.String("static_prefix", staticPrefix),
			zap.Int("candidates", len(prefixes)))
		return []string{}
	}

	r.logger.Info("Glob expanded root_folder",
		zap.String("pattern", r.cfg.RootFolder),
		zap.Strings("matched", matched))

	return matched
}

// generatePrefixesForRoot generates time-based prefixes for a single root folder.
func (r *pollingReceiver) generatePrefixesForRoot(startingTime, endingTime time.Time, rootFolder string) []*string {
	generated, err := generateTimePrefixes(startingTime, endingTime, r.cfg.TimePattern, rootFolder)
	if err != nil {
		r.logger.Error("Failed to generate time prefixes, falling back to root folder",
			zap.String("root_folder", rootFolder),
			zap.Error(err))
		return []*string{&rootFolder}
	}

	prefixes := make([]*string, len(generated))
	for i, prefix := range generated {
		p := prefix
		prefixes[i] = &p
	}
	return prefixes
}

func (r *pollingReceiver) processBlobsLoop(ctx context.Context, doneChan chan struct{}, errChan chan error, blobChan chan []*azureblob.BlobInfo, pollStartTime time.Time, startingTime, endingTime time.Time) {
	totalProcessed := 0
	totalListed := 0
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context cancelled during poll")
			return
		case <-doneChan:
			r.logger.Info("Poll completed",
				zap.Int("total_listed", totalListed),
				zap.Int("total_processed", totalProcessed),
				zap.Int("total_skipped", totalListed-totalProcessed),
				zap.Int("duration_seconds", int(time.Since(pollStartTime).Seconds())))

			// Update checkpoint with poll time
			r.mut.Lock()
			r.checkpoint.UpdatePollTime(endingTime)
			r.mut.Unlock()

			if err := r.makeCheckpoint(ctx); err != nil {
				r.logger.Error("Error saving checkpoint after poll", zap.Error(err))
			}
			return
		case err := <-errChan:
			r.logger.Error("Error during poll", zap.Error(err))
			return
		case br, ok := <-blobChan:
			if !ok {
				r.logger.Info("Poll completed",
					zap.Int("total_listed", totalListed),
					zap.Int("total_processed", totalProcessed),
					zap.Int("total_skipped", totalListed-totalProcessed),
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
			totalListed += len(br)
			numProcessed := r.processBlobs(ctx, br, startingTime, endingTime)
			totalProcessed += numProcessed
		}
	}
}

func (r *pollingReceiver) processBlobs(ctx context.Context, blobs []*azureblob.BlobInfo, startingTime, endingTime time.Time) (numProcessedBlobs int) {
	r.logger.Debug("Received a batch of blobs, parsing through them", zap.Int("num_blobs", len(blobs)))
	processedBlobCount := atomic.Int64{}
	skippedFilename := 0
	skippedOther := 0

blobLoop:
	for _, blob := range blobs {
		select {
		case <-ctx.Done():
			break blobLoop
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
				skippedFilename++
				continue
			}
		}

		blobTime, shouldProcess := r.shouldProcessBlob(blob, startingTime, endingTime)

		if shouldProcess && blobTime != nil {
			r.wg.Add(1)
			go r.processBlobGoRoutine(ctx, blob, blobTime, &processedBlobCount)
		} else {
			skippedOther++
		}
	}

	r.wg.Wait()

	processed := int(processedBlobCount.Load())
	r.logger.Info("Batch processing summary",
		zap.Int("num_in_batch", len(blobs)),
		zap.Int("processed", processed),
		zap.Int("skipped_filename", skippedFilename),
		zap.Int("skipped_other", skippedOther))

	return processed
}

func (r *pollingReceiver) shouldProcessBlob(blob *azureblob.BlobInfo, startingTime, endingTime time.Time) (*time.Time, bool) {
	if r.cfg.UseLastModified {
		// Use LastModified timestamp mode
		if blob.LastModified.IsZero() {
			r.logger.Debug("Skipping blob with zero LastModified", zap.String("blob", blob.Name))
			return nil, false
		}
		shouldParse := r.checkpoint.ShouldParse(blob.LastModified, blob.Name)
		inRange := blobconsume.IsInTimeRange(blob.LastModified, startingTime, endingTime)
		if !shouldParse {
			r.logger.Debug("Skipping blob, already processed (checkpoint)",
				zap.String("blob", blob.Name),
				zap.Time("last_ts", r.checkpoint.LastTs))
		} else if !inRange {
			r.logger.Debug("Skipping blob, outside time range",
				zap.String("blob", blob.Name),
				zap.Time("blob_time", blob.LastModified),
				zap.Time("start", startingTime),
				zap.Time("end", endingTime))
		}
		return &blob.LastModified, shouldParse && inRange
	}

	if r.cfg.TimePattern != "" {
		// Use custom time pattern mode
		parsedTime, err := parseTimeFromPattern(blob.Name, r.cfg.TimePattern)
		if err != nil {
			r.logger.Debug("Skipping blob, failed to parse time from pattern",
				zap.String("blob", blob.Name),
				zap.String("pattern", r.cfg.TimePattern),
				zap.Error(err))
			return nil, false
		}
		shouldParse := r.checkpoint.ShouldParse(*parsedTime, blob.Name)
		inRange := blobconsume.IsInTimeRange(*parsedTime, startingTime, endingTime)
		if !shouldParse {
			r.logger.Debug("Skipping blob, already processed (checkpoint)",
				zap.String("blob", blob.Name),
				zap.Time("last_ts", r.checkpoint.LastTs))
		} else if !inRange {
			r.logger.Debug("Skipping blob, outside time range",
				zap.String("blob", blob.Name),
				zap.Time("blob_time", *parsedTime),
				zap.Time("start", startingTime),
				zap.Time("end", endingTime))
		}
		return parsedTime, shouldParse && inRange
	}

	// Use default structured path parsing mode (year=YYYY/month=MM/...)
	parsedTime, parsedType, err := blobconsume.ParseEntityPath(blob.Name)
	switch {
	case errors.Is(err, blobconsume.ErrInvalidEntityPath):
		r.logger.Debug("Skipping Blob, non-matching blob path", zap.String("blob", blob.Name))
		return nil, false
	case err != nil:
		r.logger.Error("Error processing blob path", zap.String("blob", blob.Name), zap.Error(err))
		return nil, false
	}
	shouldParse := r.checkpoint.ShouldParse(*parsedTime, blob.Name)
	inRange := blobconsume.IsInTimeRange(*parsedTime, startingTime, endingTime)
	if !shouldParse {
		r.logger.Debug("Skipping blob, already processed (checkpoint)",
			zap.String("blob", blob.Name),
			zap.Time("last_ts", r.checkpoint.LastTs))
	} else if !inRange {
		r.logger.Debug("Skipping blob, outside time range",
			zap.String("blob", blob.Name),
			zap.Time("blob_time", *parsedTime),
			zap.Time("start", startingTime),
			zap.Time("end", endingTime))
	}
	return parsedTime, shouldParse && inRange &&
		parsedType == r.supportedTelemetry
}

func (r *pollingReceiver) processBlobGoRoutine(ctx context.Context, blob *azureblob.BlobInfo, blobTime *time.Time, processedBlobCount *atomic.Int64) {
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

	r.mut.Lock()
	if r.lastBlobTime == nil || r.lastBlobTime.Before(*blobTime) {
		r.lastBlob = blob
		r.lastBlobTime = blobTime
	}
	r.mut.Unlock()
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
		blobBuffer, err = blobconsume.GzipDecompress(blobBuffer[:size])
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
	r.mut.Lock()
	defer r.mut.Unlock()
	r.checkpoint.UpdateCheckpoint(*r.lastBlobTime, r.lastBlob.Name)
	r.logger.Info("Checkpoint saved",
		zap.Time("last_ts", r.checkpoint.LastTs),
		zap.Int("parsed_entities_count", len(r.checkpoint.ParsedEntities)),
		zap.Time("last_poll_time", r.checkpoint.LastPollTime))
	return r.checkpointStore.SaveStorageData(ctx, r.checkpointKey(), r.checkpoint)
}

func (r *pollingReceiver) conditionallyDeleteBlob(ctx context.Context, blob *azureblob.BlobInfo) error {
	if !r.cfg.DeleteOnRead {
		return nil
	}
	return r.azureClient.DeleteBlob(ctx, r.cfg.Container, blob.Name)
}
