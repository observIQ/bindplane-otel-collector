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

// Package logdeduplicationprocessor provides a processor that counts logs as metrics.
package logdeduplicationprocessor

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config defaults
const (
	// defaultInterval is the default export interval.
	defaultInterval = 10 * time.Second

	// defaultLogCountAttribute is the default log count attribute
	defaultLogCountAttribute = "log_count"

	// defaultTimezone is the default timezone
	defaultTimezone = "UTC"

	// bodyField is the name of the body field for matching
	bodyField = "body"

	// attributeField is the name of the attribute field for matching
	attributeField = "attributes"
)

var defaultMatchFields = []string{bodyField, attributeField}

// Config errors
var (
	errInvalidLogCountAttribute = errors.New("log_count_attribute must be set")
	errInvalidInterval          = errors.New("interval must be greater than 0")
)

// Config is the config of the processor.
type Config struct {
	LogCountAttribute string        `mapstructure:"log_count_attribute"`
	Interval          time.Duration `mapstructure:"interval"`
	Timezone          string        `mapstructure:"timezone"`
	MatchFields       []string      `mapstructure:"match_fields"`
}

// createDefaultConfig returns the default config for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		LogCountAttribute: defaultLogCountAttribute,
		Interval:          defaultInterval,
		Timezone:          defaultTimezone,
		MatchFields:       defaultMatchFields,
	}
}

// Validate validates the configuration
func (c Config) Validate() error {
	if c.Interval <= 0 {
		return errInvalidInterval
	}

	if c.LogCountAttribute == "" {
		return errInvalidLogCountAttribute
	}

	_, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return fmt.Errorf("timezone is invalid: %w", err)
	}

	return c.validateMatchFields()
}

// validateMatchFields validates that all the match fields start with either `body` or `attributes`
func (c Config) validateMatchFields() error {
	for _, field := range c.MatchFields {
		parts := strings.Split(field, ".")
		if parts[0] != bodyField && parts[0] != attributeField {
			return fmt.Errorf("a match_field must start with %s or %s", bodyField, attributeField)
		}
	}

	return nil
}