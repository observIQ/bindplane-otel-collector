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

package restapireceiver

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

const (
	checkpointStorageKey = "restapi_checkpoint"
)

// checkpointData represents the data stored in the checkpoint.
type checkpointData struct {
	PaginationState *paginationState `json:"pagination_state"`
}

// baseReceiver contains shared functionality for REST API receivers.
type baseReceiver struct {
	settings      component.TelemetrySettings
	logger        *zap.Logger
	cfg           *Config
	client        restAPIClient
	storageClient storage.Client
	id            component.ID

	wg                  sync.WaitGroup
	mu                  sync.Mutex
	currentPollInterval time.Duration // current adaptive poll interval
	cancel              context.CancelFunc
	paginationState     *paginationState
}

// initializeClient creates the HTTP client.
func (b *baseReceiver) initializeClient(ctx context.Context, host component.Host) error {
	client, err := newRESTAPIClient(ctx, b.settings, b.cfg, host)
	if err != nil {
		return fmt.Errorf("failed to create REST API client: %w", err)
	}
	b.client = client
	return nil
}

// initializeStorage sets up storage and loads checkpoint if configured.
func (b *baseReceiver) initializeStorage(ctx context.Context, host component.Host) error {
	if b.cfg.StorageID != nil {
		storageClient, err := adapter.GetStorageClient(ctx, host, b.cfg.StorageID, b.id)
		if err != nil {
			return fmt.Errorf("failed to get storage client: %w", err)
		}
		b.storageClient = storageClient
		b.loadCheckpoint(ctx)
	} else {
		b.storageClient = storage.NewNopClient()
	}
	return nil
}

// initializePagination sets up pagination state.
// Only creates a new state if one wasn't already loaded from checkpoint.
func (b *baseReceiver) initializePagination() {
	if b.paginationState == nil {
		b.paginationState = newPaginationState(b.cfg)
	}
}

// shutdownBase handles common shutdown logic.
func (b *baseReceiver) shutdownBase(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()

	if b.client != nil {
		if err := b.client.Shutdown(); err != nil {
			b.logger.Error("failed to shutdown client", zap.Error(err))
		}
	}

	if b.storageClient != nil {
		if err := b.saveCheckpoint(ctx); err != nil {
			b.logger.Error("failed to save checkpoint", zap.Error(err))
		}
		return b.storageClient.Close(ctx)
	}

	return nil
}

// pollResult contains the outcome of a poll cycle.
type pollResult struct {
	recordCount  int  // total records received across all pages
	lastPageFull bool // true if the poll cycle ended because a page limit was hit, not because data ran out
}

// adjustPollInterval adjusts the poll interval based on the poll result.
// Only resets to min interval when the last page was full (likely more data waiting).
// When the response had data but the last page was partial, we're caught up — back off.
func (b *baseReceiver) adjustPollInterval(result pollResult) {
	if result.lastPageFull {
		// Last page was full - likely more data waiting, poll aggressively
		if b.currentPollInterval != b.cfg.MinPollInterval {
			b.logger.Debug("resetting poll interval after full response",
				zap.Duration("new_interval", b.cfg.MinPollInterval),
				zap.Duration("previous_interval", b.currentPollInterval))
			b.currentPollInterval = b.cfg.MinPollInterval
		}
	} else {
		// No data or partial page - increase interval (backoff) up to max
		newInterval := time.Duration(float64(b.currentPollInterval) * b.cfg.BackoffMultiplier)
		if newInterval > b.cfg.MaxPollInterval {
			newInterval = b.cfg.MaxPollInterval
		}
		if newInterval != b.currentPollInterval {
			b.logger.Debug("increasing poll interval",
				zap.Duration("new_interval", newInterval),
				zap.Duration("previous_interval", b.currentPollInterval),
				zap.Int("record_count", result.recordCount))
			b.currentPollInterval = newInterval
		}
	}
}

