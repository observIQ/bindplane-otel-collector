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

package throughputanomalyconnector

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
)

var _ component.Config = (*Config)(nil)

// Config defines the configuration parameters for the log anomaly detector connector.
type Config struct {
	// How often to take measurements
	AnalysisInterval time.Duration `mapstructure:"analysis_interval"`
	// MaxWindowAge defines the maximum age of samples to retain in the detection window.
	// Samples older than this duration are pruned from the analysis window.
	// This duration determines how far back the detector looks when establishing baseline behavior.
	MaxWindowAge time.Duration `mapstructure:"max_window_age"`
	// ZScoreThreshold is the number of standard deviations from the mean that a sample
	// must exceed to be considered an anomaly. A larger threshold means fewer but more
	// significant anomalies will be detected. For normally distributed data, a value
	// of 3.0 means only ~0.3% of samples would naturally fall outside this range.
	ZScoreThreshold float64 `mapstructure:"zscore_threshold"`
	// MADThreshold is the number of Median Absolute Deviations (MAD) from the median that
	// a sample must exceed to be considered an anomaly. MAD is more robust to outliers
	// than Z-score. Like ZScoreThreshold, higher values mean more extreme deviations
	// are required to trigger an anomaly.
	MADThreshold float64 `mapstructure:"mad_threshold"`
}

// Validate checks whether the input configuration has all of the required fields for the processor.
// An error is returned if there are any invalid inputs.
func (config *Config) Validate() error {
	if config.AnalysisInterval <= 0 {
		return fmt.Errorf("analysis_interval must be positive, got %v", config.AnalysisInterval)
	}
	if config.AnalysisInterval < time.Minute {
		return fmt.Errorf("analysis_interval must be at least 1 minute, got %v", config.AnalysisInterval)
	}
	if config.AnalysisInterval > time.Hour {
		return fmt.Errorf("analysis_interval must not exceed 1 hour, got %v", config.AnalysisInterval)
	}
	if config.MaxWindowAge <= 0 {
		return fmt.Errorf("max_window_age must be positive, got %v", config.MaxWindowAge)
	}
	if config.MaxWindowAge < time.Hour {
		return fmt.Errorf("max_window_age must be at least 1 hour, got %v", config.MaxWindowAge)
	}

	if config.MaxWindowAge < config.AnalysisInterval*10 {
		return fmt.Errorf("max_window_age (%v) must be at least 10 times larger than analysis_interval (%v)",
			config.MaxWindowAge, config.AnalysisInterval)
	}

	if config.ZScoreThreshold <= 0 {
		return fmt.Errorf("zscore_threshold must be positive, got %v", config.ZScoreThreshold)
	}

	if config.MADThreshold <= 0 {
		return fmt.Errorf("mad_threshold must be positive, got %v", config.MADThreshold)
	}
	return nil
}
