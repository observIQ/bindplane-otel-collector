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
	authModeAPIKey         AuthMode = "apikey"
	authModeBearer         AuthMode = "bearer"
	authModeBasic          AuthMode = "basic"
	authModeAkamaiEdgeGrid AuthMode = "akamai_edgegrid"
)

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (m *AuthMode) UnmarshalText(text []byte) error {
	mode := AuthMode(text)
	switch mode {
	case authModeAPIKey, authModeBearer, authModeBasic, authModeAkamaiEdgeGrid:
		*m = mode
		return nil
	default:
		return fmt.Errorf("invalid auth mode: %s, must be one of: none, apikey, bearer, basic, akamai_edgegrid", text)
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
	AuthMode string `mapstructure:"auth_mode"`

	AuthAPIKeyHeaderName string `mapstructure:"apikey_header_name"`
	AuthAPIKeyValue      string `mapstructure:"apikey_value"`
	AuthBearerToken      string `mapstructure:"bearer_token"`
	AuthBasicUsername    string `mapstructure:"username"`
	AuthBasicPassword    string `mapstructure:"password"`

	// Akamai EdgeGrid authentication
	AuthAkamaiAccessToken  string `mapstructure:"akamai_access_token"`
	AuthAkamaiClientToken  string `mapstructure:"akamai_client_token"`
	AuthAkamaiClientSecret string `mapstructure:"akamai_client_secret"`

	// Pagination defines pagination configuration.
	Pagination PaginationConfig `mapstructure:"pagination"`

	// PollInterval is the interval between API polls.
	PollInterval time.Duration `mapstructure:"poll_interval"`

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

	// PageSizeFieldName is the name of the query parameter for page size (e.g., "perPage", "limit").
	PageSizeFieldName string `mapstructure:"page_size_field_name"`

	// PageSize is the page size to use.
	PageSize int `mapstructure:"page_size"`

	// InitialTimestamp is the initial timestamp to start from (optional).
	// If not set, will start from the beginning.
	InitialTimestamp time.Time `mapstructure:"initial_timestamp"`
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
	case string(authModeAPIKey), string(authModeBearer), string(authModeBasic), string(authModeAkamaiEdgeGrid):
		// Valid modes
	default:
		return fmt.Errorf("invalid auth mode: %s, must be one of: apikey, bearer, basic, akamai_edgegrid", c.AuthMode)
	}

	// Validate auth mode specific requirements
	switch c.AuthMode {
	case string(authModeAPIKey):
		if c.AuthAPIKeyHeaderName == "" {
			return fmt.Errorf("apikey_header_name is required when auth_mode is apikey")
		}
		if c.AuthAPIKeyValue == "" {
			return fmt.Errorf("apikey_value is required when auth_mode is apikey")
		}
	case string(authModeBearer):
		if c.AuthBearerToken == "" {
			return fmt.Errorf("bearer_token is required when auth_mode is bearer")
		}
	case string(authModeBasic):
		if c.AuthBasicUsername == "" {
			return fmt.Errorf("basic_username is required when auth_mode is basic")
		}
		if c.AuthBasicPassword == "" {
			return fmt.Errorf("basic_password is required when auth_mode is basic")
		}
	case string(authModeAkamaiEdgeGrid):
		if c.AuthAkamaiAccessToken == "" {
			return fmt.Errorf("akamai_access_token is required when auth_mode is akamai_edgegrid")
		}
		if c.AuthAkamaiClientToken == "" {
			return fmt.Errorf("akamai_client_token is required when auth_mode is akamai_edgegrid")
		}
		if c.AuthAkamaiClientSecret == "" {
			return fmt.Errorf("akamai_client_secret is required when auth_mode is akamai_edgegrid")
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
			return fmt.Errorf("pagination.offset_limit.offset_field_name is required when pagination.mode is offset_limit")
		}
		if c.Pagination.OffsetLimit.LimitFieldName == "" {
			return fmt.Errorf("pagination.offset_limit.limit_field_name is required when pagination.mode is offset_limit")
		}
	case paginationModePageSize:
		if c.Pagination.PageSize.PageNumFieldName == "" {
			return fmt.Errorf("pagination.page_size.page_num_field_name is required when pagination.mode is page_size")
		}
		if c.Pagination.PageSize.PageSizeFieldName == "" {
			return fmt.Errorf("pagination.page_size.page_size_field_name is required when pagination.mode is page_size")
		}
	case paginationModeTimestamp:
		if c.Pagination.Timestamp.ParamName == "" {
			return fmt.Errorf("pagination.timestamp.param_name is required when pagination.mode is timestamp")
		}
		if c.Pagination.Timestamp.TimestampFieldName == "" {
			return fmt.Errorf("pagination.timestamp.timestamp_field_name is required when pagination.mode is timestamp")
		}
		if c.Pagination.Timestamp.PageSizeFieldName == "" {
			return fmt.Errorf("pagination.timestamp.page_size_field_name is required when pagination.mode is timestamp")
		}
	}

	return nil
}
