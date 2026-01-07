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
	"fmt"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// paginationState tracks the current state of pagination.
type paginationState struct {
	// For offset/limit pagination
	currentOffset int
	limit         int

	// For page/size pagination
	currentPage int
	pageSize    int

	// For timestamp-based pagination
	currentTimestamp time.Time

	// Metadata
	totalRecords int
	totalPages   int
	pagesFetched int
}

// newPaginationState creates a new pagination state based on the configuration.
func newPaginationState(cfg *Config) *paginationState {
	state := &paginationState{
		limit:    10, // default limit
		pageSize: 20, // default page size
	}

	switch cfg.Pagination.Mode {
	case paginationModeOffsetLimit:
		state.currentOffset = cfg.Pagination.OffsetLimit.StartingOffset
		// Use a default limit - this will be sent as a query parameter
		// The actual page size may differ based on API response
		state.limit = 10

	case paginationModePageSize:
		if cfg.Pagination.ZeroBasedIndex {
			state.currentPage = cfg.Pagination.PageSize.StartingPage
		} else {
			state.currentPage = cfg.Pagination.PageSize.StartingPage
		}
		if cfg.Pagination.PageSize.PageSizeFieldName != "" {
			// Use a default page size if not specified
			state.pageSize = 20
		}

	case paginationModeTimestamp:
		// Set initial timestamp if provided, otherwise start from zero time
		if !cfg.Pagination.Timestamp.InitialTimestamp.IsZero() {
			state.currentTimestamp = cfg.Pagination.Timestamp.InitialTimestamp
		}
		if cfg.Pagination.Timestamp.PageSize > 0 {
			state.pageSize = cfg.Pagination.Timestamp.PageSize
		} else {
			state.pageSize = 100 // Default page size for timestamp pagination
		}
	}

	// Set limit if configured for offset/limit pagination
	if cfg.Pagination.Mode == paginationModeOffsetLimit &&
		cfg.Pagination.OffsetLimit.LimitFieldName != "" {
		state.limit = 10 // reasonable default
	}

	return state
}

// buildPaginationParams builds query parameters for pagination based on the current state.
func buildPaginationParams(cfg *Config, state *paginationState) url.Values {
	params := url.Values{}

	switch cfg.Pagination.Mode {
	case paginationModeOffsetLimit:
		if cfg.Pagination.OffsetLimit.OffsetFieldName != "" {
			params.Set(cfg.Pagination.OffsetLimit.OffsetFieldName, fmt.Sprintf("%d", state.currentOffset))
		}
		if cfg.Pagination.OffsetLimit.LimitFieldName != "" {
			params.Set(cfg.Pagination.OffsetLimit.LimitFieldName, fmt.Sprintf("%d", state.limit))
		}

	case paginationModePageSize:
		if cfg.Pagination.PageSize.PageNumFieldName != "" {
			params.Set(cfg.Pagination.PageSize.PageNumFieldName, fmt.Sprintf("%d", state.currentPage))
		}
		if cfg.Pagination.PageSize.PageSizeFieldName != "" {
			params.Set(cfg.Pagination.PageSize.PageSizeFieldName, fmt.Sprintf("%d", state.pageSize))
		}

	case paginationModeTimestamp:
		// Add page size parameter
		if cfg.Pagination.Timestamp.PageSizeFieldName != "" {
			params.Set(cfg.Pagination.Timestamp.PageSizeFieldName, fmt.Sprintf("%d", state.pageSize))
		}
		// Add timestamp parameter if we have one
		if !state.currentTimestamp.IsZero() {
			if cfg.Pagination.Timestamp.ParamName != "" {
				timestampForRequest := state.currentTimestamp
				// Only add microsecond offset on subsequent requests (after first page fetch)
				// to avoid re-fetching the last item. On the first request, use the exact
				// initial_timestamp to ensure we don't miss the first record.
				if state.pagesFetched > 0 {
					// Increment by 1 microsecond to ensure we get items strictly after this timestamp.
					// We use microsecond (not nanosecond) because most timestamp formats only preserve
					// microsecond precision, so adding 1 nanosecond wouldn't change the formatted value.
					timestampForRequest = timestampForRequest.Add(time.Microsecond)
				}
				// Use configured format or default to RFC3339
				format := cfg.Pagination.Timestamp.TimestampFormat
				if format == "" {
					format = time.RFC3339
				}
				params.Set(cfg.Pagination.Timestamp.ParamName, timestampForRequest.Format(format))
			}
		}

	case paginationModeNone:
		// No pagination parameters
	}

	return params
}

