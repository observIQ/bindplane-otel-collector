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

// Package aggregationprocessor provides a processor that samples pdata base level objects.
package aggregationprocessor

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/observiq/observiq-otel-collector/processor/aggregationprocessor/internal/aggregate"
)

// Config is the configuration for the processor
type Config struct {
	Interval time.Duration `mapstructure:"interval"`
	// Include is a regex that must match the metric name for it to be sampled.
	// Otherwise, the metric is passed through.
	Include string `mapstructure:"include"`
	// List of aggregations for the metric
	Aggregations []AggregateConfig `mapstructure:"aggregations"`
}

// AggregateConfig is a config that specifies which aggregations to perform for each incoming metric
type AggregateConfig struct {
	// Type of aggregation
	Type aggregate.AggregationType `mapstructure:"type"`
	// MetricName is the name for the re-emitted metric. Defaults to `$0` (this is what is matched by the regex)
	MetricName string `mapstructure:"metric_name"`
}

// Validate validate the config, returning an error explaining why it isn't if the config is invalid.
func (a AggregateConfig) Validate() error {
	if !a.Type.Valid() {
		return fmt.Errorf("invalid aggregate type for `type`: %s", a.Type)
	}

	return nil
}

// MetricNameString returns the configured name for the emitted metric, or "$0" if none was specified.
func (a AggregateConfig) MetricNameString() string {
	if a.MetricName != "" {
		return a.MetricName
	}

	return "$0"
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	if _, err := regexp.Compile(cfg.Include); err != nil {
		return fmt.Errorf("`include` regex must be valid: %w", err)
	}

	if cfg.Interval <= 0 {
		return errors.New("aggregation interval must be positive")
	}

	// don't check aggregations if using defaults
	if cfg.Aggregations == nil {
		return nil
	}

	if len(cfg.Aggregations) == 0 {
		return errors.New("at least one aggregation must be specified")
	}

	for _, a := range cfg.Aggregations {
		if err := a.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AggregationConfigs gets the default aggregation configs if none were specified, otherwise the specified aggregation configs
func (cfg Config) AggregationConfigs() []AggregateConfig {
	if cfg.Aggregations == nil {
		// fallback to defaults
		return []AggregateConfig{
			{
				Type:       aggregate.MinType,
				MetricName: "$0.min",
			},
			{
				Type:       aggregate.MaxType,
				MetricName: "$0.max",
			},
			{
				Type:       aggregate.AvgType,
				MetricName: "$0.avg",
			},
		}
	}

	return cfg.Aggregations
}
