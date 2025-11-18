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

package kandjireceiver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/observiq/bindplane-otel-collector/receiver/kandjireceiver/internal/metadata"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestScrapeEmptyEndpoints(t *testing.T) {
	cfg := &Config{
		EndpointParams: map[string]map[string]any{},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
	scraper := newKandjiScraper(receivertest.NewNopSettings(typ), cfg)

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	require.NotNil(t, metrics)
}

func TestScrapeUnknownEndpoint(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{}

	cfg := &Config{
		EndpointParams: map[string]map[string]any{
			"GET /unknown": {},
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
	scraper := newKandjiScraper(receivertest.NewNopSettings(typ), cfg)

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err) // Unknown endpoints are logged but don't error
	require.NotNil(t, metrics)
}

func TestScrapePaginated(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		EPAuditEventsList: {
			SupportsPagination: true,
			ResponseType:        AuditEventsResponse{},
		},
	}

	cfg := &Config{
		EndpointParams: map[string]map[string]any{
			string(EPAuditEventsList): {
				"limit": 100,
			},
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
	scraper := newKandjiScraper(receivertest.NewNopSettings(typ), cfg)

	mockClient := &mockKandjiClient{}
	scraper.client = mockClient

	// First page with next cursor
	resp1 := &AuditEventsResponse{
		Results: []AuditEvent{
			{ID: "1", Action: "test"},
			{ID: "2", Action: "test"},
		},
		Next: stringPtr("cursor123"),
	}

	// Second page without next cursor (end)
	resp2 := &AuditEventsResponse{
		Results: []AuditEvent{
			{ID: "3", Action: "test"},
		},
		Next: nil,
	}

	callCount := new(int)
	mockClient.
		On("CallAPI", mock.Anything, EPAuditEventsList, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if out, ok := args.Get(3).(*AuditEventsResponse); ok {
				if *callCount == 0 {
					*out = *resp1
				} else {
					*out = *resp2
				}
				*callCount++
			}
		}).
		Return(200, nil).Twice()
	mockClient.On("Shutdown").Return(nil)

	// Start the scraper
	host := componenttest.NewNopHost()
	err := scraper.start(context.Background(), host)
	require.NoError(t, err)

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	mockClient.AssertExpectations(t)
}

func TestScrapeSingle(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		EPAuditEventsList: {
			SupportsPagination: false,
			ResponseType:        AuditEventsResponse{},
		},
	}

	cfg := &Config{
		EndpointParams: map[string]map[string]any{
			string(EPAuditEventsList): {},
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
	scraper := newKandjiScraper(receivertest.NewNopSettings(typ), cfg)

	mockClient := &mockKandjiClient{}
	scraper.client = mockClient

	resp := &AuditEventsResponse{
		Results: []AuditEvent{
			{ID: "1", Action: "test"},
		},
	}

	mockClient.
		On("CallAPI", mock.Anything, EPAuditEventsList, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if out, ok := args.Get(3).(*AuditEventsResponse); ok {
				*out = *resp
			}
		}).
		Return(200, nil)
	mockClient.On("Shutdown").Return(nil)

	host := componenttest.NewNopHost()
	err := scraper.start(context.Background(), host)
	require.NoError(t, err)

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	mockClient.AssertExpectations(t)
}

func TestScrapeError(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		EPAuditEventsList: {
			SupportsPagination: true,
			ResponseType:        AuditEventsResponse{},
		},
	}

	cfg := &Config{
		EndpointParams: map[string]map[string]any{
			string(EPAuditEventsList): {},
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
	scraper := newKandjiScraper(receivertest.NewNopSettings(typ), cfg)

	mockClient := &mockKandjiClient{}
	scraper.client = mockClient

	mockClient.
		On("CallAPI", mock.Anything, EPAuditEventsList, mock.Anything, mock.Anything).
		Return(500, errors.New("API error"))
	mockClient.On("Shutdown").Return(nil)

	host := componenttest.NewNopHost()
	err := scraper.start(context.Background(), host)
	require.NoError(t, err)

	_, err = scraper.scrape(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "API error")

	mockClient.AssertExpectations(t)
}

func TestBuildPageParams(t *testing.T) {
	tests := []struct {
		name   string
		base   map[string]any
		cursor *string
		want   map[string]any
	}{
		{
			name:   "no cursor",
			base:   map[string]any{"limit": 100},
			cursor: nil,
			want:   map[string]any{"limit": 100},
		},
		{
			name:   "with cursor",
			base:   map[string]any{"limit": 100},
			cursor: stringPtr("abc123"),
			want:   map[string]any{"limit": 100, "cursor": "abc123"},
		},
		{
			name:   "empty cursor string",
			base:   map[string]any{"limit": 100},
			cursor: stringPtr(""),
			want:   map[string]any{"limit": 100},
		},
		{
			name:   "multiple base params",
			base:   map[string]any{"limit": 100, "sort_by": "occurred_at"},
			cursor: stringPtr("xyz789"),
			want:   map[string]any{"limit": 100, "sort_by": "occurred_at", "cursor": "xyz789"},
		},
		{
			name:   "cursor overwrites existing",
			base:   map[string]any{"limit": 100, "cursor": "old"},
			cursor: stringPtr("new"),
			want:   map[string]any{"limit": 100, "cursor": "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPageParams(tt.base, tt.cursor)
			require.Equal(t, tt.want, result)
			// Ensure base wasn't mutated
			require.Equal(t, tt.base, map[string]any{"limit": tt.base["limit"]})
		})
	}
}

func TestExtractNextCursor(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected *string
	}{
		{
			name: "with next cursor",
			input: &AuditEventsResponse{
				Next: stringPtr("cursor123"),
			},
			expected: stringPtr("cursor123"),
		},
		{
			name: "no next cursor",
			input: &AuditEventsResponse{
				Next: nil,
			},
			expected: nil,
		},
		{
			name: "empty next cursor",
			input: &AuditEventsResponse{
				Next: stringPtr(""),
			},
			expected: nil,
		},
		{
			name:     "wrong type",
			input:    "not a response",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNextCursor(tt.input)
			if tt.expected == nil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestGetRecordCount(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int64
	}{
		{
			name: "with results",
			input: &AuditEventsResponse{
				Results: []AuditEvent{
					{ID: "1"},
					{ID: "2"},
					{ID: "3"},
				},
			},
			expected: 3,
		},
		{
			name: "empty results",
			input: &AuditEventsResponse{
				Results: []AuditEvent{},
			},
			expected: 0,
		},
		{
			name:     "wrong type",
			input:    "not a response",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRecordCount(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCloneResponseType(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "AuditEventsResponse",
			input:    AuditEventsResponse{},
			expected: &AuditEventsResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cloneResponseType(tt.input)
			require.IsType(t, tt.expected, result)
		})
	}
}


