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
	CurrentOffset int `json:"current_offset,omitempty"`
	Limit         int `json:"limit,omitempty"`

	// For page/size pagination
	CurrentPage int `json:"current_page,omitempty"`
	PageSize    int `json:"page_size,omitempty"`

	// For timestamp-based pagination
	CurrentTimestamp  time.Time `json:"current_timestamp,omitempty"`
	TimestampFromData bool      `json:"timestamp_from_data,omitempty"` // true if CurrentTimestamp was set from response data (vs initial config)

	// Metadata
	TotalRecords int `json:"total_records,omitempty"`
	TotalPages   int `json:"total_pages,omitempty"`
	PagesFetched int `json:"pages_fetched,omitempty"`
}

// newPaginationState creates a new pagination state based on the configuration.
func newPaginationState(cfg *Config) *paginationState {
	state := &paginationState{
		Limit:    10, // default limit
		PageSize: 20, // default page size
	}

	switch cfg.Pagination.Mode {
	case paginationModeOffsetLimit:
		state.CurrentOffset = cfg.Pagination.OffsetLimit.StartingOffset
		// Use a default limit - this will be sent as a query parameter
		// The actual page size may differ based on API response
		state.Limit = 10

	case paginationModePageSize:
		if cfg.Pagination.ZeroBasedIndex {
			state.CurrentPage = cfg.Pagination.PageSize.StartingPage
		} else {
			state.CurrentPage = cfg.Pagination.PageSize.StartingPage
		}
		if cfg.Pagination.PageSize.PageSizeFieldName != "" {
			// Use a default page size if not specified
			state.PageSize = 20
		}

	case paginationModeTimestamp:
		// Set initial timestamp if provided, otherwise start from zero time.
		// Config validation ensures the timestamp is parseable.
		if cfg.Pagination.Timestamp.InitialTimestamp != "" {
			// First try the user's configured format (they likely copied the timestamp from the API)
			if cfg.Pagination.Timestamp.TimestampFormat != "" {
				if t, err := time.Parse(cfg.Pagination.Timestamp.TimestampFormat, cfg.Pagination.Timestamp.InitialTimestamp); err == nil {
					state.CurrentTimestamp = t
				}
			}
			// Fall back to RFC3339 (the default format)
			if state.CurrentTimestamp.IsZero() {
				if t, err := time.Parse(time.RFC3339, cfg.Pagination.Timestamp.InitialTimestamp); err == nil {
					state.CurrentTimestamp = t
				}
			}
		}
		if cfg.Pagination.Timestamp.PageSize > 0 {
			state.PageSize = cfg.Pagination.Timestamp.PageSize
		} else {
			state.PageSize = 100 // Default page size for timestamp pagination
		}
	}

	// Set limit if configured for offset/limit pagination
	if cfg.Pagination.Mode == paginationModeOffsetLimit &&
		cfg.Pagination.OffsetLimit.LimitFieldName != "" {
		state.Limit = 10 // reasonable default
	}

	return state
}

