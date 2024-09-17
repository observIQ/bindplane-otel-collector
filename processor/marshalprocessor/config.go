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

// Package marshalprocessor provides a processor that marshals logs to a specified format.
package marshalprocessor

import (
	"errors"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/multierr"
)

var errInvalidMarshalTo = errors.New("marshal_to must be JSON or KV")
var errXMLNotSupported = errors.New("XML not yet supported")
var errKVSeparatorsEqual = errors.New("kv_separator and kv_pair_separator must be different")
var errMapKVSeparatorsEqual = errors.New("map_kv_separator and map_kv_pair_separator must be different")

const (
	defaultMarshalTo          = "JSON"
	defaultKVSeparator        = '='
	defaultKVPairSeparator    = ' '
	defaultMapKVSeparator     = '='
	defaultMapKVPairSeparator = ','
)

// Config is the configuration for the processor
type Config struct {
	MarshalTo          string `mapstructure:"marshal_to"` // MarshalTo is either JSON or KV
	KVSeparator        rune   `mapstructure:"kv_separator"`
	KVPairSeparator    rune   `mapstructure:"kv_pair_separator"`
	MapKVSeparator     rune   `mapstructure:"map_kv_separator"`
	MapKVPairSeparator rune   `mapstructure:"map_kv_pair_separator"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	var errs error

	// Validate MarshalTo choice
	switch strings.ToUpper(cfg.MarshalTo) {
	case "JSON":
	case "XML":
		errs = multierr.Append(errs, errXMLNotSupported)
	case "KV":
		// Validate KV separators, which must be different from each other
		if cfg.KVSeparator == cfg.KVPairSeparator && cfg.KVSeparator != 0 {
			errs = multierr.Append(errs, errKVSeparatorsEqual)
		}
		// Validate Map KV separators, which must be different from each other (but can match KV separators)
		if cfg.MapKVSeparator == cfg.MapKVPairSeparator && cfg.MapKVSeparator != 0 {
			errs = multierr.Append(errs, errMapKVSeparatorsEqual)
		}
	default:
		errs = multierr.Append(errs, errInvalidMarshalTo)
	}

	return errs
}

// createDefaultConfig returns the default config for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		MarshalTo:          defaultMarshalTo,
		KVSeparator:        defaultKVSeparator,
		KVPairSeparator:    defaultKVPairSeparator,
		MapKVSeparator:     defaultMapKVSeparator,
		MapKVPairSeparator: defaultMapKVPairSeparator,
	}
}
