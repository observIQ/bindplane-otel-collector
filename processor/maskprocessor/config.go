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

// Config is the configuration for the processor.
type Config struct {
	// Rules are the rules used to mask values.
	Rules map[string]string `mapstructure:"rules"`

	// Exclude is a list of fields to exclude when masking.
	Exclude []string `mapstructure:"exclude"`
}

// Validate validates the processor configuration.
func (cfg Config) Validate() error {
	if len(cfg.Rules) > 0 {
		_, err := compileRules(cfg.Rules)
		return err
	}

	return nil
}
