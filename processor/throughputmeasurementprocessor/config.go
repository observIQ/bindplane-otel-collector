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

// Package throughputmeasurementprocessor provides a processor that measure the amount of otlp structures flowing through it
package throughputmeasurementprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

var errInvalidSamplingRatio = errors.New("sampling_ratio must be between 0.0 and 1.0")

// Config is the configuration for the processor
type Config struct {
	// Enable controls whether measurements are taken or not.
	Enabled bool `mapstructure:"enabled"`

	// SamplingRatio is the ratio of payloads that are measured. Values between 0.0 and 1.0 are valid.
	SamplingRatio float64 `mapstructure:"sampling_ratio"`

	// Bindplane extension to use in order to report metrics. Optional.
	BindplaneExtension component.ID `mapstructure:"bindplane_extension"`

	// Extra labels to add to measurements and associate with emitted metrics
	ExtraLabels map[string]string `mapstructure:"extra_labels"`

	// When true, for logs, the processor will measure the raw bytes of the payload in addition to the protobuf size. This is more expensive but provides raw measurements if designated.
	MeasureLogRawBytes bool `mapstructure:"measure_log_raw_bytes"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	// Processor not enabled no validation needed
	if !cfg.Enabled {
		return nil
	}

	// Validate sampling ration
	if cfg.SamplingRatio < 0.0 || cfg.SamplingRatio > 1.0 {
		return errInvalidSamplingRatio
	}

	return nil
}
