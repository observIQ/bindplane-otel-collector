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

package pcapreceiver

import (
	"fmt"
	"strings"
)

const (
	minSnapLen = 64
	maxSnapLen = 65535
)

// Validate checks the Config is valid
func (cfg *Config) Validate() error {
	// Validate interface is specified
	if cfg.Interface == "" {
		return fmt.Errorf("interface must be specified")
	}

	// Validate interface name for shell injection
	if err := validateInterfaceName(cfg.Interface); err != nil {
		return fmt.Errorf("invalid interface: %w", err)
	}

	// Validate filter for shell injection
	if cfg.Filter != "" {
		if err := validateFilter(cfg.Filter); err != nil {
			return fmt.Errorf("invalid filter: %w", err)
		}
	}

	// Validate snaplen if specified
	if cfg.SnapLen != 0 {
		if cfg.SnapLen < minSnapLen || cfg.SnapLen > maxSnapLen {
			return fmt.Errorf("snaplen must be between %d and %d", minSnapLen, maxSnapLen)
		}
	}

	return nil
}

// validateFilter validates the BPF filter to prevent shell injection
// BPF filters can have parentheses, spaces, etc., but not command injection chars
func validateFilter(filter string) error {
	// Dangerous characters for command injection
	// Note: BPF filters legitimately use spaces, parentheses, brackets, equals, etc.
	dangerousChars := []string{
		";",  // Command separator
		"|",  // Single pipe (BPF uses 'or', not '|')
		"&",  // Single ampersand (BPF uses 'and', not '&')
		"$",  // Variable expansion
		"`",  // Command substitution
		"\n", // Newline
		"\r", // Carriage return
		">>", // Append redirect
		"<<", // Here document
		"&&", // Logical AND (command chaining)
		"||", // Logical OR (command chaining)
	}

	for _, char := range dangerousChars {
		if strings.Contains(filter, char) {
			return fmt.Errorf("invalid character %q in filter expression", char)
		}
	}

	return nil
}
