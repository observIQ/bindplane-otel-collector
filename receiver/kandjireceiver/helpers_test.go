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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateParams(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	// Set up test registry
	maxLimit := 500
	minLimit := 10
	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		EPAuditEventsList: {
			Params: []ParamSpec{
				{
					Name: "limit",
					Type: ParamInt,
					Constraints: &ParamConstraints{
						MinInt: &minLimit,
						MaxInt: &maxLimit,
					},
				},
				{
					Name:        "sort_by",
					Type:        ParamString,
					AllowedVals: []string{"occurred_at", "-occurred_at"},
				},
				{
					Name:     "cursor",
					Type:     ParamString,
					Required: false,
				},
				{
					Name:     "required_param",
					Type:     ParamString,
					Required: true,
				},
			},
		},
	}

	tests := []struct {
		name    string
		ep      KandjiEndpoint
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid params",
			ep:   EPAuditEventsList,
			params: map[string]any{
				"limit":        100,
				"sort_by":      "occurred_at",
				"cursor":       "abc123",
				"required_param": "value",
			},
			wantErr: false,
		},
		{
			name: "missing required param",
			ep:   EPAuditEventsList,
			params: map[string]any{
				"limit": 100,
			},
			wantErr: true,
			errMsg:  "missing required param",
		},
		{
			name: "invalid int type",
			ep:   EPAuditEventsList,
			params: map[string]any{
				"limit":         "not an int",
				"required_param": "value",
			},
			wantErr: true,
			errMsg:  "must be int",
		},
		{
			name: "int below min",
			ep:   EPAuditEventsList,
			params: map[string]any{
				"limit":         5,
				"required_param": "value",
			},
			wantErr: true,
			errMsg:  "must be >=",
		},
		{
			name: "int above max",
			ep:   EPAuditEventsList,
			params: map[string]any{
				"limit":         1000,
				"required_param": "value",
			},
			wantErr: true,
			errMsg:  "must be <=",
		},
		{
			name: "invalid enum value",
			ep:   EPAuditEventsList,
			params: map[string]any{
				"limit":         100,
				"sort_by":       "invalid",
				"required_param": "value",
			},
			wantErr: true,
			errMsg:  "must be one of",
		},
		{
			name: "valid enum value",
			ep:   EPAuditEventsList,
			params: map[string]any{
				"limit":         100,
				"sort_by":       "-occurred_at",
				"required_param": "value",
			},
			wantErr: false,
		},
		{
			name:    "unknown endpoint",
			ep:      KandjiEndpoint("GET /unknown"),
			params:  map[string]any{},
			wantErr: true,
			errMsg:  "unknown endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParams(tt.ep, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNormalizeCursor(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected *string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty string",
			input:    stringPtr(""),
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    stringPtr("   "),
			expected: nil,
		},
		{
			name:     "simple cursor",
			input:    stringPtr("abc123"),
			expected: stringPtr("abc123"),
		},
		{
			name:     "cursor with whitespace",
			input:    stringPtr("  abc123  "),
			expected: stringPtr("abc123"),
		},
		{
			name:     "cursor from URL query",
			input:    stringPtr("https://example.com?cursor=xyz789"),
			expected: stringPtr("xyz789"),
		},
		{
			name:     "cursor from query string",
			input:    stringPtr("?cursor=query123"),
			expected: stringPtr("query123"),
		},
		{
			name:     "URL without cursor param",
			input:    stringPtr("https://example.com?other=value"),
			expected: stringPtr("https://example.com?other=value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCursor(tt.input)
			if tt.expected == nil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestSanitizeCursor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "simple cursor",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "cursor with whitespace",
			input:    "  abc123  ",
			expected: "abc123",
		},
		{
			name:     "cursor from URL",
			input:    "https://example.com?cursor=xyz789",
			expected: "xyz789",
		},
		{
			name:     "cursor from query string",
			input:    "?cursor=query123",
			expected: "query123",
		},
		{
			name:     "URL without cursor",
			input:    "https://example.com?other=value",
			expected: "https://example.com?other=value",
		},
		{
			name:     "query string without cursor",
			input:    "?other=value&another=test",
			expected: "?other=value&another=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeCursor(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateParamsTime(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	now := time.Now()
	notBefore := now.Add(-24 * time.Hour)
	notAfter := now.Add(24 * time.Hour)
	maxAge := 7 * 24 * time.Hour

	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		EPAuditEventsList: {
			Params: []ParamSpec{
				{
					Name: "time_param",
					Type: ParamTime,
					Constraints: &ParamConstraints{
						NotBefore:       &notBefore,
						NotAfter:        &notAfter,
						MaxAge:          &maxAge,
						NotNewerThanNow: true,
					},
				},
			},
		},
	}

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid time",
			params: map[string]any{
				"time_param": now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "invalid time format",
			params: map[string]any{
				"time_param": "not a time",
			},
			wantErr: true,
			errMsg:  "must be valid RFC3339",
		},
		{
			name: "time in future",
			params: map[string]any{
				"time_param": now.Add(1 * time.Hour).Format(time.RFC3339),
			},
			wantErr: true,
			errMsg:  "cannot be in the future",
		},
		{
			name: "time before notBefore",
			params: map[string]any{
				"time_param": notBefore.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			wantErr: true,
			errMsg:  "must be >=",
		},
		{
			name: "time after notAfter",
			params: map[string]any{
				"time_param": notAfter.Add(1 * time.Hour).Format(time.RFC3339),
			},
			wantErr: true,
			errMsg:  "must be <=",
		},
		{
			name: "time exceeds max age",
			params: map[string]any{
				"time_param": now.Add(-8 * 24 * time.Hour).Format(time.RFC3339),
			},
			wantErr: true,
			errMsg:  "exceeds max age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParams(EPAuditEventsList, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

