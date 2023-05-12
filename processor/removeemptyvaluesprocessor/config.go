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

// Package removeemptyvaluesprocessor provides a processor that removes empty values from telemetry data
package removeemptyvaluesprocessor

import (
	"bytes"
	"fmt"
)

// valid fields that can be referenced in the MapKey's field
const (
	attributesField = "attributes"
	resourceField   = "resource"
	bodyField       = "body"
)

var allFields = []string{attributesField, resourceField, bodyField}

// MapKey represents a key into a particular map (denoted by field)
type MapKey struct {
	field string
	key   string
}

// UnmarshalText unmarshals the given []byte into a MapKey.
// The format of the key is "<field>.<path-to-key>"
func (m *MapKey) UnmarshalText(text []byte) error {
	for _, field := range allFields {
		if field == string(text) {
			// this is an exact match for excluding an entire field, so just filling in
			// the field is acceptable.
			m.field = field
			return nil
		}
	}

	field, key, found := bytes.Cut(text, []byte("."))
	if !found {
		return fmt.Errorf("failed to determine field: %s", text)
	}

	if len(key) == 0 {
		return fmt.Errorf("key part of (%s) must be non-zero in length", text)
	}

	m.field = string(field)
	m.key = string(key)

	return nil
}

// Validate validates the MapKey, returning an error if it is invalid.
func (m MapKey) Validate() error {
	for _, field := range allFields {
		if field == m.field {
			return nil
		}
	}

	return fmt.Errorf("invalid field (%s), field must be body, attributes, or resource", m.field)
}

// Config is the configuration for the processor
type Config struct {
	RemoveNulls       bool     `mapstructure:"remove_nulls"`
	RemoveEmptyLists  bool     `mapstructure:"remove_empty_lists"`
	RemoveEmptyMaps   bool     `mapstructure:"remove_empty_maps"`
	EmptyStringValues []string `mapstructure:"empty_string_values"`
	ExcludeKeys       []MapKey `mapstructure:"exclude_keys"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	for i, key := range cfg.ExcludeKeys {
		if err := key.Validate(); err != nil {
			return fmt.Errorf("exclude_keys[%d]: %w", i, err)
		}
	}
	return nil
}