// buildPaginationParams builds query parameters for pagination based on the current state.
func buildPaginationParams(cfg *Config, state *paginationState) url.Values {
	params := url.Values{}

	switch cfg.Pagination.Mode {
	case paginationModeOffsetLimit:
		if cfg.Pagination.OffsetLimit.OffsetFieldName != "" {
			params.Set(cfg.Pagination.OffsetLimit.OffsetFieldName, fmt.Sprintf("%d", state.CurrentOffset))
		}
		if cfg.Pagination.OffsetLimit.LimitFieldName != "" {
			params.Set(cfg.Pagination.OffsetLimit.LimitFieldName, fmt.Sprintf("%d", state.Limit))
		}

	case paginationModePageSize:
		if cfg.Pagination.PageSize.PageNumFieldName != "" {
			params.Set(cfg.Pagination.PageSize.PageNumFieldName, fmt.Sprintf("%d", state.CurrentPage))
		}
		if cfg.Pagination.PageSize.PageSizeFieldName != "" {
			params.Set(cfg.Pagination.PageSize.PageSizeFieldName, fmt.Sprintf("%d", state.PageSize))
		}

	case paginationModeTimestamp:
		// Add page size parameter
		if cfg.Pagination.Timestamp.PageSizeFieldName != "" {
			params.Set(cfg.Pagination.Timestamp.PageSizeFieldName, fmt.Sprintf("%d", state.PageSize))
		}
		// Add timestamp parameter if we have one
		if !state.CurrentTimestamp.IsZero() {
			if cfg.Pagination.Timestamp.ParamName != "" {
				timestampForRequest := state.CurrentTimestamp
				// Check if we should add an offset to avoid re-fetching the same record.
				// We add the offset when:
				// 1. pagesFetched > 0: Within a poll cycle, after the first page
				// 2. timestampFromData is true: The timestamp came from response data (not initial config),
				//    meaning we've already fetched records up to this timestamp in a previous cycle
				if state.PagesFetched > 0 || state.TimestampFromData {
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
					state.TotalRecords = int(total)
				} else if total, ok := totalVal.(int); ok {
					state.TotalRecords = total
				}
			}
		}
	}

	// Determine if there are more records
	// If we have total records, compare current offset + actual items returned to total
	if state.TotalRecords > 0 {
		// Use actual data count if available, otherwise use limit
		dataCount := getDataCount(response)
		itemsProcessed := state.CurrentOffset + dataCount
		hasMore := itemsProcessed < state.TotalRecords
		return hasMore, nil
	}

	// If no total records field, check if we got a full page
	// This is a heuristic: if we got exactly 'limit' items, assume there might be more
	dataCount := getDataCount(response)
	if dataCount >= state.Limit {
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
					state.TotalPages = int(totalPages)
				} else if totalPages, ok := totalPagesVal.(int); ok {
					state.TotalPages = totalPages
				}
			}
		}
	}

	// Determine if there are more pages
	// If we have total pages, compare current page to total
	if state.TotalPages > 0 {
		hasMore := state.CurrentPage < state.TotalPages
		return hasMore, nil
	}

	// If no total pages field, check if we got a full page
	// This is a heuristic: if we got exactly 'pageSize' items, assume there might be more
	dataCount := getDataCount(response)
	if dataCount >= state.PageSize {
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
		zap.Int("page_size", state.PageSize),
		zap.Time("current_state_timestamp", state.CurrentTimestamp))

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
			zap.Time("previous_timestamp", state.CurrentTimestamp))
	}

	// If we got fewer items than pageSize, definitely no more pages
	if len(dataArray) < state.PageSize {
		logger.Debug("parseTimestampResponse: partial page received, no more pages",
			zap.Int("received", len(dataArray)),
			zap.Int("page_size", state.PageSize),
			zap.Time("max_timestamp", maxTimestamp),
			zap.Time("old_timestamp", state.CurrentTimestamp))
		if !maxTimestamp.IsZero() && maxTimestamp.After(state.CurrentTimestamp) {
			state.CurrentTimestamp = maxTimestamp
			state.TimestampFromData = true // Mark that timestamp came from response data
		}
		return false, nil
	}

	// If we got exactly pageSize items, there might be more
	// However, only continue if we successfully extracted a timestamp
	if !maxTimestamp.IsZero() {
		logger.Debug("parseTimestampResponse: full page received, more pages likely",
			zap.Int("received", len(dataArray)),
			zap.Time("max_timestamp", maxTimestamp),
			zap.Time("old_timestamp", state.CurrentTimestamp))
		if maxTimestamp.After(state.CurrentTimestamp) {
			state.CurrentTimestamp = maxTimestamp
			state.TimestampFromData = true // Mark that timestamp came from response data
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
		state.CurrentOffset += state.Limit
		state.PagesFetched++

	case paginationModePageSize:
		state.CurrentPage++
		state.PagesFetched++

	case paginationModeTimestamp:
		// Timestamp is updated in parseTimestampResponse
		state.PagesFetched++
	}
}

// checkPageLimit checks if the page limit has been reached.
func checkPageLimit(cfg *Config, state *paginationState) bool {
	if cfg.Pagination.PageLimit == 0 {
		return true // No limit
	}

	return state.PagesFetched < cfg.Pagination.PageLimit
}
