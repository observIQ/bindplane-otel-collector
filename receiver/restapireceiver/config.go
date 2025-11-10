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
	AuthModeNone   AuthMode = "none"
	AuthModeAPIKey AuthMode = "apikey"
	AuthModeBearer AuthMode = "bearer"
	AuthModeBasic  AuthMode = "basic"
)

// PaginationMode defines the pagination mode for the REST API receiver.
type PaginationMode string

const (
	PaginationModeNone        PaginationMode = "none"
	PaginationModeOffsetLimit PaginationMode = "offset_limit"
	PaginationModePageSize    PaginationMode = "page_size"
)

// Config defines configuration for the REST API receiver.
type Config struct {
	// URL is the base URL for the REST API endpoint (required).
	URL string `mapstructure:"url"`

	// ResponseField is the name of the field in the response that contains the array of items.
	// If empty, the response is assumed to be a top-level array.
	ResponseField string `mapstructure:"response_field"`

	// Auth defines authentication configuration.
	Auth AuthConfig `mapstructure:"auth"`

	// Pagination defines pagination configuration.
	Pagination PaginationConfig `mapstructure:"pagination"`

	// TimeBasedOffset defines time-based offset configuration.
	TimeBasedOffset TimeBasedOffsetConfig `mapstructure:"time_based_offset"`

	// PollInterval is the interval between API polls.
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// ClientConfig defines HTTP client configuration.
	ClientConfig confighttp.ClientConfig `mapstructure:",squash"`

	// StorageID is the optional storage extension ID for checkpointing.
	StorageID *component.ID `mapstructure:"storage"`
}

// AuthConfig defines authentication configuration.
type AuthConfig struct {
	// Mode is the authentication mode: "none", "apikey", "bearer", or "basic".
	Mode AuthMode `mapstructure:"mode"`

	// APIKey defines API key authentication.
	APIKey APIKeyAuth `mapstructure:"apikey"`

	// BearerToken is the bearer token for bearer authentication.
	BearerToken string `mapstructure:"bearer_token"`

	// BasicAuth defines basic authentication.
	BasicAuth BasicAuth `mapstructure:"basic"`
}

// APIKeyAuth defines API key authentication configuration.
type APIKeyAuth struct {
	// HeaderName is the name of the header to use for the API key.
	HeaderName string `mapstructure:"header_name"`

	// Value is the API key value.
	Value string `mapstructure:"value"`
}

// BasicAuth defines basic authentication configuration.
type BasicAuth struct {
	// Username is the username for basic authentication.
	Username string `mapstructure:"username"`

	// Password is the password for basic authentication.
	Password string `mapstructure:"password"`
}

// PaginationConfig defines pagination configuration.
type PaginationConfig struct {
	// Mode is the pagination mode: "none", "offset_limit", or "page_size".
	Mode PaginationMode `mapstructure:"mode"`

	// OffsetLimit defines offset/limit pagination.
	OffsetLimit OffsetLimitPagination `mapstructure:"offset_limit"`

	// PageSize defines page/size pagination.
	PageSize PageSizePagination `mapstructure:"page_size"`

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

// TimeBasedOffsetConfig defines time-based offset configuration.
type TimeBasedOffsetConfig struct {
	// Enabled indicates whether time-based offset is enabled.
	Enabled bool `mapstructure:"enabled"`

	// ParamName is the name of the query parameter for the time-based offset.
	ParamName string `mapstructure:"param_name"`

	// OffsetTimestamp is the initial offset timestamp.
	OffsetTimestamp time.Time `mapstructure:"offset_timestamp"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	// Validate auth mode
	switch c.Auth.Mode {
	case AuthModeNone, AuthModeAPIKey, AuthModeBearer, AuthModeBasic:
		// Valid modes
	default:
		return fmt.Errorf("invalid auth mode: %s, must be one of: none, apikey, bearer, basic", c.Auth.Mode)
	}

	// Validate auth mode specific requirements
	switch c.Auth.Mode {
	case AuthModeAPIKey:
		if c.Auth.APIKey.HeaderName == "" {
			return fmt.Errorf("auth.apikey.header_name is required when auth.mode is apikey")
		}
		if c.Auth.APIKey.Value == "" {
			return fmt.Errorf("auth.apikey.value is required when auth.mode is apikey")
		}
	case AuthModeBearer:
		if c.Auth.BearerToken == "" {
			return fmt.Errorf("auth.bearer_token is required when auth.mode is bearer")
		}
	case AuthModeBasic:
		if c.Auth.BasicAuth.Username == "" {
			return fmt.Errorf("auth.basic.username is required when auth.mode is basic")
		}
		if c.Auth.BasicAuth.Password == "" {
			return fmt.Errorf("auth.basic.password is required when auth.mode is basic")
		}
	}

	// Validate pagination mode
	switch c.Pagination.Mode {
	case PaginationModeNone, PaginationModeOffsetLimit, PaginationModePageSize:
		// Valid modes
	default:
		return fmt.Errorf("invalid pagination mode: %s, must be one of: none, offset_limit, page_size", c.Pagination.Mode)
	}

	// Validate pagination mode specific requirements
	switch c.Pagination.Mode {
	case PaginationModeOffsetLimit:
		if c.Pagination.OffsetLimit.OffsetFieldName == "" {
			return fmt.Errorf("pagination.offset_limit.offset_field_name is required when pagination.mode is offset_limit")
		}
		if c.Pagination.OffsetLimit.LimitFieldName == "" {
			return fmt.Errorf("pagination.offset_limit.limit_field_name is required when pagination.mode is offset_limit")
		}
	case PaginationModePageSize:
		if c.Pagination.PageSize.PageNumFieldName == "" {
			return fmt.Errorf("pagination.page_size.page_num_field_name is required when pagination.mode is page_size")
		}
		if c.Pagination.PageSize.PageSizeFieldName == "" {
			return fmt.Errorf("pagination.page_size.page_size_field_name is required when pagination.mode is page_size")
		}
	}

	// Validate time-based offset
	if c.TimeBasedOffset.Enabled {
		if c.TimeBasedOffset.ParamName == "" {
			return fmt.Errorf("time_based_offset.param_name is required when time_based_offset.enabled is true")
		}
	}

	return nil
}
