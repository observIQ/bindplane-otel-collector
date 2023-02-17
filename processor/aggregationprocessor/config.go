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

const metricNameKey = "metric_name"

// Config is the configuration for the processor
type Config struct {
	Interval time.Duration `mapstructure:"interval"`
	// Include is a regex that must match the metric name for it to be sampled.
	// Otherwise, the metric is passed through.
	Include string `mapstructure:"include"`
	// List of aggregations for the metric
	Aggregations []aggregate.AggregationType `mapstructure:"aggregations"`
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

	seenTypes := map[aggregate.AggregationType]struct{}{}
	for _, a := range cfg.Aggregations {
		if !a.Valid() {
			return fmt.Errorf("invalid aggregate type for `type`: %s", a)
		}
		if _, seen := seenTypes[a]; seen {
			return fmt.Errorf("each aggregation type can only be specified once (%s specified more than once)", a)
		}
		seenTypes[a] = struct{}{}
	}

	return nil
}

// AggregationTypes gets the default aggregation configs if none were specified, otherwise the specified aggregation configs
func (cfg Config) AggregationTypes() []aggregate.AggregationType {
	if cfg.Aggregations == nil {
		// fallback to default
		return []aggregate.AggregationType{
			aggregate.MinType,
			aggregate.MaxType,
			aggregate.AvgType,
		}
	}

	return cfg.Aggregations
}
