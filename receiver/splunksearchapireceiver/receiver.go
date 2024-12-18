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

package splunksearchapireceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

const (
	eventStorageKey             = "last_event_offset"
	splunkDefaultEventBatchSize = 100
)

var (
	offset         = 0     // offset for pagination and checkpointing
	exportedEvents = 0     // track the number of events returned by the results endpoint that are exported
	limitReached   = false // flag to stop processing search results when limit is reached
)

type splunksearchapireceiver struct {
	logger           *zap.Logger
	logsConsumer     consumer.Logs
	config           *Config
	settings         component.TelemetrySettings
	id               component.ID
	cancel           context.CancelFunc
	client           splunkSearchAPIClient
	storageClient    storage.Client
	checkpointRecord *EventRecord
}

func newSSAPIReceiver(
	logger *zap.Logger,
	config *Config,
	settings component.TelemetrySettings,
	id component.ID,
) *splunksearchapireceiver {
	return &splunksearchapireceiver{
		logger:           logger,
		config:           config,
		settings:         settings,
		storageClient:    storage.NewNopClient(),
		id:               id,
		checkpointRecord: &EventRecord{},
	}
}

func (ssapir *splunksearchapireceiver) Start(ctx context.Context, host component.Host) error {
	var err error
	ssapir.client, err = newDefaultSplunkSearchAPIClient(ctx, ssapir.settings, *ssapir.config, host)
	if err != nil {
		return err
	}

	// set cancel function
	cancelCtx, cancel := context.WithCancel(ctx)
	ssapir.cancel = cancel

	// create storage client
	storageClient, err := adapter.GetStorageClient(ctx, host, ssapir.config.StorageID, ssapir.id)
	if err != nil {
		return fmt.Errorf("failed to get storage client: %w", err)
	}
	ssapir.storageClient = storageClient

	err = ssapir.initCheckpoint(cancelCtx)
	if err != nil {
		return fmt.Errorf("failed to initialize checkpoint: %w", err)
	}
	go ssapir.runQueries(cancelCtx)
	return nil
}

func (ssapir *splunksearchapireceiver) Shutdown(ctx context.Context) error {
	ssapir.logger.Debug("shutting down logs receiver")
	if ssapir.cancel != nil {
		ssapir.cancel()
	}

	if ssapir.storageClient != nil {
		if err := ssapir.checkpoint(ctx); err != nil {
			ssapir.logger.Error("failed checkpoint", zap.Error(err))
		}
		return ssapir.storageClient.Close(ctx)
	}
	return nil
}

func (ssapir *splunksearchapireceiver) runQueries(ctx context.Context) {
	for _, search := range ssapir.config.Searches {
		// set current search query
		ssapir.checkpointRecord.Search = search.Query

		// set default event batch size (matches Splunk API default)
		if search.EventBatchSize == 0 {
			search.EventBatchSize = splunkDefaultEventBatchSize
		}

		// parse time strings to time.Time
		earliestTime, _ := time.Parse(time.RFC3339, search.EarliestTime)
		latestTime, _ := time.Parse(time.RFC3339, search.LatestTime)

		// create search in Splunk
		searchID, err := ssapir.createSplunkSearch(search)
		if err != nil {
			ssapir.logger.Error("error creating search", zap.Error(err))
			return
		}

		// wait for search to complete
		if err = ssapir.pollSearchCompletion(ctx, searchID); err != nil {
			ssapir.logger.Error("error polling for search completion", zap.Error(err))
			return
		}

		for {
			if ctx.Err() != nil {
				ssapir.logger.Error("context cancelled, stopping search result export", zap.Error(ctx.Err()))
				return
			}

			ssapir.logger.Info("fetching search results")
			results, err := ssapir.getSplunkSearchResults(searchID, offset, search.EventBatchSize)
			if err != nil {
				ssapir.logger.Error("error fetching search results", zap.Error(err))
			}
			ssapir.logger.Info("search results fetched", zap.Int("num_results", len(results.Results)))

			logs := plog.NewLogs()
			for idx, splunkLog := range results.Results {
				if (idx+exportedEvents) >= search.Limit && search.Limit != 0 {
					limitReached = true
					break
				}
				// convert log timestamp to ISO 8601 (UTC() makes RFC 3339 into ISO 8601)
				logTimestamp, err := time.Parse(time.RFC3339, splunkLog.Time)
				if err != nil {
					ssapir.logger.Error("error parsing log timestamp", zap.Error(err))
					break
				}
				if logTimestamp.UTC().Before(earliestTime.UTC()) {
					ssapir.logger.Info("skipping log entry - timestamp before earliestTime", zap.Time("time", logTimestamp.UTC()), zap.Time("earliestTime", earliestTime.UTC()))
					break
				}
				if logTimestamp.UTC().After(latestTime.UTC()) {
					ssapir.logger.Info("skipping log entry - timestamp after latestTime", zap.Time("time", logTimestamp.UTC()), zap.Time("latestTime", latestTime.UTC()))
					continue
				}
				log := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				// convert time to timestamp
				timestamp := pcommon.NewTimestampFromTime(logTimestamp.UTC())
				log.SetTimestamp(timestamp)
				log.Body().SetStr(splunkLog.Raw)
			}

			if logs.ResourceLogs().Len() == 0 {
				ssapir.logger.Info("search returned no logs within the given time range")
				break
			}

			// pass logs, wait for exporter to confirm successful export to GCP
			err = ssapir.logsConsumer.ConsumeLogs(ctx, logs)
			if err != nil {
				// error from down the pipeline, freak out
				ssapir.logger.Error("error exporting logs", zap.Error(err))
				return
			}
			// last batch of logs has been successfully exported
			exportedEvents += logs.ResourceLogs().Len()
			offset += len(results.Results)

			// update checkpoint
			ssapir.checkpointRecord.Offset = offset
			err = ssapir.checkpoint(ctx)
			if err != nil {
				ssapir.logger.Error("error writing checkpoint", zap.Error(err))
			}
			if limitReached {
				ssapir.logger.Info("limit reached, stopping search result export")
				break
			}
			// if the number of results is less than the results per request, we have queried all pages for the search
			if len(results.Results) < search.EventBatchSize {
				ssapir.logger.Debug("results less than batch size, stopping search result export")
				break
			}
		}
		ssapir.logger.Info("search results exported", zap.String("query", search.Query), zap.Int("total results", exportedEvents))
	}
	ssapir.logger.Info("all search results exported")
}

