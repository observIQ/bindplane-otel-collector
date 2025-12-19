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
)

var (
	// errMissingContext is the error for missing required field 'context'
	errMissingContext = errors.New("missing required field 'context'")
	// errMissingField is the error for missing required field 'field'
	errMissingField = errors.New("missing required field 'field'")
	// errInvalidContext is the error for an invalid context
	errInvalidContext = errors.New("invalid context")
	// errMissingSource is the error for missing source configuration
	errMissingSource = errors.New("must specify either 'csv', 'redis', or 'api' configuration")
)

// Config is the configuration for the processor
type Config struct {
	// Common fields
	Context    string `mapstructure:"context"`
	Field      string `mapstructure:"field"`
	SourceType string `mapstructure:"source_type"`

	// Cache configuration
	CacheEnabled bool          `mapstructure:"cache_enabled"`
	CacheTTL     time.Duration `mapstructure:"cache_ttl"`
	StorageID    *component.ID `mapstructure:"storage"`

	// Source configurations
	CSV   string       `mapstructure:"csv"`
	Redis *RedisConfig `mapstructure:"redis"`
	API   *APIConfig   `mapstructure:"api"`
}

// APIConfig is the configuration for API-based lookups
type APIConfig struct {
	URL             string            `mapstructure:"url"`
	Method          string            `mapstructure:"method"`
	Headers         map[string]string `mapstructure:"headers"`
	Timeout         time.Duration     `mapstructure:"timeout"`
	ResponseMapping map[string]string `mapstructure:"response_mapping"`
}

// RedisConfig is the configuration for Redis-based lookups
type RedisConfig struct {
	Address   string `mapstructure:"address"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	TLS       bool   `mapstructure:"tls"`
	KeyPrefix string `mapstructure:"key_prefix"`
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

	// Validate that at least one source is configured
	sourceCount := 0
	if cfg.CSV != "" {
		sourceCount++
	}
	if cfg.Redis != nil {
		sourceCount++
	}
	if cfg.API != nil {
		sourceCount++
	}

	if sourceCount == 0 {
		return errMissingSource
	}

	return nil
}
