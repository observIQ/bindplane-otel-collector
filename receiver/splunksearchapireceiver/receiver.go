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
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type splunksearchapireceiver struct {
	host           component.Host
	logger         *zap.Logger
	logsConsumer   consumer.Logs
	config         *Config
	settings       component.TelemetrySettings
	client         *http.Client
	eventBatchSize int
	wg             *sync.WaitGroup
}

func (ssapir *splunksearchapireceiver) Start(ctx context.Context, host component.Host) error {
	ssapir.host = host
	client, err := ssapir.config.ClientConfig.ToClient(ctx, host, ssapir.settings)
	if err != nil {
		return err
	}
	ssapir.client = client
	go ssapir.runQueries(ctx)
	return nil
}

func (ssapir *splunksearchapireceiver) Shutdown(_ context.Context) error {
	return nil
}

func (ssapir *splunksearchapireceiver) runQueries(ctx context.Context) error {
	for _, search := range ssapir.config.Searches {
		// create search in Splunk
		ssapir.logger.Info("creating search", zap.String("query", search.Query))
		searchID, err := ssapir.createSplunkSearch(search.Query)
		if err != nil {
			ssapir.logger.Error("error creating search", zap.Error(err))
		}

		// wait for search to complete
		if err = ssapir.pollSearchCompletion(ctx, searchID); err != nil {
			ssapir.logger.Error("error polling for search completion", zap.Error(err))
		}

		var resultCountTracker = 0 // track number of results exported
		var offset = 0             // offset for pagination
		for {
			// fetch search results
			ssapir.logger.Info("fetching search results")
			results, err := ssapir.getSplunkSearchResults(searchID, offset)
			if err != nil {
				ssapir.logger.Error("error fetching search results", zap.Error(err))
			}
			ssapir.logger.Info("search results fetched", zap.Int("num_results", len(results.Results)))

			// parse time strings to time.Time
			earliestTime, err := time.Parse(time.RFC3339, search.EarliestTime)
			if err != nil {
				// should be impossible to reach with config validation
				ssapir.logger.Error("earliest_time failed to be parsed as RFC3339", zap.Error(err))
			}

			latestTime, err := time.Parse(time.RFC3339, search.LatestTime)
			if err != nil {
				// should be impossible to reach with config validation
				ssapir.logger.Error("latest_time failed to be parsed as RFC3339", zap.Error(err))
			}

			logs := plog.NewLogs()
			for idx, splunkLog := range results.Results {
				if idx >= search.Limit && search.Limit != 0 {
					break
				}
				// convert log timestamp to ISO8601 (UTC() makes RFC3339 into ISO8601)
				logTimestamp, err := time.Parse(time.RFC3339, splunkLog.Time)
				if err != nil {
					ssapir.logger.Error("error parsing log timestamp", zap.Error(err))
					break
				}
				if logTimestamp.UTC().After(latestTime.UTC()) {
					ssapir.logger.Info("skipping log entry - timestamp after latestTime", zap.Time("time", logTimestamp.UTC()), zap.Time("latestTime", latestTime.UTC()))
					// logger will only log up to 10 times for a given code block, known weird behavior
					continue
				}
				if logTimestamp.UTC().Before(earliestTime) {
					ssapir.logger.Info("skipping log entry - timestamp before earliestTime", zap.Time("time", logTimestamp.UTC()), zap.Time("earliestTime", earliestTime.UTC()))
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
				return nil
			}

			// pass logs, wait for exporter to confirm successful export to GCP
			err = ssapir.logsConsumer.ConsumeLogs(ctx, logs)
			if err != nil {
				// Error from down the pipeline, freak out
				ssapir.logger.Error("error consuming logs", zap.Error(err))
			}
			// if the number of results is less than the results per request, we have queried all pages for the search
			if len(results.Results) < ssapir.eventBatchSize {
				resultCountTracker += len(results.Results)
				break
			}
			resultCountTracker += logs.ResourceLogs().Len()
			offset += ssapir.eventBatchSize
			time.Sleep(5 * time.Second)
		}
		ssapir.logger.Info("search results exported", zap.String("query", search.Query), zap.Int("results", resultCountTracker))
		// search results completely exported, reset checkpoint storage
	}
	return nil
}

func (ssapir *splunksearchapireceiver) pollSearchCompletion(ctx context.Context, searchID string) error {
	t := time.NewTicker(ssapir.config.JobPollInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			ssapir.logger.Info("polling for search completion")
			done, err := ssapir.isSearchCompleted(searchID)
			if err != nil {
				return fmt.Errorf("error polling for search completion: %v", err)
			}
			if done {
				ssapir.logger.Info("search completed")
				return nil
			}
			ssapir.logger.Info("search not completed yet")
		case <-ctx.Done():
			return nil
		}
	}
}

func (ssapir *splunksearchapireceiver) createSplunkSearch(search string) (string, error) {
	resp, err := ssapir.createSearchJob(search)
	if err != nil {
		return "", err
	}
	return resp.SID, nil
}

func (ssapir *splunksearchapireceiver) isSearchCompleted(sid string) (bool, error) {
	resp, err := ssapir.getJobStatus(sid)
	if err != nil {
		return false, err
	}

	for _, key := range resp.Content.Dict.Keys {
		if key.Name == "dispatchState" {
			if key.Value == "DONE" {
				return true, nil
			}
			break
		}
	}
	return false, nil
}

func (ssapir *splunksearchapireceiver) getSplunkSearchResults(sid string, offset int) (SearchResults, error) {
	resp, err := ssapir.getSearchResults(sid, offset)
	if err != nil {
		return SearchResults{}, err
	}
	return resp, nil
}
