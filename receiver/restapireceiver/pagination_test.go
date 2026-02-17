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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBuildPaginationParams_OffsetLimit(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				OffsetFieldName: "offset",
				LimitFieldName:  "limit",
				StartingOffset:  0,
			},
		},
	}

	state := &paginationState{
		CurrentOffset: 0,
		Limit:         10,
	}

	params := buildPaginationParams(cfg, state)
	require.Equal(t, "0", params.Get("offset"))
	require.Equal(t, "10", params.Get("limit"))
}

func TestBuildPaginationParams_OffsetLimit_NonZero(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				OffsetFieldName: "skip",
				LimitFieldName:  "take",
				StartingOffset:  0,
			},
		},
	}

	state := &paginationState{
		CurrentOffset: 20,
		Limit:         50,
	}

	params := buildPaginationParams(cfg, state)
	require.Equal(t, "20", params.Get("skip"))
	require.Equal(t, "50", params.Get("take"))
}

func TestBuildPaginationParams_PageSize_OneBased(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:           paginationModePageSize,
			ZeroBasedIndex: false,
			PageSize: PageSizePagination{
				PageNumFieldName:  "page",
				PageSizeFieldName: "size",
				StartingPage:      1,
			},
		},
	}

	state := &paginationState{
		CurrentPage: 1,
		PageSize:    20,
	}

	params := buildPaginationParams(cfg, state)
	require.Equal(t, "1", params.Get("page"))
	require.Equal(t, "20", params.Get("size"))
}

func TestBuildPaginationParams_PageSize_ZeroBased(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:           paginationModePageSize,
			ZeroBasedIndex: true,
			PageSize: PageSizePagination{
				PageNumFieldName:  "page",
				PageSizeFieldName: "size",
				StartingPage:      0,
			},
		},
	}

	state := &paginationState{
		CurrentPage: 0,
		PageSize:    20,
	}

	params := buildPaginationParams(cfg, state)
	require.Equal(t, "0", params.Get("page"))
	require.Equal(t, "20", params.Get("size"))
}

func TestBuildPaginationParams_None(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeNone,
		},
	}

	state := &paginationState{}
	params := buildPaginationParams(cfg, state)
	require.Empty(t, params)
}

func TestParsePaginationResponse_OffsetLimit_HasMore(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				OffsetFieldName: "offset",
				LimitFieldName:  "limit",
			},
			TotalRecordCountField: "total",
		},
	}

	// Response with 25 total records, we're at offset 0 with limit 10
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"}, {"id": "3"}, {"id": "4"}, {"id": "5"},
			{"id": "6"}, {"id": "7"}, {"id": "8"}, {"id": "9"}, {"id": "10"},
		},
		"total": 25,
	}

	state := &paginationState{
		CurrentOffset: 0,
		Limit:         10,
	}

	// Extract data for pagination parsing (not used for offset/limit mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.True(t, hasMore)
	require.Equal(t, 25, state.TotalRecords)
}

func TestParsePaginationResponse_OffsetLimit_NoMore(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				OffsetFieldName: "offset",
				LimitFieldName:  "limit",
			},
			TotalRecordCountField: "total",
		},
	}

	// Response with 5 total records, we're at offset 0 with limit 10
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"}, {"id": "3"}, {"id": "4"}, {"id": "5"},
		},
		"total": 5,
	}

	state := &paginationState{
		CurrentOffset: 0,
		Limit:         10,
	}

	// Extract data for pagination parsing (not used for offset/limit mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Equal(t, 5, state.TotalRecords)
}

func TestParsePaginationResponse_OffsetLimit_NoTotalField(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				OffsetFieldName: "offset",
				LimitFieldName:  "limit",
			},
		},
	}

	// Response without total field - assume has more if we got a full page
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"}, {"id": "3"}, {"id": "4"}, {"id": "5"},
			{"id": "6"}, {"id": "7"}, {"id": "8"}, {"id": "9"}, {"id": "10"},
		},
	}

	state := &paginationState{
		CurrentOffset: 0,
		Limit:         10,
	}

	// Extract data for pagination parsing (not used for offset/limit mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.True(t, hasMore) // Full page, assume more
}