// parsePaginationResponse parses the pagination response to determine if there are more pages.
// It also updates the state with metadata from the response.
// The extractedData parameter contains the already-extracted data array from extractDataFromResponse.
func parsePaginationResponse(cfg *Config, response any, extractedData []map[string]any, state *paginationState, logger *zap.Logger) (bool, error) {
	switch cfg.Pagination.Mode {
	case paginationModeOffsetLimit:
		return parseOffsetLimitResponse(cfg, response, state)

	case paginationModePageSize:
		return parsePageSizeResponse(cfg, response, state)

	case paginationModeTimestamp:
		return parseTimestampResponse(cfg, extractedData, state, logger)

	case paginationModeNone:
		return false, nil

	default:
		return false, fmt.Errorf("unsupported pagination mode: %s", cfg.Pagination.Mode)
	}
}

// parseOffsetLimitResponse parses the response for offset/limit pagination.
func parseOffsetLimitResponse(cfg *Config, response any, state *paginationState) (bool, error) {
	// Try to extract total record count if configured
	if cfg.Pagination.TotalRecordCountField != "" {
		if responseMap, ok := response.(map[string]any); ok {
			if totalVal, exists := responseMap[cfg.Pagination.TotalRecordCountField]; exists {
				if total, ok := totalVal.(float64); ok {
					state.totalRecords = int(total)
				} else if total, ok := totalVal.(int); ok {
					state.totalRecords = total
				}
			}
		}
	}

	// Determine if there are more records
	// If we have total records, compare current offset + actual items returned to total
	if state.totalRecords > 0 {
		// Use actual data count if available, otherwise use limit
		dataCount := getDataCount(response)
		itemsProcessed := state.currentOffset + dataCount
		hasMore := itemsProcessed < state.totalRecords
		return hasMore, nil
	}

	// If no total records field, check if we got a full page
	// This is a heuristic: if we got exactly 'limit' items, assume there might be more
	dataCount := getDataCount(response)
	if dataCount >= state.limit {
		return true, nil // Full page, assume more
	}

	return false, nil // Partial page, no more
}

// parsePageSizeResponse parses the response for page/size pagination.
func parsePageSizeResponse(cfg *Config, response any, state *paginationState) (bool, error) {
	// Try to extract total pages if configured
	if cfg.Pagination.PageSize.TotalPagesFieldName != "" {
		if responseMap, ok := response.(map[string]any); ok {
			if totalPagesVal, exists := responseMap[cfg.Pagination.PageSize.TotalPagesFieldName]; exists {
				if totalPages, ok := totalPagesVal.(float64); ok {
					state.totalPages = int(totalPages)
				} else if totalPages, ok := totalPagesVal.(int); ok {
					state.totalPages = totalPages
				}
			}
		}
	}

	// Determine if there are more pages
	// If we have total pages, compare current page to total
	if state.totalPages > 0 {
		hasMore := state.currentPage < state.totalPages
		return hasMore, nil
	}

	// If no total pages field, check if we got a full page
	// This is a heuristic: if we got exactly 'pageSize' items, assume there might be more
	dataCount := getDataCount(response)
	if dataCount >= state.pageSize {
		return true, nil // Full page, assume more
	}

	return false, nil // Partial page, no more
}

// Common timestamp formats to try when parsing response timestamps
var timestampFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05.000000-07:00", // RFC3339 with microseconds
	"2006-01-02T15:04:05.000000Z",      // RFC3339 with microseconds, UTC
	"2006-01-02T15:04:05-07:00",        // RFC3339 without fractional seconds
	"2006-01-02 15:04:05.000000-07:00", // Space separator with microseconds
	"2006-01-02 15:04:05-07:00",        // Space separator without fractional
	"2006-01-02 15:04:05",              // Simple datetime
	"2006-01-02",                       // Date only
}

