// Copyright observIQ, Inc.
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

package threatenrichmentprocessor

import (
	"fmt"
	"strings"

	"github.com/observiq/bindplane-otel-collector/internal/filter"
	"go.opentelemetry.io/collector/component"
)

// FilterConfig is the configuration for which filter algorithm to use and its parameters.
// Set Kind to one of: bloom, cuckoo, scalable_cuckoo, vacuum. Other fields apply per kind.
type FilterConfig struct {
	Kind string `mapstructure:"kind"`

	// Bloom: estimated number of elements and target false positive rate.
	EstimatedCount    uint    `mapstructure:"estimated_count"`
	FalsePositiveRate float64 `mapstructure:"false_positive_rate"`
	MaxEstimatedCount uint    `mapstructure:"max_estimated_count"`

	// Cuckoo / Vacuum: capacity (expected number of elements).
	Capacity uint `mapstructure:"capacity"`

	// ScalableCuckoo: initial capacity and load factor (0 = use library defaults).
	InitialCapacity uint    `mapstructure:"initial_capacity"`
	LoadFactor      float32 `mapstructure:"load_factor"`

	// Vacuum: fingerprint size in bits (default 8).
	FingerprintBits uint `mapstructure:"fingerprint_bits"`
}

func filterKindOf(fc FilterConfig) filter.Kind {
	return filter.Kind(strings.ToLower(strings.TrimSpace(fc.Kind)))
}

// toFilterConfig returns the internal filter package config for the selected kind.
func (c *FilterConfig) toFilterConfig() (filter.FilterConfig, error) {
	kind := filterKindOf(*c)
	switch kind {
	case filter.KindBloom:
		return filter.BloomOptions{
			EstimatedCount:    c.EstimatedCount,
			FalsePositiveRate: c.FalsePositiveRate,
			MaxEstimatedCount: c.MaxEstimatedCount,
		}, nil
	case filter.KindCuckoo:
		return filter.CuckooOptions{Capacity: c.Capacity}, nil
	case filter.KindScalableCuckoo:
		return filter.ScalableCuckooOptions{
			InitialCapacity: c.InitialCapacity,
			LoadFactor:      c.LoadFactor,
		}, nil
	case filter.KindVacuum:
		return filter.VacuumOptions{
			Capacity:        c.Capacity,
			FingerprintBits: c.FingerprintBits,
		}, nil
	default:
		return nil, fmt.Errorf("filter kind %q is not one of: bloom, cuckoo, scalable_cuckoo, vacuum", strings.TrimSpace(c.Kind))
	}
}

// Rule defines one indicator type: a named filter (lookup set) and the log fields to check against it.
// Each rule has its own indicator file and list of attribute keys (or "body") to look up in that filter.
type Rule struct {
	// Name identifies this indicator type (e.g. "ips", "domains"); used in threat.rule when matched.
	Name string `mapstructure:"name"`

	// IndicatorFile is the path to values for this rule (e.g. malicious IPs), one per line or JSON array of strings.
	IndicatorFile string `mapstructure:"indicator_file"`

	// LookupFields are log attribute keys to check. The value of each field is looked up in this rule's filter.
	// Use "body" to use the log body as the lookup value.
	LookupFields []string `mapstructure:"lookup_fields"`

	// Filter optionally overrides the default filter algorithm for this rule (e.g. different capacity per type).
	Filter *FilterConfig `mapstructure:"filter"`
}

// Config is the configuration for the threat enrichment processor.
type Config struct {
	// StorageID is the ID of the storage extension to use for persisting state.
	StorageID *component.ID `mapstructure:"storage"`

	// Filter is the default filter algorithm and parameters. Each rule uses this unless it sets rule.filter.
	Filter FilterConfig `mapstructure:"filter"`

	// Rules defines multiple indicator types. Each rule has its own lookup set (indicator_file) and
	// list of fields to check. For each log record, the processor gets the value from each lookup
	// field and checks it in that rule's filter; if found, it adds threat.matched and threat.rule.
	Rules []Rule `mapstructure:"rules"`
}

