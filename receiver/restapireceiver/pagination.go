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
)

// paginationState tracks the current state of pagination.
type paginationState struct {
	// For offset/limit pagination
	currentOffset int
	limit         int

	// For page/size pagination
	currentPage int
	pageSize    int

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
	case PaginationModeOffsetLimit:
		state.currentOffset = cfg.Pagination.OffsetLimit.StartingOffset
		// Use a default limit - this will be sent as a query parameter
		// The actual page size may differ based on API response
		state.limit = 10

	case PaginationModePageSize:
		if cfg.Pagination.ZeroBasedIndex {
			state.currentPage = cfg.Pagination.PageSize.StartingPage
		} else {
			state.currentPage = cfg.Pagination.PageSize.StartingPage
		}
		if cfg.Pagination.PageSize.PageSizeFieldName != "" {
			// Use a default page size if not specified
			state.pageSize = 20
		}
	}

	return state
}

// buildPaginationParams builds query parameters for pagination based on the current state.
func buildPaginationParams(cfg *Config, state *paginationState) url.Values {
	params := url.Values{}

	switch cfg.Pagination.Mode {
	case PaginationModeOffsetLimit:
		if cfg.Pagination.OffsetLimit.OffsetFieldName != "" {
			params.Set(cfg.Pagination.OffsetLimit.OffsetFieldName, fmt.Sprintf("%d", state.currentOffset))
		}
		if cfg.Pagination.OffsetLimit.LimitFieldName != "" {
			params.Set(cfg.Pagination.OffsetLimit.LimitFieldName, fmt.Sprintf("%d", state.limit))
		}

	case PaginationModePageSize:
		if cfg.Pagination.PageSize.PageNumFieldName != "" {
			params.Set(cfg.Pagination.PageSize.PageNumFieldName, fmt.Sprintf("%d", state.currentPage))
		}
		if cfg.Pagination.PageSize.PageSizeFieldName != "" {
			params.Set(cfg.Pagination.PageSize.PageSizeFieldName, fmt.Sprintf("%d", state.pageSize))
		}

	case PaginationModeNone:
		// No pagination parameters
	}

	return params
}

// parsePaginationResponse parses the pagination response to determine if there are more pages.
// It also updates the state with metadata from the response.
func parsePaginationResponse(cfg *Config, response any, state *paginationState) (bool, error) {
	switch cfg.Pagination.Mode {
	case PaginationModeOffsetLimit:
		return parseOffsetLimitResponse(cfg, response, state)

	case PaginationModePageSize:
		return parsePageSizeResponse(cfg, response, state)

	case PaginationModeNone:
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
	case PaginationModeOffsetLimit:
		state.currentOffset += state.limit
		state.pagesFetched++

	case PaginationModePageSize:
		if cfg.Pagination.ZeroBasedIndex {
			state.currentPage++
		} else {
			state.currentPage++
		}
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
