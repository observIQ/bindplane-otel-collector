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

// Package samplingprocessor provides a processor that samples pdata base level objects.
package samplingprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/config"
)

var errInvalidDropRatio = errors.New("drop_ratio must be between 0.0 and 1.0")

// Config is the configuration for the processor
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"`

	// DropRatio is the ratio of payloads that are dropped. Values between 0.0 and 1.0 are valid.
	DropRatio float64 `mapstructure:"drop_ratio"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	// Validate drop ratio
	if cfg.DropRatio < 0.0 || cfg.DropRatio > 1.0 {
		return errInvalidDropRatio
	}

	return nil
}
