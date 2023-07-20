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

// Package metricstatsprocessor provides a processor that samples pdata base level objects.
package metricstatsprocessor

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/observiq/bindplane-agent/processor/metricstatsprocessor/internal/stats"
)

// Config is the configuration for the processor
type Config struct {
	Interval time.Duration `mapstructure:"interval"`
	// Include is a regex that must match the metric name for it to be sampled.
	// Otherwise, the metric is passed through.
	Include string `mapstructure:"include"`
	// List of stats to calculate for each metric
	Stats []stats.StatType `mapstructure:"stats"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	if _, err := regexp.Compile(cfg.Include); err != nil {
		return fmt.Errorf("`include` regex must be valid: %w", err)
	}

	if cfg.Interval <= 0 {
		return errors.New("interval must be positive")
	}

	// don't check stats if using defaults
	if cfg.Stats == nil {
		return nil
	}

	if len(cfg.Stats) == 0 {
		return errors.New("at least one statistic must be specified in `stats`")
	}

	seenTypes := map[stats.StatType]struct{}{}
	for _, a := range cfg.Stats {
		if !a.Valid() {
			return fmt.Errorf("invalid statistic type for `type`: %s", a)
		}
		if _, seen := seenTypes[a]; seen {
			return fmt.Errorf("each statistic type can only be specified once (%s specified more than once)", a)
		}
		seenTypes[a] = struct{}{}
	}

	return nil
}

// StatTypes gets the default stats to calculate if none were specified, otherwise the configured stat types
func (cfg Config) StatTypes() []stats.StatType {
	if cfg.Stats == nil {
		// fallback to default
		return []stats.StatType{
			stats.MinType,
			stats.MaxType,
			stats.AvgType,
		}
	}

	return cfg.Stats
}
