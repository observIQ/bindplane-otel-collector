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

//go:build windows

package pcapreceiver

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
)

// createDefaultConfig creates the default configuration for the receiver on Windows
func createDefaultConfig() component.Config {
	return &Config{
		Interface:      "1",   // Default Windows interface (numeric ID, typically the first interface)
		SnapLen:        65535, // Maximum snapshot length
		Promiscuous:    true,  // Enable promiscuous mode by default
		AddAttributes:  true,  // Add attributes by default
		ExecutablePath: "dumpcap",
	}
}

// validateInterfaceName validates the interface name to prevent shell injection on Windows
// Windows interface names can be:
// - Numeric indices: "1", "2"
// - Device paths: "\Device\NPF_{GUID}"
// - Friendly names: "Ethernet", "Local Area Connection"
func validateInterfaceName(iface string) error {
	// Windows: Allow backslashes, braces, and spaces (for device paths and friendly names)
	// But still block command injection characters
	dangerousChars := []string{
		";",  // Command separator
		"|",  // Pipe
		"&",  // Background/AND
		"$",  // Variable expansion
		"`",  // Command substitution
		"\n", // Newline
		"\r", // Carriage return
		">",  // Redirect
		"<",  // Redirect
		"\"", // Quote
		"'",  // Quote
	}

	for _, char := range dangerousChars {
		if strings.Contains(iface, char) {
			return fmt.Errorf("invalid character %q in interface name", char)
		}
	}

	return nil
}