func TestParsePaginationResponse_OffsetLimit_PartialPage(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				OffsetFieldName: "offset",
				LimitFieldName:  "limit",
			},
		},
	}

	// Response with only 3 items when limit is 10 - no more pages
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"}, {"id": "3"},
		},
	}

	state := &paginationState{
		CurrentOffset: 0,
		Limit:         10,
	}

	// Extract data for pagination parsing (not used for offset/limit mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.False(t, hasMore) // Partial page, no more
}

func TestParsePaginationResponse_PageSize_HasMore(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModePageSize,
			PageSize: PageSizePagination{
				PageNumFieldName:    "page",
				PageSizeFieldName:   "size",
				TotalPagesFieldName: "total_pages",
			},
		},
	}

	// Response with 5 total pages, we're on page 1
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"},
		},
		"total_pages": 5,
	}

	state := &paginationState{
		CurrentPage: 1,
		PageSize:    20,
	}

	// Extract data for pagination parsing (not used for page/size mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.True(t, hasMore)
	require.Equal(t, 5, state.TotalPages)
}

func TestParsePaginationResponse_PageSize_NoMore(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModePageSize,
			PageSize: PageSizePagination{
				PageNumFieldName:    "page",
				PageSizeFieldName:   "size",
				TotalPagesFieldName: "total_pages",
			},
		},
	}

	// Response with 1 total page, we're on page 1
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"},
		},
		"total_pages": 1,
	}

	state := &paginationState{
		CurrentPage: 1,
		PageSize:    20,
	}

	// Extract data for pagination parsing (not used for page/size mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Equal(t, 1, state.TotalPages)
}

func TestParsePaginationResponse_PageSize_NoTotalPagesField(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModePageSize,
			PageSize: PageSizePagination{
				PageNumFieldName:  "page",
				PageSizeFieldName: "size",
			},
		},
	}

	// Response without total_pages - assume has more if we got a full page
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"}, {"id": "3"}, {"id": "4"}, {"id": "5"},
		},
	}

	state := &paginationState{
		CurrentPage: 1,
		PageSize:    5,
	}

	// Extract data for pagination parsing (not used for page/size mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.True(t, hasMore) // Full page, assume more
}

func TestParsePaginationResponse_PageSize_PartialPage(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModePageSize,
			PageSize: PageSizePagination{
				PageNumFieldName:  "page",
				PageSizeFieldName: "size",
			},
		},
	}

	// Response with only 2 items when page size is 5 - no more pages
	response := map[string]any{
		"data": []map[string]any{
			{"id": "1"}, {"id": "2"},
		},
	}

	state := &paginationState{
		CurrentPage: 1,
		PageSize:    5,
	}

	// Extract data for pagination parsing (not used for page/size mode)
	data := extractDataFromResponse(response, "", nil)

	hasMore, err := parsePaginationResponse(cfg, response, data, state, zap.NewNop())
	require.NoError(t, err)
	require.False(t, hasMore) // Partial page, no more
}

func TestUpdatePaginationState_OffsetLimit(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				StartingOffset: 0,
			},
		},
	}

	state := newPaginationState(cfg)
	require.Equal(t, 0, state.CurrentOffset)
	require.Equal(t, 10, state.Limit) // default limit

	// Update after fetching a page
	state.CurrentOffset = 10
	state.Limit = 10

	// Next page should be at offset 20
	updatePaginationState(cfg, state)
	require.Equal(t, 20, state.CurrentOffset)
}

func TestUpdatePaginationState_PageSize_OneBased(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:           paginationModePageSize,
			ZeroBasedIndex: false,
			PageSize: PageSizePagination{
				StartingPage: 1,
			},
		},
	}

	state := newPaginationState(cfg)
	require.Equal(t, 1, state.CurrentPage)

	// Update after fetching page 1
	state.CurrentPage = 1

	// Next page should be 2
	updatePaginationState(cfg, state)
	require.Equal(t, 2, state.CurrentPage)
}

