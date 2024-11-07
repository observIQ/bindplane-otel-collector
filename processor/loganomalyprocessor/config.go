package loganomalyprocessor 

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
)

var _ component.Config = (*Config)(nil)

type Config struct {
	ComparisonWindows  EvaluationWindow `mapstructure:"comparison_windows"`
	DeviationThreshold float64          `mapstructure:"deviation_threshold"`
}

type EvaluationWindow struct {
	CurrentWindow  time.Duration `mapstructure:"current_window"`
	BaselineWindow time.Duration `mapstructure:"baseline_window"`
}

// Validate checks whether the input configuration has all of the required fields for the processor.
// An error is returned if there are any invalid inputs.
func (config *Config) Validate() error {
	if config.ComparisonWindows.CurrentWindow <= 0 {
		return fmt.Errorf("current_window must be greater than 0, got %v",
			config.ComparisonWindows.CurrentWindow)
	}

	if config.ComparisonWindows.BaselineWindow <= config.ComparisonWindows.CurrentWindow {
		return fmt.Errorf("baseline_window (%v) must be greater than current_window (%v)",
			config.ComparisonWindows.BaselineWindow,
			config.ComparisonWindows.CurrentWindow)
	}

	if config.DeviationThreshold <= 0 || config.DeviationThreshold > 100 {
		return fmt.Errorf("deviation_threshold must be between 0 and 100, got %f",
			config.DeviationThreshold)
	}

	return nil
}
