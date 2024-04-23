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

package awss3rehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver"

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/observiq/bindplane-agent/internal/rehydration"
	"github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver/internal/aws"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

// newAWSS3Client is the function used to create new AWS S3 clients.
// Meant to be overwritten for tests
var newAWSS3Client = aws.NewAWSClient

type rehydrationReceiver struct {
	logger             *zap.Logger
	id                 component.ID
	cfg                *Config
	awsClient          aws.S3Client
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
	awsClient, err := newAWSS3Client(cfg.Region, cfg.RoleArn)
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

	ctx, cancel := context.WithCancelCause(context.Background())

	return &rehydrationReceiver{
		logger:          logger,
		id:              id,
		cfg:             cfg,
		awsClient:       awsClient,
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
	numBlobsRehydrated := r.rehydrate(checkpoint, marker)
	emptyEntityCounter := checkEntityCount(numBlobsRehydrated, 0)

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			// Polling for blobs has egress charges so we want to stop polling
			// after we stop finding blobs.
			if emptyEntityCounter == emptyPollLimit {
				return
			}

			numBlobsRehydrated := r.rehydrate(checkpoint, marker)
			emptyEntityCounter = checkEntityCount(numBlobsRehydrated, emptyEntityCounter)
		}
	}
}

func (r *rehydrationReceiver) rehydrate(checkpoint *rehydration.CheckPoint, marker *string) (numEntitiesRehydrated int) {
	rehydrateCtx, cancel := context.WithTimeout(r.ctx, r.cfg.PollTimeout)
	defer cancel()

	var prefix *string
	if r.cfg.S3Prefix != "" {
		prefix = &r.cfg.S3Prefix
	}

	objects, nextMarker, err := r.awsClient.ListObjects(rehydrateCtx, r.cfg.S3Bucket, prefix, marker)
	if err != nil {
		r.logger.Error("Failed to list objects", zap.Error(err))
		return
	}

	marker = nextMarker

	processedObjectNames := make([]string, 0, len(objects))

	for _, object := range objects {
		r.logger.Debug("Object", zap.String("name", object.Name))
		objectTime, telemetryType, err := rehydration.ParseEntityPath(object.Name)
		switch {
		case errors.Is(err, rehydration.ErrInvalidEntityPath):
			r.logger.Debug("Skipping Object, non-matching entity path", zap.String("object", object.Name))
		case err != nil:
			r.logger.Error("Error processing object path", zap.String("object", object.Name), zap.Error(err))
		case checkpoint.ShouldParse(*objectTime, object.Name):
			// if the object is not in the specified time range or not of the telemetry type supported by this receiver
			// then skip consuming it.
			if !rehydration.IsInTimeRange(*objectTime, r.startingTime, r.endingTime) || telemetryType != r.supportedTelemetry {
				continue
			}

			// Process and consume the object at the given path
			if err := r.processObject(object); err != nil {
				r.logger.Error("Error consuming object", zap.String("object", object.Name), zap.Error(err))
				continue
			}

			checkpoint.UpdateCheckpoint(*objectTime, object.Name)
			if err := r.checkpointStore.SaveCheckpoint(r.ctx, r.checkpointKey(), checkpoint); err != nil {
				r.logger.Error("Error while saving checkpoint", zap.Error(err))
			}

			// keep track of object names for number processed and deleting
			processedObjectNames = append(processedObjectNames, object.Name)
		}
	}

	numEntitiesRehydrated = len(processedObjectNames)

	// Delete objects
	if r.cfg.DeleteOnRead && len(processedObjectNames) > 0 {
		if err := r.awsClient.DeleteObjects(r.ctx, r.cfg.S3Bucket, processedObjectNames); err != nil {
			r.logger.Error("Error while attempting to delete objects", zap.Error(err))
		}
	}

	return
}

// processObject does the following:
// 1. Downloads the object
// 2. Decompresses the object if applicable
// 3. Pass the object to the consumer
func (r *rehydrationReceiver) processObject(object *aws.ObjectInfo) error {
	objectBuffer := make([]byte, object.Size)

	size, err := r.awsClient.DownloadObject(r.ctx, r.cfg.S3Bucket, object.Name, objectBuffer)
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

	if err := r.consumer.Consume(r.ctx, objectBuffer); err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	return nil
}

// checkpointStorageKey is the key used for storing the checkpoint
const checkpointStorageKey = "aws_s3_checkpoint"

// checkpointKey returns the key used for storing the checkpoint
func (r *rehydrationReceiver) checkpointKey() string {
	return fmt.Sprintf("%s_%s_%s", checkpointStorageKey, r.id, r.supportedTelemetry.String())
}

// checkEntityCount checks the number of entities rehydrated and the current state of the
// empty counter. If zero entities were rehydrated increment the counter.
// If there were entities rehydrated reset the counter as we want to track consecutive zero sized polls.
func checkEntityCount(numEntitiesRehydrated, emptyEntityCounter int) int {
	switch {
	case emptyEntityCounter == emptyPollLimit: // If we are at the limit return the limit
		return emptyPollLimit
	case numEntitiesRehydrated == 0: // If no entities were rehydrated then increment the empty entities counter
		return emptyEntityCounter + 1
	default: // Default case is numEntitiesRehydrated > 0 so reset emptyEntityCounter to 0
		return 0
	}
}
