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
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

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
	TimeOffset      *time.Time       `json:"time_offset,omitempty"`
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

	wg              sync.WaitGroup
	mu              sync.Mutex
	pollInterval    time.Duration
	cancel          context.CancelFunc
	paginationState *paginationState
	timeOffset      time.Time
}

// newRESTAPILogsReceiver creates a new REST API logs receiver.
func newRESTAPILogsReceiver(
	params receiver.Settings,
	cfg *Config,
	consumer consumer.Logs,
) (*restAPILogsReceiver, error) {
	return &restAPILogsReceiver{
		settings:     params.TelemetrySettings,
		logger:       params.Logger,
		consumer:     consumer,
		cfg:          cfg,
		id:           params.ID,
		pollInterval: cfg.PollInterval,
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
	if r.cfg.Pagination.Mode == PaginationModeOffsetLimit && r.cfg.Pagination.OffsetLimit.LimitFieldName != "" {
		// Use a reasonable default limit (can be overridden by API)
		r.paginationState.limit = 10
	}

	// Initialize time offset
	if r.cfg.TimeBasedOffset.Enabled {
		r.timeOffset = r.cfg.TimeBasedOffset.OffsetTimestamp
		if r.timeOffset.IsZero() {
			r.timeOffset = time.Now().Add(-r.pollInterval)
		}
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
	// Run immediately on startup
	if err := r.poll(ctx); err != nil {
		r.logger.Error("error on initial poll", zap.Error(err))
		// Continue with periodic polling even if initial poll fails
	}

	// Start periodic polling
	ticker := time.NewTicker(r.pollInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := r.poll(ctx); err != nil {
					r.logger.Error("error while polling", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// poll performs a single polling cycle.
func (r *restAPILogsReceiver) poll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Build request URL with pagination and time-based offset
	requestURL := r.cfg.URL
	params := url.Values{}

	// Add pagination parameters
	paginationParams := buildPaginationParams(r.cfg, r.paginationState)
	for key, values := range paginationParams {
		for _, value := range values {
			params.Add(key, value)
		}
	}

	// Add time-based offset parameter
	if r.cfg.TimeBasedOffset.Enabled {
		params.Set(r.cfg.TimeBasedOffset.ParamName, r.timeOffset.Format(time.RFC3339))
	}

	// Handle pagination - fetch all pages in this poll cycle
	for {
		// Get full response to parse pagination info and extract data
		fullResponse, err := r.client.GetFullResponse(ctx, requestURL, params)
		if err != nil {
			return fmt.Errorf("failed to get full response: %w", err)
		}

		// Extract data array from response
		data := extractDataFromResponse(fullResponse, r.cfg.ResponseField, r.logger)

		// Convert to logs
		logs := convertJSONToLogs(data, r.logger)

		// Consume logs if any
		if logs.LogRecordCount() > 0 {
			if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
				return fmt.Errorf("failed to consume logs: %w", err)
			}
		}

		// Check pagination
		if r.cfg.Pagination.Mode == PaginationModeNone {
			// No pagination, we're done
			break
		}

		// Parse pagination response to check if there are more pages
		hasMore, err := parsePaginationResponse(r.cfg, fullResponse, r.paginationState)
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
		if r.cfg.Pagination.Mode == PaginationModeOffsetLimit {
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

		// Re-add time-based offset if enabled
		if r.cfg.TimeBasedOffset.Enabled {
			params.Set(r.cfg.TimeBasedOffset.ParamName, r.timeOffset.Format(time.RFC3339))
		}
	}

	// Update time offset if enabled
	if r.cfg.TimeBasedOffset.Enabled {
		r.timeOffset = time.Now()
	}

	return nil
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
	if err := json.Unmarshal(bytes, &checkpoint); err != nil {
		r.logger.Warn("unable to decode checkpoint, starting fresh", zap.Error(err))
		return
	}

	if checkpoint.PaginationState != nil {
		r.paginationState = checkpoint.PaginationState
	}

	if checkpoint.TimeOffset != nil {
		r.timeOffset = *checkpoint.TimeOffset
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

	if r.cfg != nil && r.cfg.TimeBasedOffset.Enabled {
		checkpoint.TimeOffset = &r.timeOffset
	}

	bytes, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return r.storageClient.Set(ctx, checkpointStorageKey, bytes)
}

// extractDataFromResponse extracts the data array from the full response.
func extractDataFromResponse(response map[string]any, responseField string, logger *zap.Logger) []map[string]any {
	var dataArray []any

	if responseField != "" {
		// Response has a field containing the array
		fieldValue, ok := response[responseField]
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

	wg              sync.WaitGroup
	mu              sync.Mutex
	pollInterval    time.Duration
	cancel          context.CancelFunc
	paginationState *paginationState
	timeOffset      time.Time
}

// newRESTAPIMetricsReceiver creates a new REST API metrics receiver.
func newRESTAPIMetricsReceiver(
	params receiver.Settings,
	cfg *Config,
	consumer consumer.Metrics,
) (*restAPIMetricsReceiver, error) {
	return &restAPIMetricsReceiver{
		settings:     params.TelemetrySettings,
		logger:       params.Logger,
		consumer:     consumer,
		cfg:          cfg,
		id:           params.ID,
		pollInterval: cfg.PollInterval,
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
	if r.cfg.Pagination.Mode == PaginationModeOffsetLimit && r.cfg.Pagination.OffsetLimit.LimitFieldName != "" {
		// Use a reasonable default limit (can be overridden by API)
		r.paginationState.limit = 10
	}

	// Initialize time offset
	if r.cfg.TimeBasedOffset.Enabled {
		r.timeOffset = r.cfg.TimeBasedOffset.OffsetTimestamp
		if r.timeOffset.IsZero() {
			r.timeOffset = time.Now().Add(-r.pollInterval)
		}
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
	// Run immediately on startup
	if err := r.poll(ctx); err != nil {
		r.logger.Error("error on initial poll", zap.Error(err))
		// Continue with periodic polling even if initial poll fails
	}

	// Start periodic polling
	ticker := time.NewTicker(r.pollInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := r.poll(ctx); err != nil {
					r.logger.Error("error while polling", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// poll performs a single polling cycle.
func (r *restAPIMetricsReceiver) poll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Build request URL with pagination and time-based offset
	requestURL := r.cfg.URL
	params := url.Values{}

	// Add pagination parameters
	paginationParams := buildPaginationParams(r.cfg, r.paginationState)
	for key, values := range paginationParams {
		for _, value := range values {
			params.Add(key, value)
		}
	}

	// Add time-based offset parameter
	if r.cfg.TimeBasedOffset.Enabled {
		params.Set(r.cfg.TimeBasedOffset.ParamName, r.timeOffset.Format(time.RFC3339))
	}

	// Handle pagination - fetch all pages in this poll cycle
	for {
		// Get full response to parse pagination info and extract data
		fullResponse, err := r.client.GetFullResponse(ctx, requestURL, params)
		if err != nil {
			return fmt.Errorf("failed to get full response: %w", err)
		}

		// Extract data array from response
		data := extractDataFromResponse(fullResponse, r.cfg.ResponseField, r.logger)

		// Convert to metrics
		metrics := convertJSONToMetrics(data, r.logger)

		// Consume metrics if any
		if metrics.MetricCount() > 0 {
			if err := r.consumer.ConsumeMetrics(ctx, metrics); err != nil {
				return fmt.Errorf("failed to consume metrics: %w", err)
			}
		}

		// Check pagination
		if r.cfg.Pagination.Mode == PaginationModeNone {
			// No pagination, we're done
			break
		}

		// Parse pagination response to check if there are more pages
		hasMore, err := parsePaginationResponse(r.cfg, fullResponse, r.paginationState)
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
		if r.cfg.Pagination.Mode == PaginationModeOffsetLimit {
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

		// Re-add time-based offset if enabled
		if r.cfg.TimeBasedOffset.Enabled {
			params.Set(r.cfg.TimeBasedOffset.ParamName, r.timeOffset.Format(time.RFC3339))
		}
	}

	// Update time offset if enabled
	if r.cfg.TimeBasedOffset.Enabled {
		r.timeOffset = time.Now()
	}

	return nil
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
	if err := json.Unmarshal(bytes, &checkpoint); err != nil {
		r.logger.Warn("unable to decode checkpoint, starting fresh", zap.Error(err))
		return
	}

	if checkpoint.PaginationState != nil {
		r.paginationState = checkpoint.PaginationState
	}

	if checkpoint.TimeOffset != nil {
		r.timeOffset = *checkpoint.TimeOffset
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

	if r.cfg != nil && r.cfg.TimeBasedOffset.Enabled {
		checkpoint.TimeOffset = &r.timeOffset
	}

	bytes, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return r.storageClient.Set(ctx, checkpointStorageKey, bytes)
}
