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
	// errMissingCSV is the error for missing required field 'csv'
	errMissingCSV = errors.New("missing required field 'csv'")
	// errMissingContext is the error for missing required field 'context'
	errMissingContext = errors.New("missing required field 'context'")
	// errMissingField is the error for missing required field 'field'
	errMissingField = errors.New("missing required field 'field'")
	// errInvalidContext is the error for an invalid context
	errInvalidContext = errors.New("invalid context")
)

// Config is the configuration for the processor
type Config struct {
	CSV     string `mapstructure:"csv"`
	Context string `mapstructure:"context"`
	Field   string `mapstructure:"field"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	if cfg.CSV == "" {
		return errMissingCSV
	}

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

	return nil
}
