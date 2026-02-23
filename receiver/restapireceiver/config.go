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

package restapireceiver // import "github.com/observiq/bindplane-otel-collector/receiver/restapireceiver"

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

// AuthMode defines the authentication mode for the REST API receiver.
type AuthMode string

const (
	authModeNone           AuthMode = "none"
	authModeAPIKey         AuthMode = "apikey"
	authModeBearer         AuthMode = "bearer"
	authModeBasic          AuthMode = "basic"
	authModeOAuth2         AuthMode = "oauth2"
	authModeAkamaiEdgeGrid AuthMode = "akamai_edgegrid"
)

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (m *AuthMode) UnmarshalText(text []byte) error {
	mode := AuthMode(text)
	switch mode {
	case authModeNone, authModeAPIKey, authModeBearer, authModeBasic, authModeOAuth2, authModeAkamaiEdgeGrid:
		*m = mode
		return nil
	default:
		return fmt.Errorf("invalid auth mode: %s, must be one of: none, apikey, bearer, basic, oauth2, akamai_edgegrid", text)
	}
}

// PaginationMode defines the pagination mode for the REST API receiver.
type PaginationMode string

const (
	paginationModeNone        PaginationMode = "none"
	paginationModeOffsetLimit PaginationMode = "offset_limit"
	paginationModePageSize    PaginationMode = "page_size"
	paginationModeTimestamp   PaginationMode = "timestamp"
)

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (m *PaginationMode) UnmarshalText(text []byte) error {
	mode := PaginationMode(text)
	switch mode {
	case paginationModeNone, paginationModeOffsetLimit, paginationModePageSize, paginationModeTimestamp:
		*m = mode
		return nil
	default:
		return fmt.Errorf("invalid pagination mode: %s, must be one of: none, offset_limit, page_size, timestamp", text)
	}
}

// Config defines configuration for the REST API receiver.
type Config struct {
	// URL is the base URL for the REST API endpoint (required).
	URL string `mapstructure:"url"`

	// ResponseField is the name of the field in the response that contains the array of items.
	// If empty, the response is assumed to be a top-level array.
	ResponseField string `mapstructure:"response_field"`

	// Auth defines authentication configuration.
	AuthMode             AuthMode             `mapstructure:"auth_mode"`
	APIKeyConfig         APIKeyConfig         `mapstructure:"apikey"`
	BearerConfig         BearerConfig         `mapstructure:"bearer"`
	BasicConfig          BasicConfig          `mapstructure:"basic"`
	OAuth2Config         OAuth2Config         `mapstructure:"oauth2"`
	AkamaiEdgeGridConfig AkamaiEdgeGridConfig `mapstructure:"akamai_edgegrid"`

	// Pagination defines pagination configuration.
	Pagination PaginationConfig `mapstructure:"pagination"`

	// MinPollInterval is the minimum interval between API polls.
	// The receiver uses adaptive polling that resets to this interval when data
	// is received, and backs off when no data is returned.
	MinPollInterval time.Duration `mapstructure:"min_poll_interval"`

	// MaxPollInterval is the maximum interval between API polls.
	// The receiver uses adaptive polling that starts with a short interval and
	// backs off when no data is returned, up to this maximum.
	MaxPollInterval time.Duration `mapstructure:"max_poll_interval"`

	// ClientConfig defines HTTP client configuration.
	ClientConfig confighttp.ClientConfig `mapstructure:",squash"`

	// StorageID is the optional storage extension ID for checkpointing.
	StorageID *component.ID `mapstructure:"storage"`

	// Metrics defines configuration for metrics extraction.
	Metrics MetricsConfig `mapstructure:"metrics"`
}

// MetricsConfig defines configuration for extracting metrics from API responses.
type MetricsConfig struct {
	// NameField is the name of the field in each response item that contains the metric name.
	// If not specified or not found, defaults to "restapi.metric".
	NameField string `mapstructure:"name_field"`

	// DescriptionField is the name of the field in each response item that contains the metric description.
	// If not specified or not found, defaults to "Metric from REST API".
	DescriptionField string `mapstructure:"description_field"`

	// TypeField is the name of the field in each response item that contains the metric type.
	// Valid types: "gauge", "sum", "histogram", "summary".
	// If not specified or not found, defaults to "gauge".
	TypeField string `mapstructure:"type_field"`

	// UnitField is the name of the field in each response item that contains the metric unit.
	// If not specified or not found, no unit will be set.
	UnitField string `mapstructure:"unit_field"`

	// MonotonicField is the name of the field in each response item that indicates if a sum metric is monotonic.
	// Only applies to sum metrics. Should contain a boolean value.
	// If not specified or not found, defaults to false for safety.
	MonotonicField string `mapstructure:"monotonic_field"`

	// AggregationTemporalityField is the name of the field in each response item that contains the aggregation temporality.
	// Valid values: "cumulative", "delta".
	// Only applies to sum and histogram metrics.
	// If not specified or not found, defaults to "cumulative".
	AggregationTemporalityField string `mapstructure:"aggregation_temporality_field"`
}

