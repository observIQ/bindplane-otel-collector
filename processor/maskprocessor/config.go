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

// Package maskprocessor provides a processor that masks data.
package maskprocessor

import (
	"errors"
	"fmt"
	"regexp"
)

var errNoRules = errors.New("no rules defined")

// Config is the configuration for the processor.
type Config struct {
	// Rules are the rules used to mask values.
	Rules map[string]string `mapstructure:"rules"`

	// Exclude is a list of fields to exclude when masking.
	Exclude []string `mapstructure:"exclude"`
}

// CompileRules compiles the rules defined in the config.
func (cfg Config) CompileRules() (map[string]*regexp.Regexp, error) {
	rules := make(map[string]*regexp.Regexp)
	for key, expr := range cfg.Rules {
		rule, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("rule '%s' does not compile as valid regex", key)
		}

		mask := fmt.Sprintf("[masked_%s]", key)
		rules[mask] = rule
	}
	return rules, nil
}

// Validate validates the processor configuration.
func (cfg Config) Validate() error {
	if len(cfg.Rules) == 0 {
		return errNoRules
	}

	if _, err := cfg.CompileRules(); err != nil {
		return err
	}

	return nil
}
