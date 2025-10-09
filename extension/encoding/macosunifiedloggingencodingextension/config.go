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

package macosunifiedloggingencodingextension // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"

// Config defines the configuration for the macOS Unified Logging encoding extension.
// This extension is designed to decode macOS Unified Logging binary files.
type Config struct {
	// DebugMode enables additional debug information during decoding
	DebugMode bool `mapstructure:"debug_mode"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// No validation needed for basic config
	return nil
}
