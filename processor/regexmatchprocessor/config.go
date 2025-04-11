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

// Package regexmatchprocessor contains the logic to match logs against a list of regexes.
package regexmatchprocessor

import (
	"errors"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher"
)

// Config is the configuration for the regex match processor.
type Config struct {
	AttributeName string               `mapstructure:"attribute_name"`
	Regexes       []matcher.NamedRegex `mapstructure:"regexes"`
}

// Validate checks the configuration for any issues.
func (c *Config) Validate() error {
	if len(c.Regexes) == 0 {
		return errors.New("at least one regex is required")
	}

	_, err := matcher.New(c.Regexes)
	if err != nil {
		return fmt.Errorf("problem with regexes: %w", err)
	}

	return nil
}