// APIKeyConfig defines API key authentication configuration.
type APIKeyConfig struct {
	HeaderName string `mapstructure:"header_name"`
	Value      string `mapstructure:"value"`
}

// BearerConfig defines bearer token authentication configuration.
type BearerConfig struct {
	Token string `mapstructure:"token"`
}

// BasicConfig defines basic authentication configuration.
type BasicConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// OAuth2Config defines OAuth2 client credentials authentication configuration.
type OAuth2Config struct {
	ClientID       string            `mapstructure:"client_id"`
	ClientSecret   string            `mapstructure:"client_secret"`
	TokenURL       string            `mapstructure:"token_url"`
	Scopes         []string          `mapstructure:"scopes"`
	EndpointParams map[string]string `mapstructure:"endpoint_params"`
}

// AkamaiEdgeGridConfig defines Akamai EdgeGrid authentication configuration.
type AkamaiEdgeGridConfig struct {
	AccessToken  string `mapstructure:"access_token"`
	ClientToken  string `mapstructure:"client_token"`
	ClientSecret string `mapstructure:"client_secret"`
}

// PaginationConfig defines pagination configuration.
type PaginationConfig struct {
	// Mode is the pagination mode: "none", "offset_limit", or "page_size".
	Mode PaginationMode `mapstructure:"mode"`

	// OffsetLimit defines offset/limit pagination.
	OffsetLimit OffsetLimitPagination `mapstructure:"offset_limit"`

	// PageSize defines page/size pagination.
	PageSize PageSizePagination `mapstructure:"page_size"`

	// Timestamp defines timestamp-based pagination.
	Timestamp TimestampPagination `mapstructure:"timestamp"`

	// TotalRecordCountField is the name of the field in the response that contains the total record count.
	TotalRecordCountField string `mapstructure:"total_record_count_field"`

	// PageLimit is the maximum number of pages to fetch (0 = no limit).
	PageLimit int `mapstructure:"page_limit"`

	// ZeroBasedIndex indicates whether pagination starts at index 0 (true) or 1 (false).
	ZeroBasedIndex bool `mapstructure:"zero_based_index"`
}

// OffsetLimitPagination defines offset/limit pagination configuration.
type OffsetLimitPagination struct {
	// OffsetFieldName is the name of the query parameter for offset.
	OffsetFieldName string `mapstructure:"offset_field_name"`

	// StartingOffset is the starting offset value.
	StartingOffset int `mapstructure:"starting_offset"`

	// LimitFieldName is the name of the query parameter for limit.
	LimitFieldName string `mapstructure:"limit_field_name"`
}

// PageSizePagination defines page/size pagination configuration.
type PageSizePagination struct {
	// PageNumFieldName is the name of the query parameter for page number.
	PageNumFieldName string `mapstructure:"page_num_field_name"`

	// StartingPage is the starting page number.
	StartingPage int `mapstructure:"starting_page"`

	// PageSizeFieldName is the name of the query parameter for page size.
	PageSizeFieldName string `mapstructure:"page_size_field_name"`

	// TotalPagesFieldName is the name of the field in the response that contains the total page count.
	TotalPagesFieldName string `mapstructure:"total_pages_field_name"`
}