// loadCheckpoint loads the checkpoint from storage.
func (b *baseReceiver) loadCheckpoint(ctx context.Context) {
	bytes, err := b.storageClient.Get(ctx, checkpointStorageKey)
	if err != nil {
		b.logger.Debug("unable to load checkpoint, starting fresh", zap.Error(err))
		return
	}

	if bytes == nil {
		return
	}

	var checkpoint checkpointData
	if err := jsoniter.Unmarshal(bytes, &checkpoint); err != nil {
		b.logger.Warn("unable to decode checkpoint, starting fresh", zap.Error(err))
		return
	}

	if checkpoint.PaginationState != nil {
		b.paginationState = checkpoint.PaginationState
	}
}

// saveCheckpoint saves the checkpoint to storage.
// Call this function after every pagination so that the checkpoint has up-to-date pagination state.
func (b *baseReceiver) saveCheckpoint(ctx context.Context) error {
	if b.storageClient == nil || b.paginationState == nil {
		return nil // No storage client or pagination state, nothing to save
	}

	checkpoint := checkpointData{
		PaginationState: b.paginationState,
	}

	bytes, err := jsoniter.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return b.storageClient.Set(ctx, checkpointStorageKey, bytes)
}

// fetchDataPage fetches a single page of data from the API.
// Returns the full response, extracted data, and any error.
func (b *baseReceiver) fetchDataPage(ctx context.Context, requestURL string, params url.Values) (map[string]any, []map[string]any, error) {
	fullResponse, err := b.client.GetFullResponse(ctx, requestURL, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get full response: %w", err)
	}

	data := extractDataFromResponse(fullResponse, b.cfg.ResponseField, b.logger)
	return fullResponse, data, nil
}

// handlePagination checks if there are more pages and updates pagination state.
// Returns true if there are more pages to fetch.
func (b *baseReceiver) handlePagination(fullResponse map[string]any, data []map[string]any) (bool, url.Values) {
	// Check pagination mode
	if b.cfg.Pagination.Mode == paginationModeNone {
		return false, nil
	}

	// Parse pagination response to check if there are more pages
	hasMore, err := parsePaginationResponse(b.cfg, fullResponse, data, b.paginationState, b.logger)
	if err != nil {
		b.logger.Warn("failed to parse pagination response", zap.Error(err))
		return false, nil
	}

	// Check page limit
	if !checkPageLimit(b.cfg, b.paginationState) {
		b.logger.Debug("page limit reached, stopping pagination")
		return false, nil
	}

	if !hasMore {
		return false, nil
	}

	// Update pagination state for next page
	if b.cfg.Pagination.Mode == paginationModeOffsetLimit {
		dataCount := getDataCount(fullResponse)
		if dataCount > 0 {
			b.paginationState.CurrentOffset += dataCount
			b.paginationState.PagesFetched++
		} else {
			updatePaginationState(b.cfg, b.paginationState)
		}
	} else {
		updatePaginationState(b.cfg, b.paginationState)
	}

	// Rebuild params with new pagination state
	params := url.Values{}
	paginationParams := buildPaginationParams(b.cfg, b.paginationState)
	for key, values := range paginationParams {
		for _, value := range values {
			params.Add(key, value)
		}
	}

	return true, params
}

// resetTimestampPagination resets the pages fetched counter after a poll cycle.
// The currentTimestamp is preserved so the next poll starts from where we left off,
// preventing duplicate data from being fetched.
func (b *baseReceiver) resetTimestampPagination() {
	if b.cfg.Pagination.Mode == paginationModeTimestamp {
		b.logger.Debug("resetting timestamp pagination state",
			zap.Time("preserved_timestamp", b.paginationState.CurrentTimestamp),
			zap.Int("pages_fetched_before_reset", b.paginationState.PagesFetched))
		// Only reset the pages fetched counter, NOT the timestamp.
		// The timestamp should persist between poll cycles to avoid re-fetching data.
		b.paginationState.PagesFetched = 0
	}
}

// restAPILogsReceiver is a receiver that pulls logs from a REST API.
type restAPILogsReceiver struct {
	baseReceiver
	consumer consumer.Logs
}

// newRESTAPILogsReceiver creates a new REST API logs receiver.
func newRESTAPILogsReceiver(
	params receiver.Settings,
	cfg *Config,
	cons consumer.Logs,
) (*restAPILogsReceiver, error) {
	return &restAPILogsReceiver{
		baseReceiver: baseReceiver{
			settings: params.TelemetrySettings,
			logger:   params.Logger,
			cfg:      cfg,
			id:       params.ID,
		},
		consumer: cons,
	}, nil
}

