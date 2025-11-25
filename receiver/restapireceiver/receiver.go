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

	// Adaptive polling constants
	minPollInterval   = 10 * time.Millisecond // Minimum poll interval (fastest polling rate)
	backoffMultiplier = 2.0                   // Multiplier for increasing interval on empty responses
)


// checkpointData represents the data stored in the checkpoint.
type checkpointData struct {
	PaginationState *paginationState `json:"pagination_state"`
}

// restAPILogsReceiver is a receiver that pulls logs from a REST API.
type restAPILogsReceiver struct {
	settings      component.TelemetrySettings
	logger        *zap.Logger
	consumer      consumer.Logs
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

// newRESTAPILogsReceiver creates a new REST API logs receiver.
func newRESTAPILogsReceiver(
	params receiver.Settings,
	cfg *Config,
	consumer consumer.Logs,
) (*restAPILogsReceiver, error) {
	return &restAPILogsReceiver{
		settings: params.TelemetrySettings,
		logger:   params.Logger,
		consumer: consumer,
		cfg:      cfg,
		id:       params.ID,
	}, nil
}

// Start starts the receiver.
func (r *restAPILogsReceiver) Start(ctx context.Context, host component.Host) error {
	// Create HTTP client
	client, err := newRESTAPIClient(ctx, r.settings, r.cfg, host)
	if err != nil {
		return fmt.Errorf("failed to create REST API client: %w", err)
	}
	r.client = client

	// Initialize storage client if configured
	if r.cfg.StorageID != nil {
		storageClient, err := adapter.GetStorageClient(ctx, host, r.cfg.StorageID, r.id)
		if err != nil {
			return fmt.Errorf("failed to get storage client: %w", err)
		}
		r.storageClient = storageClient
		r.loadCheckpoint(ctx)
	} else {
		r.storageClient = storage.NewNopClient()
		r.loadCheckpoint(ctx) // Still load checkpoint even with nop client
	}

	// Initialize pagination state
	r.paginationState = newPaginationState(r.cfg)
	// Set limit if configured
	if r.cfg.Pagination.Mode == paginationModeOffsetLimit && r.cfg.Pagination.OffsetLimit.LimitFieldName != "" {
		// Use a reasonable default limit (can be overridden by API)
		r.paginationState.limit = 10
	}

	// Start polling
	cancelCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	return r.startPolling(cancelCtx)
}

// Shutdown stops the receiver.
func (r *restAPILogsReceiver) Shutdown(ctx context.Context) error {
	r.logger.Debug("shutting down REST API logs receiver")
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()

	if r.storageClient != nil {
		if err := r.saveCheckpoint(ctx); err != nil {
			r.logger.Error("failed to save checkpoint", zap.Error(err))
		}
		return r.storageClient.Close(ctx)
	}

	return nil
}

// startPolling starts the polling goroutine.
func (r *restAPILogsReceiver) startPolling(ctx context.Context) error {
	// Initialize with minimum poll interval for responsive startup
	r.currentPollInterval = minPollInterval

	// Run immediately on startup
	recordCount, err := r.poll(ctx)
	if err != nil {
		r.logger.Error("error on initial poll", zap.Error(err))
		// Continue with periodic polling even if initial poll fails
	}
	r.adjustPollInterval(recordCount)

	// Start periodic polling with adaptive timer
	timer := time.NewTimer(r.currentPollInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				recordCount, err := r.poll(ctx)
				if err != nil {
					r.logger.Error("error while polling", zap.Error(err))
				}
				r.adjustPollInterval(recordCount)
				timer.Reset(r.currentPollInterval)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// adjustPollInterval adjusts the poll interval based on whether data was received.
func (r *restAPILogsReceiver) adjustPollInterval(recordCount int) {
	if recordCount > 0 {
		// Data received - reset to minimum interval for responsive polling
		if r.currentPollInterval != minPollInterval {
			r.logger.Debug("resetting poll interval after receiving data",
				zap.Duration("new_interval", minPollInterval),
				zap.Duration("previous_interval", r.currentPollInterval))
			r.currentPollInterval = minPollInterval
		}
	} else {
		// No data - increase interval (backoff) up to max
		newInterval := time.Duration(float64(r.currentPollInterval) * backoffMultiplier)
		if newInterval > r.cfg.MaxPollInterval {
			newInterval = r.cfg.MaxPollInterval
		}
		if newInterval != r.currentPollInterval {
			r.logger.Debug("increasing poll interval due to no data",
				zap.Duration("new_interval", newInterval),
				zap.Duration("previous_interval", r.currentPollInterval))
			r.currentPollInterval = newInterval
		}
	}
}

// poll performs a single polling cycle. Returns the total number of records received.
func (r *restAPILogsReceiver) poll(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	totalRecords := 0

	// Build request URL with pagination
	requestURL := r.cfg.URL
	params := url.Values{}

	// Add pagination parameters
	paginationParams := buildPaginationParams(r.cfg, r.paginationState)
	for key, values := range paginationParams {
		for _, value := range values {
			params.Add(key, value)
		}
	}

	// Handle pagination - fetch all pages in this poll cycle
	for {
		// Get full response to parse pagination info and extract data
		fullResponse, err := r.client.GetFullResponse(ctx, requestURL, params)
		if err != nil {
			return totalRecords, fmt.Errorf("failed to get full response: %w", err)
		}

		// Extract data array from response
		data := extractDataFromResponse(fullResponse, r.cfg.ResponseField, r.logger)

		// Convert to logs
		logs := convertJSONToLogs(data, r.logger)

		// Consume logs if any
		if logs.LogRecordCount() > 0 {
			totalRecords += logs.LogRecordCount()
			if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
				return totalRecords, fmt.Errorf("failed to consume logs: %w", err)
			}
		}

		// Check pagination
		if r.cfg.Pagination.Mode == paginationModeNone {
			// No pagination, we're done
			break
		}

		// Parse pagination response to check if there are more pages
		// Pass the already-extracted data array to avoid re-extraction inconsistencies
		hasMore, err := parsePaginationResponse(r.cfg, fullResponse, data, r.paginationState)
		if err != nil {
			r.logger.Warn("failed to parse pagination response", zap.Error(err))
			break
		}

		// Check page limit
		if !checkPageLimit(r.cfg, r.paginationState) {
			r.logger.Debug("page limit reached, stopping pagination")
			break
		}

		if !hasMore {
			// No more pages
			break
		}

		// Update pagination state for next page
		// For offset/limit, use actual data count instead of limit
		if r.cfg.Pagination.Mode == paginationModeOffsetLimit {
			dataCount := getDataCount(fullResponse)
			if dataCount > 0 {
				r.paginationState.currentOffset += dataCount
				r.paginationState.pagesFetched++
			} else {
				// Fallback to using limit
				updatePaginationState(r.cfg, r.paginationState)
			}
		} else {
			updatePaginationState(r.cfg, r.paginationState)
		}

		// Rebuild params with new pagination state
		params = url.Values{}
		paginationParams := buildPaginationParams(r.cfg, r.paginationState)
		for key, values := range paginationParams {
			for _, value := range values {
				params.Add(key, value)
			}
		}
	}

	// Reset timestamp after completing poll cycle
	// This ensures next poll starts fresh from the initial timestamp
	if r.cfg.Pagination.Mode == paginationModeTimestamp {
		if !r.cfg.Pagination.Timestamp.InitialTimestamp.IsZero() {
			r.paginationState.currentTimestamp = r.cfg.Pagination.Timestamp.InitialTimestamp
		} else {
			r.paginationState.currentTimestamp = time.Time{}
		}
		r.paginationState.pagesFetched = 0
	}

	return totalRecords, nil
}

// loadCheckpoint loads the checkpoint from storage.
func (r *restAPILogsReceiver) loadCheckpoint(ctx context.Context) {
	bytes, err := r.storageClient.Get(ctx, checkpointStorageKey)
	if err != nil {
		r.logger.Info("unable to load checkpoint, starting fresh", zap.Error(err))
		return
	}

	if bytes == nil {
		return
	}

	var checkpoint checkpointData
	if err := jsoniter.Unmarshal(bytes, &checkpoint); err != nil {
		r.logger.Warn("unable to decode checkpoint, starting fresh", zap.Error(err))
		return
	}

	if checkpoint.PaginationState != nil {
		r.paginationState = checkpoint.PaginationState
	}
}

// saveCheckpoint saves the checkpoint to storage.
func (r *restAPILogsReceiver) saveCheckpoint(ctx context.Context) error {
	if r.storageClient == nil || r.paginationState == nil {
		return nil // No storage client or pagination state, nothing to save
	}

	checkpoint := checkpointData{
		PaginationState: r.paginationState,
	}

	bytes, err := jsoniter.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return r.storageClient.Set(ctx, checkpointStorageKey, bytes)
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
	settings      component.TelemetrySettings
	logger        *zap.Logger
	consumer      consumer.Metrics
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

// newRESTAPIMetricsReceiver creates a new REST API metrics receiver.
func newRESTAPIMetricsReceiver(
	params receiver.Settings,
	cfg *Config,
	consumer consumer.Metrics,
) (*restAPIMetricsReceiver, error) {
	return &restAPIMetricsReceiver{
		settings: params.TelemetrySettings,
		logger:   params.Logger,
		consumer: consumer,
		cfg:      cfg,
		id:       params.ID,
	}, nil
}

// Start starts the receiver.
func (r *restAPIMetricsReceiver) Start(ctx context.Context, host component.Host) error {
	// Create HTTP client
	client, err := newRESTAPIClient(ctx, r.settings, r.cfg, host)
	if err != nil {
		return fmt.Errorf("failed to create REST API client: %w", err)
	}
	r.client = client

	// Initialize storage client if configured
	if r.cfg.StorageID != nil {
		storageClient, err := adapter.GetStorageClient(ctx, host, r.cfg.StorageID, r.id)
		if err != nil {
			return fmt.Errorf("failed to get storage client: %w", err)
		}
		r.storageClient = storageClient
		r.loadCheckpoint(ctx)
	} else {
		r.storageClient = storage.NewNopClient()
		r.loadCheckpoint(ctx) // Still load checkpoint even with nop client
	}

	// Initialize pagination state
	r.paginationState = newPaginationState(r.cfg)
	// Set limit if configured
	if r.cfg.Pagination.Mode == paginationModeOffsetLimit && r.cfg.Pagination.OffsetLimit.LimitFieldName != "" {
		// Use a reasonable default limit (can be overridden by API)
		r.paginationState.limit = 10
	}

	// Start polling
	cancelCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	return r.startPolling(cancelCtx)
}

// Shutdown stops the receiver.
func (r *restAPIMetricsReceiver) Shutdown(ctx context.Context) error {
	r.logger.Debug("shutting down REST API metrics receiver")
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()

	if r.storageClient != nil {
		if err := r.saveCheckpoint(ctx); err != nil {
			r.logger.Error("failed to save checkpoint", zap.Error(err))
		}
		return r.storageClient.Close(ctx)
	}

	return nil
}

// startPolling starts the polling goroutine.
func (r *restAPIMetricsReceiver) startPolling(ctx context.Context) error {
	// Initialize with minimum poll interval for responsive startup
	r.currentPollInterval = minPollInterval

	// Run immediately on startup
	recordCount, err := r.poll(ctx)
	if err != nil {
		r.logger.Error("error on initial poll", zap.Error(err))
		// Continue with periodic polling even if initial poll fails
	}
	r.adjustPollInterval(recordCount)

	// Start periodic polling with adaptive timer
	timer := time.NewTimer(r.currentPollInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				recordCount, err := r.poll(ctx)
				if err != nil {
					r.logger.Error("error while polling", zap.Error(err))
				}
				r.adjustPollInterval(recordCount)
				timer.Reset(r.currentPollInterval)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// adjustPollInterval adjusts the poll interval based on whether data was received.
func (r *restAPIMetricsReceiver) adjustPollInterval(recordCount int) {
	if recordCount > 0 {
		// Data received - reset to minimum interval for responsive polling
		if r.currentPollInterval != minPollInterval {
			r.logger.Debug("resetting poll interval after receiving data",
				zap.Duration("new_interval", minPollInterval),
				zap.Duration("previous_interval", r.currentPollInterval))
			r.currentPollInterval = minPollInterval
		}
	} else {
		// No data - increase interval (backoff) up to max
		newInterval := time.Duration(float64(r.currentPollInterval) * backoffMultiplier)
		if newInterval > r.cfg.MaxPollInterval {
			newInterval = r.cfg.MaxPollInterval
		}
		if newInterval != r.currentPollInterval {
			r.logger.Debug("increasing poll interval due to no data",
				zap.Duration("new_interval", newInterval),
				zap.Duration("previous_interval", r.currentPollInterval))
			r.currentPollInterval = newInterval
		}
	}
}

// poll performs a single polling cycle. Returns the total number of records received.
func (r *restAPIMetricsReceiver) poll(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	totalRecords := 0

	// Build request URL with pagination
	requestURL := r.cfg.URL
	params := url.Values{}

	// Add pagination parameters
	paginationParams := buildPaginationParams(r.cfg, r.paginationState)
	for key, values := range paginationParams {
		for _, value := range values {
			params.Add(key, value)
		}
	}

	// Handle pagination - fetch all pages in this poll cycle
	for {
		// Get full response to parse pagination info and extract data
		fullResponse, err := r.client.GetFullResponse(ctx, requestURL, params)
		if err != nil {
			return totalRecords, fmt.Errorf("failed to get full response: %w", err)
		}

		// Extract data array from response
		data := extractDataFromResponse(fullResponse, r.cfg.ResponseField, r.logger)

		// Convert to metrics
		metrics := convertJSONToMetrics(data, &r.cfg.Metrics, r.logger)

		// Consume metrics if any
		if metrics.MetricCount() > 0 {
			totalRecords += metrics.MetricCount()
			if err := r.consumer.ConsumeMetrics(ctx, metrics); err != nil {
				return totalRecords, fmt.Errorf("failed to consume metrics: %w", err)
			}
		}

		// Check pagination
		if r.cfg.Pagination.Mode == paginationModeNone {
			// No pagination, we're done
			break
		}

		// Parse pagination response to check if there are more pages
		// Pass the already-extracted data array to avoid re-extraction inconsistencies
		hasMore, err := parsePaginationResponse(r.cfg, fullResponse, data, r.paginationState)
		if err != nil {
			r.logger.Warn("failed to parse pagination response", zap.Error(err))
			break
		}

		// Check page limit
		if !checkPageLimit(r.cfg, r.paginationState) {
			r.logger.Debug("page limit reached, stopping pagination")
			break
		}

		if !hasMore {
			// No more pages
			break
		}

		// Update pagination state for next page
		// For offset/limit, use actual data count instead of limit
		if r.cfg.Pagination.Mode == paginationModeOffsetLimit {
			dataCount := getDataCount(fullResponse)
			if dataCount > 0 {
				r.paginationState.currentOffset += dataCount
				r.paginationState.pagesFetched++
			} else {
				// Fallback to using limit
				updatePaginationState(r.cfg, r.paginationState)
			}
		} else {
			updatePaginationState(r.cfg, r.paginationState)
		}

		// Rebuild params with new pagination state
		params = url.Values{}
		paginationParams := buildPaginationParams(r.cfg, r.paginationState)
		for key, values := range paginationParams {
			for _, value := range values {
				params.Add(key, value)
			}
		}
	}

	// Reset timestamp after completing poll cycle
	// This ensures next poll starts fresh from the initial timestamp
	if r.cfg.Pagination.Mode == paginationModeTimestamp {
		if !r.cfg.Pagination.Timestamp.InitialTimestamp.IsZero() {
			r.paginationState.currentTimestamp = r.cfg.Pagination.Timestamp.InitialTimestamp
		} else {
			r.paginationState.currentTimestamp = time.Time{}
		}
		r.paginationState.pagesFetched = 0
	}

	return totalRecords, nil
}

// loadCheckpoint loads the checkpoint from storage.
func (r *restAPIMetricsReceiver) loadCheckpoint(ctx context.Context) {
	bytes, err := r.storageClient.Get(ctx, checkpointStorageKey)
	if err != nil {
		r.logger.Info("unable to load checkpoint, starting fresh", zap.Error(err))
		return
	}

	if bytes == nil {
		return
	}

	var checkpoint checkpointData
	if err := jsoniter.Unmarshal(bytes, &checkpoint); err != nil {
		r.logger.Warn("unable to decode checkpoint, starting fresh", zap.Error(err))
		return
	}

	if checkpoint.PaginationState != nil {
		r.paginationState = checkpoint.PaginationState
	}
}

// saveCheckpoint saves the checkpoint to storage.
func (r *restAPIMetricsReceiver) saveCheckpoint(ctx context.Context) error {
	if r.storageClient == nil || r.paginationState == nil {
		return nil // No storage client or pagination state, nothing to save
	}

	checkpoint := checkpointData{
		PaginationState: r.paginationState,
	}

	bytes, err := jsoniter.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return r.storageClient.Set(ctx, checkpointStorageKey, bytes)
}