func (ssapir *splunksearchapireceiver) pollSearchCompletion(ctx context.Context, searchID string) error {
	t := time.NewTicker(ssapir.config.JobPollInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			ssapir.logger.Debug("polling for search completion")
			resp, err := ssapir.client.GetJobStatus(searchID)
			if err != nil {
				return fmt.Errorf("error getting search job status: %v", err)
			}
			done := ssapir.isSearchCompleted(resp)
			if done {
				ssapir.logger.Info("search completed")
				return nil
			}
			ssapir.logger.Debug("search not completed yet")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ssapir *splunksearchapireceiver) createSplunkSearch(search Search) (string, error) {
	timeFormat := "%Y-%m-%dT%H:%M:%S"
	searchQuery := fmt.Sprintf("%s starttime=\"%s\" endtime=\"%s\" timeformat=\"%s\"", search.Query, search.EarliestTime, search.LatestTime, timeFormat)
	ssapir.logger.Info("creating search", zap.String("query", searchQuery))
	resp, err := ssapir.client.CreateSearchJob(searchQuery)
	if err != nil {
		return "", err
	}
	return resp.SID, nil
}

func (ssapir *splunksearchapireceiver) isSearchCompleted(resp SearchJobStatusResponse) bool {
	for _, key := range resp.Content.Dict.Keys {
		if key.Name == "dispatchState" {
			if key.Value == "DONE" {
				return true
			}
			break
		}
	}
	return false
}

func (ssapir *splunksearchapireceiver) getSplunkSearchResults(sid string, offset int, batchSize int) (SearchResults, error) {
	resp, err := ssapir.client.GetSearchResults(sid, offset, batchSize)
	if err != nil {
		return SearchResults{}, err
	}
	return resp, nil
}

func (ssapir *splunksearchapireceiver) initCheckpoint(ctx context.Context) error {
	ssapir.logger.Debug("initializing checkpoint")
	// if a checkpoint already exists, use the offset from the checkpoint
	if err := ssapir.loadCheckpoint(ctx); err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}
	if ssapir.checkpointRecord.Offset != 0 {
		// check if the search query in the checkpoint record matches any of the search queries in the config
		for idx, search := range ssapir.config.Searches {
			if search.Query == ssapir.checkpointRecord.Search {
				ssapir.logger.Info("found offset checkpoint in storage extension", zap.Int("offset", ssapir.checkpointRecord.Offset), zap.String("search", ssapir.checkpointRecord.Search))
				// skip searches that have already been processed, use the offset from the checkpoint
				ssapir.config.Searches = ssapir.config.Searches[idx:]
				offset = ssapir.checkpointRecord.Offset
				return nil
			}
		}
		ssapir.logger.Info("while initializing checkpoint, no matching search query found, starting from the beginning")
	}
	return nil
}

func (ssapir *splunksearchapireceiver) checkpoint(ctx context.Context) error {
	if ssapir.checkpointRecord == nil {
		return nil
	}

	marshalBytes, err := json.Marshal(ssapir.checkpointRecord)
	if err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}
	return ssapir.storageClient.Set(ctx, eventStorageKey, marshalBytes)
}

func (ssapir *splunksearchapireceiver) loadCheckpoint(ctx context.Context) error {
	marshalBytes, err := ssapir.storageClient.Get(ctx, eventStorageKey)
	if err != nil {
		return err
	}
	if marshalBytes == nil {
		ssapir.logger.Info("no checkpoint found")
		return nil
	}
	return json.Unmarshal(marshalBytes, ssapir.checkpointRecord)
}