func TestUpdatePaginationState_PageSize_ZeroBased(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:           paginationModePageSize,
			ZeroBasedIndex: true,
			PageSize: PageSizePagination{
				StartingPage: 0,
			},
		},
	}

	state := newPaginationState(cfg)
	require.Equal(t, 0, state.CurrentPage)

	// Update after fetching page 0
	state.CurrentPage = 0

	// Next page should be 1
	updatePaginationState(cfg, state)
	require.Equal(t, 1, state.CurrentPage)
}

func TestCheckPageLimit_WithinLimit(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:      paginationModePageSize,
			PageLimit: 10,
		},
	}

	state := &paginationState{
		CurrentPage:  5,
		PagesFetched: 5,
	}

	withinLimit := checkPageLimit(cfg, state)
	require.True(t, withinLimit)
}

func TestCheckPageLimit_ExceedsLimit(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:      paginationModePageSize,
			PageLimit: 10,
		},
	}

	state := &paginationState{
		CurrentPage:  11,
		PagesFetched: 11,
	}

	withinLimit := checkPageLimit(cfg, state)
	require.False(t, withinLimit)
}

func TestCheckPageLimit_NoLimit(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:      paginationModePageSize,
			PageLimit: 0, // 0 means no limit
		},
	}

	state := &paginationState{
		CurrentPage:  100,
		PagesFetched: 100,
	}

	withinLimit := checkPageLimit(cfg, state)
	require.True(t, withinLimit)
}

func TestNewPaginationState_OffsetLimit(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
			OffsetLimit: OffsetLimitPagination{
				StartingOffset: 5,
			},
		},
	}

	state := newPaginationState(cfg)
	require.Equal(t, 5, state.CurrentOffset)
	require.Equal(t, 10, state.Limit) // default
	require.Equal(t, 0, state.CurrentPage)
}

func TestNewPaginationState_PageSize_OneBased(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:           paginationModePageSize,
			ZeroBasedIndex: false,
			PageSize: PageSizePagination{
				StartingPage: 1,
			},
		},
	}

	state := newPaginationState(cfg)
	require.Equal(t, 1, state.CurrentPage)
	require.Equal(t, 20, state.PageSize) // default
	require.Equal(t, 0, state.CurrentOffset)
}

func TestNewPaginationState_PageSize_ZeroBased(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode:           paginationModePageSize,
			ZeroBasedIndex: true,
			PageSize: PageSizePagination{
				StartingPage: 0,
			},
		},
	}

	state := newPaginationState(cfg)
	require.Equal(t, 0, state.CurrentPage)
	require.Equal(t, 20, state.PageSize) // default
	require.Equal(t, 0, state.CurrentOffset)
}

func TestParsePaginationResponse_WithDataArray(t *testing.T) {
	cfg := &Config{
		Pagination: PaginationConfig{
			Mode: paginationModeOffsetLimit,
		},
	}

	// Response where data is an array (not in a map)
	responseData := []map[string]any{
		{"id": "1"}, {"id": "2"}, {"id": "3"}, {"id": "4"}, {"id": "5"},
		{"id": "6"}, {"id": "7"}, {"id": "8"}, {"id": "9"}, {"id": "10"},
	}

	// Convert to JSON and back to simulate real response
	jsonBytes, _ := json.Marshal(responseData)
	var response any
	json.Unmarshal(jsonBytes, &response)

	state := &paginationState{
		CurrentOffset: 0,
		Limit:         10,
	}

	// Extract data for pagination parsing
	responseMap := map[string]any{"data": response}
	data := extractDataFromResponse(responseMap, "", nil)

	// When response is directly an array, we need to handle it differently
	// For now, we'll assume if we got a full page, there might be more
	hasMore, err := parsePaginationResponse(cfg, responseMap, data, state, zap.NewNop())
	require.NoError(t, err)
	require.True(t, hasMore) // Full page of 10 items
}
