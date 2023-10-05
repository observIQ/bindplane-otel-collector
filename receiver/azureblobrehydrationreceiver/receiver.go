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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver/internal/azureblob"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.uber.org/zap"
)

const (
	// azurePathSeparator the path separator that Azure storage uses
	azurePathSeparator = "/"

	// timestampStorageKey the key used for storing the last processed rehydration time
	timestampStorageKey = "last_rehydrated_time"
)

var errInvalidBlobPath = errors.New("invalid blob path")

type rehydrationReceiver struct {
	logger             *zap.Logger
	id                 component.ID
	cfg                *Config
	azureClient        azureblob.BlobClient
	supportedTelemetry component.DataType
	consumer           blobConsumer
	storageClient      storage.Client

	startingTime time.Time
	endingTime   time.Time

	doneChan   chan struct{}
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
	r.consumer = newMetricsConsumer(nextConsumer)

	return r, nil
}

// newLogsReceiver creates a new logs specific receiver.
func newLogsReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Logs) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = component.DataTypeLogs
	r.consumer = newLogsConsumer(nextConsumer)

	return r, nil
}

// newTracesReceiver creates a new traces specific receiver.
func newTracesReceiver(id component.ID, logger *zap.Logger, cfg *Config, nextConsumer consumer.Traces) (*rehydrationReceiver, error) {
	r, err := newRehydrationReceiver(id, logger, cfg)
	if err != nil {
		return nil, err
	}

	r.supportedTelemetry = component.DataTypeTraces
	r.consumer = newTracesConsumer(nextConsumer)

	return r, nil
}

// newRehydrationReceiver creates a new rehydration receiver
func newRehydrationReceiver(id component.ID, logger *zap.Logger, cfg *Config) (*rehydrationReceiver, error) {
	azureClient, err := azureblob.NewAzureBlobClient(cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("new Azure client: %w", err)
	}

	// We should not get an error for either of these time parsings as we check in config validate.
	// Doing error checking anyways just in case.
	startingTime, err := time.Parse(timeFormat, cfg.StartingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid starting_time timestamp: %w", err)
	}

	endingTime, err := time.Parse(timeFormat, cfg.EndingTime)
	if err != nil {
		return nil, fmt.Errorf("invalid ending_time timestamp: %w", err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())

	return &rehydrationReceiver{
		logger:        logger,
		id:            id,
		cfg:           cfg,
		azureClient:   azureClient,
		doneChan:      make(chan struct{}),
		storageClient: storage.NewNopClient(),
		startingTime:  startingTime,
		endingTime:    endingTime,
		ctx:           ctx,
		cancelFunc:    cancel,
	}, nil
}

// Start starts the rehydration receiver
func (r *rehydrationReceiver) Start(ctx context.Context, host component.Host) error {
	if r.cfg.StorageID != nil {
		storageClient, err := adapter.GetStorageClient(ctx, host, r.cfg.StorageID, r.id)
		if err != nil {
			return fmt.Errorf("getStorageClient: %w", err)
		}

		r.storageClient = storageClient
	}

	go r.rehydrateBlobs()
	return nil
}

// Shutdown shuts down the rehydration receiver
func (r *rehydrationReceiver) Shutdown(ctx context.Context) error {
	r.cancelFunc(errors.New("shutdown"))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.doneChan:
		return nil
	}
}