// Start starts the receiver.
func (r *restAPILogsReceiver) Start(ctx context.Context, host component.Host) error {
	if err := r.initializeClient(ctx, host); err != nil {
		return err
	}
	if err := r.initializeStorage(ctx, host); err != nil {
		return err
	}
	r.initializePagination()

	cancelCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	return r.startPolling(cancelCtx)
}

// Shutdown stops the receiver.
func (r *restAPILogsReceiver) Shutdown(ctx context.Context) error {
	r.logger.Debug("shutting down REST API logs receiver")
	return r.shutdownBase(ctx)
}

// startPolling starts the polling goroutine.
func (r *restAPILogsReceiver) startPolling(ctx context.Context) error {
	// Initialize with minimum poll interval for responsive startup
	r.currentPollInterval = r.cfg.MinPollInterval

	// Run immediately on startup
	result, err := r.poll(ctx)
	if err != nil {
		r.logger.Error("error on initial poll", zap.Error(err))
		// Continue with periodic polling even if initial poll fails
	}
	r.adjustPollInterval(result)

	// Start periodic polling with adaptive timer
	timer := time.NewTimer(r.currentPollInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				result, err := r.poll(ctx)
				if err != nil {
					r.logger.Error("error while polling", zap.Error(err))
				}
				r.adjustPollInterval(result)
				timer.Reset(r.currentPollInterval)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// poll performs a single polling cycle. Returns a pollResult with the total record count
// and whether the last page was full (indicating more data may be available soon).
func (r *restAPILogsReceiver) poll(ctx context.Context) (pollResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := pollResult{}

	// Build initial pagination parameters
	params := buildPaginationParams(r.cfg, r.paginationState)

	r.logger.Debug("starting poll cycle",
		zap.String("url", r.cfg.URL),
		zap.String("pagination_mode", string(r.cfg.Pagination.Mode)),
		zap.Time("current_timestamp", r.paginationState.CurrentTimestamp),
		zap.Int("pages_fetched", r.paginationState.PagesFetched),
		zap.String("params", params.Encode()))

	// Handle pagination - fetch all pages in this poll cycle
	pageNum := 0
	for {
		pageNum++
		fullResponse, data, err := r.fetchDataPage(ctx, r.cfg.URL, params)
		if err != nil {
			return result, err
		}

		r.logger.Debug("fetched page",
			zap.Int("page_num", pageNum),
			zap.Int("records_in_page", len(data)),
			zap.String("request_params", params.Encode()))

		// Log first and last record timestamps if available for debugging duplicates
		if len(data) > 0 && r.cfg.Pagination.Mode == paginationModeTimestamp {
			timestampField := r.cfg.Pagination.Timestamp.TimestampFieldName
			if timestampField != "" {
				if firstTS, ok := data[0][timestampField]; ok {
					r.logger.Debug("first record in page",
						zap.Int("page_num", pageNum),
						zap.Any("timestamp", firstTS),
						zap.Any("record_preview", truncateRecord(data[0])))
				}
				if lastTS, ok := data[len(data)-1][timestampField]; ok {
					r.logger.Debug("last record in page",
						zap.Int("page_num", pageNum),
						zap.Any("timestamp", lastTS),
						zap.Any("record_preview", truncateRecord(data[len(data)-1])))
				}
			}
		}

		// Convert to logs and consume
		logs := convertJSONToLogs(data, r.logger)
		if logs.LogRecordCount() > 0 {
			result.recordCount += logs.LogRecordCount()
			if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
				return result, fmt.Errorf("failed to consume logs: %w", err)
			}
		}

		// Check for more pages
		hasMore, nextParams := r.handlePagination(fullResponse, data)
		r.logger.Debug("pagination decision",
			zap.Int("page_num", pageNum),
			zap.Bool("has_more", hasMore),
			zap.Time("current_timestamp_state", r.paginationState.CurrentTimestamp))

		if !hasMore {
			// Determine if we stopped because the page limit was reached (API may have more data)
			// or because the data was exhausted (partial/empty last page).
			if len(data) > 0 && r.cfg.Pagination.PageLimit > 0 && !checkPageLimit(r.cfg, r.paginationState) {
				result.lastPageFull = true
			}
			break
		}

		if err := r.saveCheckpoint(ctx); err != nil {
			r.logger.Error("failed to save checkpoint", zap.Error(err))
		}

		params = nextParams
	}

	r.logger.Debug("poll cycle complete",
		zap.Int("total_records", result.recordCount),
		zap.Int("pages_fetched", pageNum),
		zap.Bool("last_page_full", result.lastPageFull),
		zap.Time("final_timestamp_state", r.paginationState.CurrentTimestamp))

	r.resetTimestampPagination()
	return result, nil
}

// truncateRecord creates a preview of a record for logging, limiting to key fields.
func truncateRecord(record map[string]any) map[string]any {
	preview := make(map[string]any)
	count := 0
	for k, v := range record {
		if count >= 3 {
			preview["..."] = fmt.Sprintf("(%d more fields)", len(record)-3)
			break
		}
		// Truncate long string values
		if s, ok := v.(string); ok && len(s) > 50 {
			preview[k] = s[:50] + "..."
		} else {
			preview[k] = v
		}
		count++
	}
	return preview
}

// getNestedField retrieves a value from a nested map using dot notation.
// For example, "response.data" will navigate to response["response"]["data"].
func getNestedField(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// extractDataFromResponse extracts the data array from the full response.
func extractDataFromResponse(response map[string]any, responseField string, logger *zap.Logger) []map[string]any {
	var dataArray []any

	if responseField != "" {
		// Response has a field containing the array (supports dot notation for nested fields)
		fieldValue, ok := getNestedField(response, responseField)
		if !ok {
			logger.Warn("response field not found", zap.String("field", responseField))
			return []map[string]any{}
		}
		dataArray, ok = fieldValue.([]any)
		if !ok {
			logger.Warn("response field is not an array", zap.String("field", responseField))
			return []map[string]any{}
		}
	} else {
		// Response is directly an array (wrapped in map by GetFullResponse)
		if arr, ok := response["data"].([]any); ok {
			dataArray = arr
		} else {
			// Try to find any array field
			for _, val := range response {
				if arr, ok := val.([]any); ok {
					dataArray = arr
					break
				}
			}
		}
	}

	// Convert []any to []map[string]any
	result := make([]map[string]any, 0, len(dataArray))
	for _, item := range dataArray {
		itemMap, ok := item.(map[string]any)
		if !ok {
			logger.Warn("skipping non-object item in array", zap.Any("item", item))
			continue
		}
		result = append(result, itemMap)
	}

	return result
}

// restAPIMetricsReceiver is a receiver that pulls metrics from a REST API.
type restAPIMetricsReceiver struct {
	baseReceiver
	consumer consumer.Metrics
}

// newRESTAPIMetricsReceiver creates a new REST API metrics receiver.
func newRESTAPIMetricsReceiver(
	params receiver.Settings,
	cfg *Config,
	cons consumer.Metrics,
) (*restAPIMetricsReceiver, error) {
	return &restAPIMetricsReceiver{
		baseReceiver: baseReceiver{
			settings: params.TelemetrySettings,
			logger:   params.Logger,
			cfg:      cfg,
			id:       params.ID,
		},
		consumer: cons,
	}, nil
}

// Start starts the receiver.
func (r *restAPIMetricsReceiver) Start(ctx context.Context, host component.Host) error {
	if err := r.initializeClient(ctx, host); err != nil {
		return err
	}
	if err := r.initializeStorage(ctx, host); err != nil {
		return err
	}
	r.initializePagination()

	cancelCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	return r.startPolling(cancelCtx)
}

// Shutdown stops the receiver.
func (r *restAPIMetricsReceiver) Shutdown(ctx context.Context) error {
	r.logger.Debug("shutting down REST API metrics receiver")
	return r.shutdownBase(ctx)
}

// startPolling starts the polling goroutine.
func (r *restAPIMetricsReceiver) startPolling(ctx context.Context) error {
	// Initialize with minimum poll interval for responsive startup
	r.currentPollInterval = r.cfg.MinPollInterval

	// Run immediately on startup
	result, err := r.poll(ctx)
	if err != nil {
		r.logger.Error("error on initial poll", zap.Error(err))
		// Continue with periodic polling even if initial poll fails
	}
	r.adjustPollInterval(result)

	// Start periodic polling with adaptive timer
	timer := time.NewTimer(r.currentPollInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				result, err := r.poll(ctx)
				if err != nil {
					r.logger.Error("error while polling", zap.Error(err))
				}
				r.adjustPollInterval(result)
				timer.Reset(r.currentPollInterval)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// poll performs a single polling cycle. Returns a pollResult with the total record count
// and whether the last page was full (indicating more data may be available soon).
func (r *restAPIMetricsReceiver) poll(ctx context.Context) (pollResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := pollResult{}

	// Build initial pagination parameters
	params := buildPaginationParams(r.cfg, r.paginationState)

	r.logger.Debug("starting poll cycle",
		zap.String("url", r.cfg.URL),
		zap.String("pagination_mode", string(r.cfg.Pagination.Mode)),
		zap.Time("current_timestamp", r.paginationState.CurrentTimestamp),
		zap.Int("pages_fetched", r.paginationState.PagesFetched),
		zap.String("params", params.Encode()))

	// Handle pagination - fetch all pages in this poll cycle
	pageNum := 0
	for {
		pageNum++
		fullResponse, data, err := r.fetchDataPage(ctx, r.cfg.URL, params)
		if err != nil {
			return result, err
		}

		r.logger.Debug("fetched page",
			zap.Int("page_num", pageNum),
			zap.Int("records_in_page", len(data)),
			zap.String("request_params", params.Encode()))

		// Log first and last record timestamps if available for debugging duplicates
		if len(data) > 0 && r.cfg.Pagination.Mode == paginationModeTimestamp {
			timestampField := r.cfg.Pagination.Timestamp.TimestampFieldName
			if timestampField != "" {
				if firstTS, ok := data[0][timestampField]; ok {
					r.logger.Debug("first record in page",
						zap.Int("page_num", pageNum),
						zap.Any("timestamp", firstTS),
						zap.Any("record_preview", truncateRecord(data[0])))
				}
				if lastTS, ok := data[len(data)-1][timestampField]; ok {
					r.logger.Debug("last record in page",
						zap.Int("page_num", pageNum),
						zap.Any("timestamp", lastTS),
						zap.Any("record_preview", truncateRecord(data[len(data)-1])))
				}
			}
		}

		// Convert to metrics and consume
		metrics := convertJSONToMetrics(data, &r.cfg.Metrics, r.logger)
		if metrics.MetricCount() > 0 {
			result.recordCount += metrics.MetricCount()
			if err := r.consumer.ConsumeMetrics(ctx, metrics); err != nil {
				return result, fmt.Errorf("failed to consume metrics: %w", err)
			}
		}

		// Check for more pages
		hasMore, nextParams := r.handlePagination(fullResponse, data)
		r.logger.Debug("pagination decision",
			zap.Int("page_num", pageNum),
			zap.Bool("has_more", hasMore),
			zap.Time("current_timestamp_state", r.paginationState.CurrentTimestamp))

		if !hasMore {
			// Determine if we stopped because the page limit was reached (API may have more data)
			// or because the data was exhausted (partial/empty last page).
			if len(data) > 0 && r.cfg.Pagination.PageLimit > 0 && !checkPageLimit(r.cfg, r.paginationState) {
				result.lastPageFull = true
			}
			break
		}

		if err := r.saveCheckpoint(ctx); err != nil {
			r.logger.Error("failed to save checkpoint", zap.Error(err))
		}

		params = nextParams
	}

	r.logger.Debug("poll cycle complete",
		zap.Int("total_records", result.recordCount),
		zap.Int("pages_fetched", pageNum),
		zap.Bool("last_page_full", result.lastPageFull),
		zap.Time("final_timestamp_state", r.paginationState.CurrentTimestamp))

	r.resetTimestampPagination()
	return result, nil
}