// parseTimestampResponse parses the response for timestamp-based pagination.
// The dataArray parameter contains the already-extracted data from extractDataFromResponse.
func parseTimestampResponse(cfg *Config, dataArray []map[string]any, state *paginationState, logger *zap.Logger) (bool, error) {
	// If no data, no more pages
	if len(dataArray) == 0 {
		logger.Debug("parseTimestampResponse: no data in response, no more pages")
		return false, nil
	}

	logger.Debug("parseTimestampResponse: processing response",
		zap.Int("data_count", len(dataArray)),
		zap.Int("page_size", state.pageSize),
		zap.Time("current_state_timestamp", state.currentTimestamp))

	// Find the maximum timestamp across ALL items in the response.
	// This is critical because APIs may return data in any order (often descending/newest first).
	// We need to track the newest timestamp seen to avoid re-fetching the same data.
	var maxTimestamp time.Time
	timestampField := cfg.Pagination.Timestamp.TimestampFieldName

	if timestampField != "" {
		for i, item := range dataArray {
			if timestampVal, exists := item[timestampField]; exists {
				parsedTime := parseTimestampValue(timestampVal)
				if !parsedTime.IsZero() && parsedTime.After(maxTimestamp) {
					maxTimestamp = parsedTime
					logger.Debug("parseTimestampResponse: found newer timestamp",
						zap.Int("item_index", i),
						zap.Time("timestamp", parsedTime))
				}
			}
		}

		logger.Debug("parseTimestampResponse: scanned all items for max timestamp",
			zap.Int("item_count", len(dataArray)),
			zap.Time("max_timestamp_found", maxTimestamp),
			zap.Time("previous_timestamp", state.currentTimestamp))
	}

	// If we got fewer items than pageSize, definitely no more pages
	if len(dataArray) < state.pageSize {
		logger.Debug("parseTimestampResponse: partial page received, no more pages",
			zap.Int("received", len(dataArray)),
			zap.Int("page_size", state.pageSize),
			zap.Time("max_timestamp", maxTimestamp),
			zap.Time("old_timestamp", state.currentTimestamp))
		if !maxTimestamp.IsZero() && maxTimestamp.After(state.currentTimestamp) {
			state.currentTimestamp = maxTimestamp
		}
		return false, nil
	}

	// If we got exactly pageSize items, there might be more
	// However, only continue if we successfully extracted a timestamp
	if !maxTimestamp.IsZero() {
		logger.Debug("parseTimestampResponse: full page received, more pages likely",
			zap.Int("received", len(dataArray)),
			zap.Time("max_timestamp", maxTimestamp),
			zap.Time("old_timestamp", state.currentTimestamp))
		if maxTimestamp.After(state.currentTimestamp) {
			state.currentTimestamp = maxTimestamp
		}
		return true, nil
	}

	// Got a full page but couldn't extract timestamp
	// This is unusual - could indicate data structure issue
	// To be safe and avoid infinite loops, we'll stop here
	logger.Debug("parseTimestampResponse: full page but no timestamp extracted, stopping")
	return false, fmt.Errorf("received full page (%d items) but failed to extract timestamp from any item", len(dataArray))
}

// parseTimestampValue parses a timestamp value from various formats.
func parseTimestampValue(timestampVal any) time.Time {
	var parsedTime time.Time

	if timestampStr, ok := timestampVal.(string); ok {
		// Try multiple timestamp formats
		for _, format := range timestampFormats {
			if t, err := time.Parse(format, timestampStr); err == nil {
				parsedTime = t
				break
			}
		}
	} else if timestampFloat, ok := timestampVal.(float64); ok {
		// Unix timestamp (seconds or milliseconds)
		if timestampFloat > 1e10 {
			// Likely milliseconds
			parsedTime = time.Unix(0, int64(timestampFloat*1e6))
		} else {
			// Likely seconds
			parsedTime = time.Unix(int64(timestampFloat), 0)
		}
	} else if timestampInt, ok := timestampVal.(int64); ok {
		if timestampInt > 1e10 {
			// Likely milliseconds
			parsedTime = time.Unix(0, timestampInt*1e6)
		} else {
			// Likely seconds
			parsedTime = time.Unix(timestampInt, 0)
		}
	}

	return parsedTime
}

// getDataCount extracts the count of data items from the response.
func getDataCount(response any) int {
	// If response is directly an array
	if arr, ok := response.([]any); ok {
		return len(arr)
	}

	// If response is a map, try to find a data field
	if responseMap, ok := response.(map[string]any); ok {
		// Try common field names
		for _, fieldName := range []string{"data", "items", "results", "records"} {
			if dataVal, exists := responseMap[fieldName]; exists {
				// Try []any first
				if arr, ok := dataVal.([]any); ok {
					return len(arr)
				}
				// Try []map[string]any (common in JSON responses)
				if arr, ok := dataVal.([]map[string]any); ok {
					return len(arr)
				}
				// Try to convert interface{} slice
				if arr, ok := dataVal.([]interface{}); ok {
					return len(arr)
				}
			}
		}
	}

	return 0
}

// updatePaginationState updates the pagination state to the next page/offset.
func updatePaginationState(cfg *Config, state *paginationState) {
	switch cfg.Pagination.Mode {
	case paginationModeOffsetLimit:
		state.currentOffset += state.limit
		state.pagesFetched++

	case paginationModePageSize:
		state.currentPage++
		state.pagesFetched++

	case paginationModeTimestamp:
		// Timestamp is updated in parseTimestampResponse
		state.pagesFetched++
	}
}

// checkPageLimit checks if the page limit has been reached.
func checkPageLimit(cfg *Config, state *paginationState) bool {
	if cfg.Pagination.PageLimit == 0 {
		return true // No limit
	}

	return state.pagesFetched < cfg.Pagination.PageLimit
}
