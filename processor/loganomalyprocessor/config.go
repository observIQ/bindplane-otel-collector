// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loganomalyprocessor

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
)

var _ component.Config = (*Config)(nil)
var defaultOpAMPExtensionID = component.MustNewID("opamp")

type Config struct {
	// How often to take measurements
	SampleInterval time.Duration `mapstructure:"sample_interval"`
	// How long to keep samples
	MaxWindowAge time.Duration `mapstructure:"max_window_age"`
	// Thresholds for anomaly detection
	ZScoreThreshold float64 `mapstructure:"zscore_threshold"`
	MADThreshold    float64 `mapstructure:"mad_threshold"`
	// Maximum number of samples to keep (emergency limit)
	EmergencyMaxSize int `mapstructure:"emergency_max_size"`
	// OpAMP arguments
	Enabled           bool         `mapstructure:"enabled"`
	OpAMP             component.ID `mapstructure:"opamp"`
	AnomalyBufferSize int          `mapstructure:"anomaly_buffer_size"`
}

// Validate checks whether the input configuration has all of the required fields for the processor.
// An error is returned if there are any invalid inputs.
func (config *Config) Validate() error {
	if config.SampleInterval <= 0 {
		return fmt.Errorf("sample_interval must be positive, got %v", config.SampleInterval)
	}
	if config.SampleInterval < time.Minute {
		return fmt.Errorf("sample_interval must be at least 1 minute, got %v", config.SampleInterval)
	}
	if config.SampleInterval > time.Hour {
		return fmt.Errorf("sample_interval must not exceed 1 hour, got %v", config.SampleInterval)
	}
	if config.MaxWindowAge <= 0 {
		return fmt.Errorf("max_window_age must be positive, got %v", config.MaxWindowAge)
	}
	if config.MaxWindowAge < time.Hour {
		return fmt.Errorf("max_window_age must be at least 1 hour, got %v", config.MaxWindowAge)
	}

	if config.MaxWindowAge < config.SampleInterval*10 {
		return fmt.Errorf("max_window_age (%v) must be at least 10 times larger than sample_interval (%v)",
			config.MaxWindowAge, config.SampleInterval)
	}

	if config.ZScoreThreshold <= 0 {
		return fmt.Errorf("zscore_threshold must be positive, got %v", config.ZScoreThreshold)
	}

	if config.MADThreshold <= 0 {
		return fmt.Errorf("mad_threshold must be positive, got %v", config.MADThreshold)
	}

	if config.EmergencyMaxSize <= 0 {
		return fmt.Errorf("emergency_max_size must be positive, got %v", config.EmergencyMaxSize)
	}
	return nil
}