// TimestampPagination defines timestamp-based pagination configuration.
type TimestampPagination struct {
	// ParamName is the name of the query parameter for the timestamp (e.g., "t0", "since", "after", "start_time").
	ParamName string `mapstructure:"param_name"`

	// TimestampFieldName is the name of the field in each response item that contains the timestamp value.
	// This is used to extract the timestamp from the last item for the next page.
	// For Meraki API, this is typically "ts" (timestamp).
	TimestampFieldName string `mapstructure:"timestamp_field_name"`

	// TimestampFormat is the Go time format string for the timestamp query parameter.
	// Common formats:
	//   - "2006-01-02T15:04:05Z07:00" (RFC3339, default)
	//   - "20060102150405" (YYYYMMDDHHMMSS)
	//   - "2006-01-02 15:04:05"
	// If not set, defaults to RFC3339.
	TimestampFormat string `mapstructure:"timestamp_format"`

	// PageSizeFieldName is the name of the query parameter for page size (e.g., "perPage", "limit").
	PageSizeFieldName string `mapstructure:"page_size_field_name"`

	// PageSize is the page size to use.
	PageSize int `mapstructure:"page_size"`

	// InitialTimestamp is the initial timestamp to start from (optional).
	// If not set, will start from the beginning.
	// Accepts the configured timestamp_format or RFC3339 (e.g., "2025-01-01T00:00:00Z").
	InitialTimestamp string `mapstructure:"initial_timestamp"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	// Validate auth
	if c.AuthMode == "" {
		return fmt.Errorf("auth is required")
	}

	// Validate auth mode
	switch c.AuthMode {
	case authModeNone, authModeAPIKey, authModeBearer, authModeBasic, authModeOAuth2, authModeAkamaiEdgeGrid:
		// Valid modes
	default:
		return fmt.Errorf("invalid auth mode: %s, must be one of: none, apikey, bearer, basic, oauth2, akamai_edgegrid", c.AuthMode)
	}

	// Validate auth mode specific requirements
	switch c.AuthMode {
	case authModeAPIKey:
		if c.APIKeyConfig.HeaderName == "" {
			return fmt.Errorf("header_name is required when auth_mode is apikey")
		}
		if c.APIKeyConfig.Value == "" {
			return fmt.Errorf("value is required when auth_mode is apikey")
		}
	case authModeBearer:
		if c.BearerConfig.Token == "" {
			return fmt.Errorf("token is required when auth_mode is bearer")
		}
	case authModeBasic:
		if c.BasicConfig.Username == "" {
			return fmt.Errorf("username is required when auth_mode is basic")
		}
		if c.BasicConfig.Password == "" {
			return fmt.Errorf("password is required when auth_mode is basic")
		}
	case authModeOAuth2:
		if c.OAuth2Config.ClientID == "" {
			return fmt.Errorf("client_id is required when auth_mode is oauth2")
		}
		if c.OAuth2Config.ClientSecret == "" {
			return fmt.Errorf("client_secret is required when auth_mode is oauth2")
		}
		if c.OAuth2Config.TokenURL == "" {
			return fmt.Errorf("token_url is required when auth_mode is oauth2")
		}
	case authModeAkamaiEdgeGrid:
		if c.AkamaiEdgeGridConfig.AccessToken == "" {
			return fmt.Errorf("access_token is required when auth_mode is akamai_edgegrid")
		}
		if c.AkamaiEdgeGridConfig.ClientToken == "" {
			return fmt.Errorf("client_token is required when auth_mode is akamai_edgegrid")
		}
		if c.AkamaiEdgeGridConfig.ClientSecret == "" {
			return fmt.Errorf("client_secret is required when auth_mode is akamai_edgegrid")
		}
	}

	// Validate pagination mode
	switch c.Pagination.Mode {
	case paginationModeNone, paginationModeOffsetLimit, paginationModePageSize, paginationModeTimestamp:
		// Valid modes
	default:
		return fmt.Errorf("invalid pagination mode: %s, must be one of: none, offset_limit, page_size, timestamp", c.Pagination.Mode)
	}

	// Validate pagination mode specific requirements
	switch c.Pagination.Mode {
	case paginationModeOffsetLimit:
		if c.Pagination.OffsetLimit.OffsetFieldName == "" {
			return fmt.Errorf("offset_field_name is required when pagination.mode is offset_limit")
		}
		if c.Pagination.OffsetLimit.LimitFieldName == "" {
			return fmt.Errorf("limit_field_name is required when pagination.mode is offset_limit")
		}
	case paginationModePageSize:
		if c.Pagination.PageSize.PageNumFieldName == "" {
			return fmt.Errorf("page_num_field_name is required when pagination.mode is page_size")
		}
		if c.Pagination.PageSize.PageSizeFieldName == "" {
			return fmt.Errorf("page_size_field_name is required when pagination.mode is page_size")
		}
	case paginationModeTimestamp:
		if c.Pagination.Timestamp.ParamName == "" {
			return fmt.Errorf("param_name is required when pagination.mode is timestamp")
		}
		if c.Pagination.Timestamp.TimestampFieldName == "" {
			return fmt.Errorf("timestamp_field_name is required when pagination.mode is timestamp")
		}
		// Validate initial_timestamp format if provided
		if c.Pagination.Timestamp.InitialTimestamp != "" {
			var parsed bool
			// First try the user's configured format (they likely copied the timestamp from the API)
			if c.Pagination.Timestamp.TimestampFormat != "" {
				if _, err := time.Parse(c.Pagination.Timestamp.TimestampFormat, c.Pagination.Timestamp.InitialTimestamp); err == nil {
					parsed = true
				}
			}
			// Fall back to RFC3339 (the default format)
			if !parsed {
				if _, err := time.Parse(time.RFC3339, c.Pagination.Timestamp.InitialTimestamp); err == nil {
					parsed = true
				}
			}
			if !parsed {
				formatHint := "RFC3339 (e.g., 2025-01-01T00:00:00Z)"
				if c.Pagination.Timestamp.TimestampFormat != "" {
					formatHint = fmt.Sprintf("configured timestamp_format (%s) or RFC3339", c.Pagination.Timestamp.TimestampFormat)
				}
				return fmt.Errorf("initial_timestamp %q could not be parsed; must match %s", c.Pagination.Timestamp.InitialTimestamp, formatHint)
			}
		}
	}

	// Apply defaults if not configured (zero value means not set)
	if c.MinPollInterval == 0 {
		c.MinPollInterval = 10 * time.Second
	}

	if c.MinPollInterval < 0 {
		return fmt.Errorf("min_poll_interval must be greater than or equal to 0")
	}

	if c.MaxPollInterval == 0 {
		c.MaxPollInterval = 5 * time.Minute
	}

	if c.MaxPollInterval < 0 {
		return fmt.Errorf("max_poll_interval must be greater than or equal to 0")
	}

	if c.MinPollInterval > c.MaxPollInterval {
		return fmt.Errorf("min_poll_interval (%s) must be less than or equal to max_poll_interval (%s)", c.MinPollInterval, c.MaxPollInterval)
	}

	return nil
}