func normalizeLookupFields(ruleIdx int, ruleName string, fields []string) ([]string, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("rules[%d] (%q): at least one lookup_fields entry is required", ruleIdx, ruleName)
	}
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{})
	for _, raw := range fields {
		t := strings.TrimSpace(raw)
		if t == "" {
			return nil, fmt.Errorf("rules[%d] (%q): lookup_fields entry is empty", ruleIdx, ruleName)
		}
		if _, dup := seen[t]; dup {
			return nil, fmt.Errorf("rules[%d] (%q): duplicate lookup_fields entry %q", ruleIdx, ruleName, t)
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out, nil
}

// Validate validates the processor configuration and normalizes string fields (trim whitespace).
func (cfg *Config) Validate() error {
	cfg.Filter.Kind = strings.TrimSpace(cfg.Filter.Kind)
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		r.Name = strings.TrimSpace(r.Name)
		r.IndicatorFile = strings.TrimSpace(r.IndicatorFile)
		lookup, err := normalizeLookupFields(i, r.Name, r.LookupFields)
		if err != nil {
			return err
		}
		r.LookupFields = lookup
		if r.Filter != nil {
			r.Filter.Kind = strings.TrimSpace(r.Filter.Kind)
		}
	}

	if cfg.Filter.Kind == "" {
		return fmt.Errorf("filter.kind is required (bloom, cuckoo, scalable_cuckoo, vacuum)")
	}
	if _, err := cfg.Filter.toFilterConfig(); err != nil {
		return err
	}
	switch filterKindOf(cfg.Filter) {
	case filter.KindBloom:
		if cfg.Filter.EstimatedCount == 0 {
			return fmt.Errorf("filter.estimated_count is required for bloom filter")
		}
		if cfg.Filter.FalsePositiveRate <= 0 || cfg.Filter.FalsePositiveRate >= 1 {
			return fmt.Errorf("filter.false_positive_rate must be between 0 and 1 for bloom filter")
		}
	case filter.KindCuckoo, filter.KindVacuum:
		if cfg.Filter.Capacity == 0 {
			return fmt.Errorf("filter.capacity is required for %s filter", filterKindOf(cfg.Filter))
		}
	}
	if len(cfg.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}
	seenNames := make(map[string]struct{})
	for i, r := range cfg.Rules {
		if r.Name == "" {
			return fmt.Errorf("rules[%d]: name is required", i)
		}
		if _, dup := seenNames[r.Name]; dup {
			return fmt.Errorf("rules[%d]: duplicate rule name %q", i, r.Name)
		}
		seenNames[r.Name] = struct{}{}
		if r.IndicatorFile == "" {
			return fmt.Errorf("rules[%d] (%q): indicator_file is required", i, r.Name)
		}
		if r.Filter != nil {
			if r.Filter.Kind == "" {
				return fmt.Errorf("rules[%d] (%q): filter.kind is required when filter is set", i, r.Name)
			}
			if _, err := r.Filter.toFilterConfig(); err != nil {
				return fmt.Errorf("rules[%d] (%q): %w", i, r.Name, err)
			}
			switch filterKindOf(*r.Filter) {
			case filter.KindBloom:
				if r.Filter.EstimatedCount == 0 {
					return fmt.Errorf("rules[%d] (%q): filter.estimated_count required for bloom", i, r.Name)
				}
				if r.Filter.FalsePositiveRate <= 0 || r.Filter.FalsePositiveRate >= 1 {
					return fmt.Errorf("rules[%d] (%q): filter.false_positive_rate must be in (0,1)", i, r.Name)
				}
			case filter.KindCuckoo, filter.KindVacuum:
				if r.Filter.Capacity == 0 {
					return fmt.Errorf("rules[%d] (%q): filter.capacity required for %s", i, r.Name, filterKindOf(*r.Filter))
				}
			}
		}
	}
	return nil
}