// rehydrateBlobs pulls blob paths from the UI and if they are within the specified
// time range then the blobs will be downloaded and rehydrated.
func (r *rehydrationReceiver) rehydrateBlobs() {
	defer close(r.doneChan)

	// load the previous checkpoint. If not exist should return zero value for time
	checkpointTime := r.loadCheckpoint(r.ctx)

	var prefix *string
	if r.cfg.RootFolder != "" {
		prefix = &r.cfg.RootFolder
	}

	var marker *string
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			// get blobs from Azure
			blobs, nextMarker, err := r.azureClient.ListBlobs(r.ctx, r.cfg.Container, prefix, marker)
			if err != nil {
				r.logger.Error("Failed to list blobs", zap.Error(err))
				continue
			}

			marker = nextMarker

			// Go through each blob and parse it's path to determine if we should consume it or not
			for _, blob := range blobs {
				blobTime, telemetryType, err := r.parseBlobPath(prefix, blob.Name)
				switch {
				case errors.Is(err, errInvalidBlobPath):
					r.logger.Debug("Skipping Blob, non-matching blob path", zap.String("blob", blob.Name))
				case err != nil:
					r.logger.Error("Error processing blob path", zap.String("blob", blob.Name), zap.Error(err))
				case blobTime.Equal(checkpointTime) || blobTime.After(checkpointTime): // If the blob time is the same as or later than our checkpoint then process it
					// if the blob is not in the specified time range or not of the telemetry type supported by this receiver
					// then skip consuming it.
					if !r.isInTimeRange(*blobTime) || telemetryType != r.supportedTelemetry {
						continue
					}

					// Process and consume the blob at the given path
					if err := r.processBlob(blob); err != nil {
						r.logger.Error("Error consuming blob", zap.String("blob", blob.Name), zap.Error(err))
						continue
					}

					// set and save the checkpoint after successfully processing the blob
					checkpointTime = *blobTime
					if err := r.checkpoint(r.ctx, checkpointTime); err != nil {
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

		}
	}
}

// checkpoint saves the checkpoint to keep place in rehydration effort
func (r *rehydrationReceiver) checkpoint(ctx context.Context, checkpointTime time.Time) error {
	data, err := json.Marshal(&checkpointTime)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	return r.storageClient.Set(ctx, timestampStorageKey, data)
}

// loadCheckpoint loads a checkpoint timestamp to be used to keep place in rehydration effort
func (r *rehydrationReceiver) loadCheckpoint(ctx context.Context) time.Time {
	data, err := r.storageClient.Get(ctx, timestampStorageKey)
	if err != nil {
		r.logger.Info("Unable to load checkpoint from storage client, continuing without a previous checkpoint", zap.Error(err))
		return time.Time{}
	}

	if data == nil {
		return time.Time{}
	}

	var checkpointTime time.Time
	if err := json.Unmarshal(data, &checkpointTime); err != nil {
		r.logger.Error("Error while decoding the stored checkpoint, continuing without a checkpoint", zap.Error(err))
		return time.Time{}
	}

	return checkpointTime
}

// constants for blob path parts
const (
	year   = "year="
	month  = "month="
	day    = "day="
	hour   = "hour="
	minute = "minute="
)

// strings that indicate what type of telemetry is in a blob
const (
	metricBlobSignifier = "metrics_"
	logsBlobSignifier   = "logs_"
	tracesBlobSignifier = "traces_"
)

// parseBlobPath returns true if the blob is within the existing time range
func (r *rehydrationReceiver) parseBlobPath(prefix *string, blobName string) (blobTime *time.Time, telemetryType component.DataType, err error) {
	parts := strings.Split(blobName, azurePathSeparator)

	if len(parts) == 0 {
		err = errInvalidBlobPath
		return
	}

	// Build timestamp in 2006-01-02T15:04 format
	tsBuilder := strings.Builder{}

	i := 0
	// If we have a prefix start looking at the second part of the path
	if prefix != nil {
		i = 1
	}

	nextExpectedPart := year
	for ; i < len(parts)-1; i++ {
		part := parts[i]

		if !strings.HasPrefix(part, nextExpectedPart) {
			err = errInvalidBlobPath
			return
		}

		val := strings.TrimPrefix(part, nextExpectedPart)

		switch nextExpectedPart {
		case year:
			nextExpectedPart = month
		case month:
			tsBuilder.WriteString("-")
			nextExpectedPart = day
		case day:
			tsBuilder.WriteString("-")
			nextExpectedPart = hour
		case hour:
			tsBuilder.WriteString("T")
			nextExpectedPart = minute
		case minute:
			tsBuilder.WriteString(":")
			nextExpectedPart = ""
		}

		tsBuilder.WriteString(val)
	}

	// Special case when using hour granularity.
	// There won't be a minute=XX part of the path if we've exited the loop
	// and we still expect minutes just write ':00' for minutes.
	if nextExpectedPart == minute {
		tsBuilder.WriteString(":00")
	}

	// Parse the expected format
	parsedTime, timeErr := time.Parse(timeFormat, tsBuilder.String())
	if err != nil {
		err = fmt.Errorf("parse blob time: %w", timeErr)
		return
	}
	blobTime = &parsedTime

	// For the last part of the path parse the telemetry type
	lastPart := parts[len(parts)-1]
	switch {
	case strings.Contains(lastPart, metricBlobSignifier):
		telemetryType = component.DataTypeMetrics
	case strings.Contains(lastPart, logsBlobSignifier):
		telemetryType = component.DataTypeLogs
	case strings.Contains(lastPart, tracesBlobSignifier):
		telemetryType = component.DataTypeTraces
	}

	return
}

// isInTimeRange returns true if startingTime <= blobTime <= endingTime
func (r *rehydrationReceiver) isInTimeRange(blobTime time.Time) bool {
	return (blobTime.Equal(r.startingTime) || blobTime.After(r.startingTime)) &&
		(blobTime.Equal(r.endingTime) || blobTime.Before(r.endingTime))
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
		blobBuffer, err = gzipDecompress(blobBuffer[:size])
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

// gzipDecompress does a gzip decompression on the passed in contents
func gzipDecompress(contents []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(contents))
	if err != nil {
		return nil, fmt.Errorf("new reader: %w", err)
	}

	result, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("decompression: %w", err)
	}

	return result, nil
}
