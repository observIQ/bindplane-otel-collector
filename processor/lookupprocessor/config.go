// Copyright  observIQ, Inc.
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

// Package lookupprocessor provides a processor that looks up values and adds them to telemetry
package lookupprocessor

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

const (
	// bodyContext is the context for the body of the telemetry
	bodyContext = "body"
	// attributesContext is the context for the attributes of the telemetry
	attributesContext = "attributes"
	// resourceContext is the context for the resource of the telemetry
	resourceContext = "resource.attributes"

	// Source types
	sourceTypeCSV   = "csv"
	sourceTypeRedis = "redis"
	sourceTypeAPI   = "api"

	// Default values
	defaultCacheTTL     = 5 * time.Minute
	defaultAPITimeout   = 30 * time.Second
	defaultHTTPMethod   = "GET"
)

var (
	// errMissingCSV is the error for missing required field 'csv'
	errMissingCSV = errors.New("missing required field 'csv'")
	// errMissingContext is the error for missing required field 'context'
	errMissingContext = errors.New("missing required field 'context'")
	// errMissingField is the error for missing required field 'field'
	errMissingField = errors.New("missing required field 'field'")
	// errInvalidContext is the error for an invalid context
	errInvalidContext = errors.New("invalid context")
	// errInvalidSourceType is the error for an invalid source type
	errInvalidSourceType = errors.New("invalid source_type, must be one of: csv, redis, api")
	// errMissingRedisConfig is the error for missing redis config
	errMissingRedisConfig = errors.New("missing required field 'redis' when source_type is 'redis'")
	// errMissingAPIConfig is the error for missing api config
	errMissingAPIConfig = errors.New("missing required field 'api' when source_type is 'api'")
	// errMissingRedisAddress is the error for missing redis address
	errMissingRedisAddress = errors.New("missing required field 'redis.address'")
	// errMissingAPIURL is the error for missing api url
	errMissingAPIURL = errors.New("missing required field 'api.url'")
)

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	TLS      bool   `mapstructure:"tls"`
	KeyPrefix string `mapstructure:"key_prefix"`
}

// APIConfig holds API endpoint configuration
type APIConfig struct {
	URL             string            `mapstructure:"url"`
	Method          string            `mapstructure:"method"`
	Headers         map[string]string `mapstructure:"headers"`
	Timeout         time.Duration     `mapstructure:"timeout"`
	ResponseMapping map[string]string `mapstructure:"response_mapping"`
}

// Config is the configuration for the processor
type Config struct {
	// Common fields
	Context string `mapstructure:"context"`
	Field   string `mapstructure:"field"`

	// Source configuration
	SourceType string `mapstructure:"source_type"`

	// CSV source (legacy, kept for backward compatibility)
	CSV string `mapstructure:"csv"`

	// Redis source
	Redis *RedisConfig `mapstructure:"redis"`

	// API source
	API *APIConfig `mapstructure:"api"`

	// Cache configuration
	CacheEnabled   bool              `mapstructure:"cache_enabled"`
	CacheTTL       time.Duration     `mapstructure:"cache_ttl"`
	CacheStorageID *component.ID     `mapstructure:"cache_storage_id"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	if cfg.Context == "" {
		return errMissingContext
	}

	if cfg.Field == "" {
		return errMissingField
	}

	switch cfg.Context {
	case bodyContext, attributesContext, resourceContext:
	default:
		return errInvalidContext
	}

	// Determine source type (backward compatibility: if CSV is set and source_type is empty, use CSV)
	if cfg.SourceType == "" {
		if cfg.CSV != "" {
			cfg.SourceType = sourceTypeCSV
		} else {
			return errMissingCSV
		}
	}

	// Validate source type
	switch cfg.SourceType {
	case sourceTypeCSV:
		if cfg.CSV == "" {
			return errMissingCSV
		}
	case sourceTypeRedis:
		if cfg.Redis == nil {
			return errMissingRedisConfig
		}
		if cfg.Redis.Address == "" {
			return errMissingRedisAddress
		}
	case sourceTypeAPI:
		if cfg.API == nil {
			return errMissingAPIConfig
		}
		if cfg.API.URL == "" {
			return errMissingAPIURL
		}
	default:
		return errInvalidSourceType
	}

	return nil
}

// SetDefaults sets default values for optional configuration fields
func (cfg *Config) SetDefaults() {
	// Set default cache settings
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = defaultCacheTTL
	}

	// Set API defaults
	if cfg.API != nil {
		if cfg.API.Method == "" {
			cfg.API.Method = defaultHTTPMethod
		}
		if cfg.API.Timeout == 0 {
			cfg.API.Timeout = defaultAPITimeout
		}
	}
}
