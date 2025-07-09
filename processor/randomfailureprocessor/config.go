// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package randomfailureprocessor provides a processor that randomly fails with a user-configured probability.
package randomfailureprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

var (
	errInvalidFailureRate  = errors.New("failure_rate must be between 0 and 1")
	errInvalidErrorMessage = errors.New("error_message must be a non-empty string")
)

var defaultErrorMessage = "random failure"
var defaultFailureRate = 0.5

// Config is the config of the processor.
type Config struct {
	// FailureRate is the rate at which failures will occur.
	// This is a float between 0 and 1.
	// 0.5 means 50% of the time, a failure will occur.
	// 1.0 means 100% of the time, a failure will occur.
	// 0.0 means 0% of the time, a failure will occur.
	// Default is 0.5.
	FailureRate float64 `mapstructure:"failure_rate"`

	// ErrorMessage is the message that will be returned when a failure occurs.
	// Default is "random failure".
	ErrorMessage string `mapstructure:"error_message"`
}

func createDefaultConfig() component.Config {
	return &Config{
		FailureRate:  defaultFailureRate,
		ErrorMessage: defaultErrorMessage,
	}
}

// Validate validates the processor configuration
func (c Config) Validate() error {
	if c.FailureRate < 0 || c.FailureRate > 1 {
		return errInvalidFailureRate
	}

	if c.ErrorMessage == "" {
		return errInvalidErrorMessage
	}

	return nil
}
