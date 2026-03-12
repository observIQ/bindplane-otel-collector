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
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

const (
	// Epoch timestamp format constants for use in TimestampFormat.
	// These send the timestamp as a numeric epoch value instead of a formatted string.
	epochSeconds           = "epoch_s"
	epochMilliseconds      = "epoch_ms"
	epochMicroseconds      = "epoch_us"
	epochNanoseconds       = "epoch_ns"
	epochSecondsFractional = "epoch_s_frac"
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

	// BackoffMultiplier is the multiplier for increasing the poll interval
	// when no data or a partial page is returned. For example, with a multiplier
	// of 2.0 and a current interval of 10s, the next interval will be 20s.
	// Must be greater than 1.0. Defaults to 2.0.
	BackoffMultiplier float64 `mapstructure:"backoff_multiplier"`

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
	TimestampFieldName string `mapstructure:"timestamp_field_name"`

	// TimestampFormat is the format for the timestamp query parameter.
	// Common formats:
	//   - "2006-01-02T15:04:05Z07:00" (RFC3339, default)
	//   - "20060102150405" (YYYYMMDDHHMMSS)
	//   - "2006-01-02 15:04:05"
	//   - "epoch_s" (Unix epoch seconds)
	//   - "epoch_ms" (Unix epoch milliseconds)
	//   - "epoch_us" (Unix epoch microseconds)
	//   - "epoch_ns" (Unix epoch nanoseconds)
	// If not set, defaults to RFC3339.
	TimestampFormat string `mapstructure:"timestamp_format"`

	// PageSizeFieldName is the name of the query parameter for page size (e.g., "perPage", "limit").
	PageSizeFieldName string `mapstructure:"page_size_field_name"`

	// PageSize is the page size to use.
	PageSize int `mapstructure:"page_size"`

	// InitialTimestamp is the initial timestamp to start from (optional).
	// If not set, will start from the beginning.
	// Accepts the configured timestamp_format or RFC3339 (e.g., "2025-01-01T00:00:00Z").
	// For epoch formats, accepts a numeric string (e.g., "1704067200" for epoch_s).
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
			// For epoch formats, validate that initial_timestamp is a numeric value
			if isEpochFormat(c.Pagination.Timestamp.TimestampFormat) {
				if _, err := strconv.ParseFloat(c.Pagination.Timestamp.InitialTimestamp, 64); err == nil {
					parsed = true
				}
				if !parsed {
					return fmt.Errorf("initial_timestamp %q must be a numeric value when using epoch timestamp_format (%s)", c.Pagination.Timestamp.InitialTimestamp, c.Pagination.Timestamp.TimestampFormat)
				}
			} else {
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

	// Apply default backoff multiplier if not configured
	if c.BackoffMultiplier == 0 {
		c.BackoffMultiplier = 2.0
	}

	if c.BackoffMultiplier <= 1.0 {
		return fmt.Errorf("backoff_multiplier must be greater than 1.0")
	}

	return nil
}

// isEpochFormat returns true if the given format string is one of the epoch timestamp formats.
func isEpochFormat(format string) bool {
	switch format {
	case epochSeconds, epochMilliseconds, epochMicroseconds, epochNanoseconds, epochSecondsFractional:
		return true
	}
	return false
}

// formatTimestampEpoch formats a time.Time as an epoch numeric string.
func formatTimestampEpoch(t time.Time, format string) string {
	switch format {
	case epochSeconds:
		return strconv.FormatInt(t.Unix(), 10)
	case epochMilliseconds:
		return strconv.FormatInt(t.UnixMilli(), 10)
	case epochMicroseconds:
		return strconv.FormatInt(t.UnixMicro(), 10)
	case epochNanoseconds:
		return strconv.FormatInt(t.UnixNano(), 10)
	case epochSecondsFractional:
		nsec := int64(t.Nanosecond())
		if nsec == 0 {
			return strconv.FormatInt(t.Unix(), 10)
		}
		frac := strings.TrimRight(fmt.Sprintf("%09d", nsec), "0")
		return fmt.Sprintf("%d.%s", t.Unix(), frac)
	default:
		return strconv.FormatInt(t.Unix(), 10)
	}
}

// parseEpochTimestamp parses a numeric epoch string into a time.Time based on the epoch format.
// For epoch_s, epoch_ms, epoch_us, epoch_ns: the value must be an integer in the configured unit.
// For epoch_s_frac: the value is fractional seconds (e.g., "1704067200.123456") where the integer
// part is seconds and the fractional digits are sub-second precision.
func parseEpochTimestamp(value string, format string) (time.Time, error) {
	// epoch_s_frac: fractional seconds (e.g., "1704067200.123456")
	if format == epochSecondsFractional {
		parts := strings.SplitN(value, ".", 2)
		sec, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch timestamp %q: %w", value, err)
		}
		var nsec int64
		if len(parts) == 2 && len(parts[1]) > 0 {
			fracStr := parts[1]
			if len(fracStr) < 9 {
				fracStr += strings.Repeat("0", 9-len(fracStr))
			} else if len(fracStr) > 9 {
				fracStr = fracStr[:9]
			}
			nsec, err = strconv.ParseInt(fracStr, 10, 64)
			if err != nil {
				return time.Time{}, fmt.Errorf("failed to parse fractional epoch timestamp %q: %w", value, err)
			}
		}
		return time.Unix(sec, nsec), nil
	}

	// All other epoch formats: integer value in the configured unit
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse epoch timestamp %q: %w", value, err)
	}
	switch format {
	case epochSeconds:
		return time.Unix(n, 0), nil
	case epochMilliseconds:
		return time.Unix(0, n*int64(time.Millisecond)), nil
	case epochMicroseconds:
		return time.Unix(0, n*int64(time.Microsecond)), nil
	case epochNanoseconds:
		return time.Unix(0, n), nil
	default:
		return time.Unix(n, 0), nil
	}
}
