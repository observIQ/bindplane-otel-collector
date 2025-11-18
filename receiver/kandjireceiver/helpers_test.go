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

func TestValidateParamName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid alphanumeric",
			input:    "limit",
			expected: true,
		},
		{
			name:     "valid with underscore",
			input:    "sort_by",
			expected: true,
		},
		{
			name:     "valid with hyphen",
			input:    "start-date",
			expected: true,
		},
		{
			name:     "valid with numbers",
			input:    "param123",
			expected: true,
		},
		{
			name:     "valid mixed",
			input:    "param_123-test",
			expected: true,
		},
		{
			name:     "invalid with space",
			input:    "param name",
			expected: false,
		},
		{
			name:     "invalid with special chars",
			input:    "param@name",
			expected: false,
		},
		{
			name:     "invalid with dot",
			input:    "param.name",
			expected: false,
		},
		{
			name:     "invalid with slash",
			input:    "param/name",
			expected: false,
		},
		{
			name:     "invalid empty",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid with unicode",
			input:    "paramñame",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateParamName(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeStringParam(t *testing.T) {
	maxLen := 10
	minLen := 3

	tests := []struct {
		name        string
		input       string
		constraints *ParamConstraints
		expected    string
	}{
		{
			name:        "trim whitespace",
			input:       "  hello  ",
			constraints: nil,
			expected:    "hello",
		},
		{
			name:        "remove control characters",
			input:       "hello\x00world\x01test",
			constraints: nil,
			expected:    "helloworldtest",
		},
		{
			name:        "keep newline tab carriage return",
			input:       "hello\nworld\ttest\r",
			constraints: nil,
			expected:    "hello\nworld\ttest\r",
		},
		{
			name:        "trim and remove controls",
			input:       "  hello\x00world  ",
			constraints: nil,
			expected:    "helloworld",
		},
		{
			name: "apply max length",
			input: "this is a very long string",
			constraints: &ParamConstraints{
				MaxLen: &maxLen,
			},
			expected: "this is a ",
		},
		{
			name: "apply min length - below min",
			input: "ab",
			constraints: &ParamConstraints{
				MinLen: &minLen,
			},
			expected: "", // Returns empty if below min (will be caught by validation)
		},
		{
			name: "apply min length - at min",
			input: "abc",
			constraints: &ParamConstraints{
				MinLen: &minLen,
			},
			expected: "abc",
		},
		{
			name: "apply both min and max",
			input: "  this is a very long string  ",
			constraints: &ParamConstraints{
				MinLen: &minLen,
				MaxLen: &maxLen,
			},
			expected: "this is a ",
		},
		{
			name:        "empty string",
			input:       "",
			constraints: nil,
			expected:    "",
		},
		{
			name:        "only whitespace",
			input:       "   \t\n  ",
			constraints: nil,
			expected:    "",
		},
		{
			name:        "unicode characters",
			input:       "hello世界",
			constraints: nil,
			expected:    "hello世界",
		},
		{
			name:        "remove null bytes",
			input:       "test\x00\x00test",
			constraints: nil,
			expected:    "testtest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeStringParam(tt.input, tt.constraints)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateUUIDFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid UUID lowercase",
			input:    "123e4567-e89b-12d3-a456-426614174000",
			expected: true,
		},
		{
			name:     "valid UUID uppercase",
			input:    "123E4567-E89B-12D3-A456-426614174000",
			expected: true,
		},
		{
			name:     "valid UUID mixed case",
			input:    "123e4567-E89b-12D3-a456-426614174000",
			expected: true,
		},
		{
			name:     "invalid - wrong format",
			input:    "123e4567e89b12d3a456426614174000",
			expected: false,
		},
		{
			name:     "invalid - wrong separators",
			input:    "123e4567_e89b_12d3_a456_426614174000",
			expected: false,
		},
		{
			name:     "invalid - too short",
			input:    "123e4567-e89b-12d3-a456-42661417400",
			expected: false,
		},
		{
			name:     "invalid - too long",
			input:    "123e4567-e89b-12d3-a456-4266141740000",
			expected: false,
		},
		{
			name:     "invalid - wrong segment lengths",
			input:    "123e456-e89b-12d3-a456-426614174000",
			expected: false,
		},
		{
			name:     "invalid - non-hex characters",
			input:    "123g4567-e89b-12d3-a456-426614174000",
			expected: false,
		},
		{
			name:     "invalid - empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid - with whitespace",
			input:    " 123e4567-e89b-12d3-a456-426614174000 ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateUUIDFormat(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateParamsWithSanitization(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	maxLen := 20
	minLen := 3

	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		EPAuditEventsList: {
			Params: []ParamSpec{
				{
					Name: "string_param",
					Type: ParamString,
					Constraints: &ParamConstraints{
						MinLen: &minLen,
						MaxLen: &maxLen,
					},
				},
				{
					Name: "uuid_param",
					Type: ParamUUID,
				},
				{
					Name: "enum_param",
					Type: ParamEnum,
					AllowedVals: []string{"value1", "value2"},
				},
			},
		},
	}

	tests := []struct {
		name           string
		params         map[string]any
		wantErr        bool
		errMsg         string
		expectedParams map[string]any // Expected sanitized params
	}{
		{
			name: "string param trimmed",
			params: map[string]any{
				"string_param": "  hello world  ",
			},
			wantErr: false,
			expectedParams: map[string]any{
				"string_param": "hello world",
			},
		},
		{
			name: "string param with control chars removed",
			params: map[string]any{
				"string_param": "hello\x00world",
			},
			wantErr: false,
			expectedParams: map[string]any{
				"string_param": "helloworld",
			},
		},
		{
			name: "string param exceeds max length",
			params: map[string]any{
				"string_param": "this is a very long string that exceeds max length",
			},
			wantErr: true,
			errMsg:  "exceeds max length",
		},
		{
			name: "string param below min length",
			params: map[string]any{
				"string_param": "ab",
			},
			wantErr: true,
			errMsg:  "must be at least",
		},
		{
			name: "valid UUID",
			params: map[string]any{
				"uuid_param": "123e4567-e89b-12d3-a456-426614174000",
			},
			wantErr: false,
			expectedParams: map[string]any{
				"uuid_param": "123e4567-e89b-12d3-a456-426614174000",
			},
		},
		{
			name: "UUID with whitespace trimmed",
			params: map[string]any{
				"uuid_param": "  123e4567-e89b-12d3-a456-426614174000  ",
			},
			wantErr: false,
			expectedParams: map[string]any{
				"uuid_param": "123e4567-e89b-12d3-a456-426614174000",
			},
		},
		{
			name: "invalid UUID format",
			params: map[string]any{
				"uuid_param": "not-a-uuid",
			},
			wantErr: true,
			errMsg:  "must be a valid UUID format",
		},
		{
			name: "enum param trimmed",
			params: map[string]any{
				"enum_param": "  value1  ",
			},
			wantErr: false,
			expectedParams: map[string]any{
				"enum_param": "value1",
			},
		},
		{
			name: "invalid parameter name",
			params: map[string]any{
				"invalid@name": "value",
			},
			wantErr: true,
			errMsg:  "invalid parameter name",
		},
		{
			name: "parameter name with space",
			params: map[string]any{
				"invalid name": "value",
			},
			wantErr: true,
			errMsg:  "invalid parameter name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of params to avoid mutating the test case
			paramsCopy := make(map[string]any)
			for k, v := range tt.params {
				paramsCopy[k] = v
			}

			err := ValidateParams(EPAuditEventsList, paramsCopy)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				if tt.expectedParams != nil {
					for k, expectedVal := range tt.expectedParams {
						actualVal, exists := paramsCopy[k]
						require.True(t, exists, "expected param %s to exist", k)
						require.Equal(t, expectedVal, actualVal, "param %s should be sanitized", k)
					}
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

