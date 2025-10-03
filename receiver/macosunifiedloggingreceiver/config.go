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

package macosunifiedloggingreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver"

import (
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"
)

// Config defines configuration for the macOS Unified Logging receiver
type Config struct {
	fileconsumer.Config `mapstructure:",squash"`
	adapter.BaseConfig  `mapstructure:",squash"`

	// Encoding specifies the encoding extension to use for decoding traceV3 files.
	// This should reference an encoding extension ID (e.g., "macos_unified_logging_encoding").
	Encoding string `mapstructure:"encoding_extension"`

	// TraceV3Paths specifies an alternate path to TraceV3 files. This can be a single file path or a glob pattern (e.g., "/path/to/logs/*.tracev3").
	TraceV3Paths []string `mapstructure:"tracev3_paths"`

	// TimesyncPaths specifies the path to timesync files for accurate timestamp conversion. This should point to the timesync directory (e.g., "/path/to/logs.logarchive/timesync/*.timesync").
	TimesyncPaths []string `mapstructure:"timesync_paths"`

	// UUIDTextPaths specifies the path to UUID text files for accurate message parsing. This should point to the UUID text directory (e.g., "/path/to/logs.logarchive/*/*").
	UUIDTextPaths []string `mapstructure:"uuidtext_paths"`

	// DSCPaths specifies the path to DSC (Dynamic Shared Cache) files for shared string parsing. This should point to the DSC directory (e.g., "/path/to/logs.logarchive/dsc/*").
	DSCPaths []string `mapstructure:"dsc_paths"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// getFileConsumerConfig returns a file consumer config with proper encoding settings
func (cfg *Config) getFileConsumerConfig() fileconsumer.Config {
	fcConfig := cfg.Config

	// If TraceV3Paths is specified, use it to override the include patterns
	if len(cfg.TraceV3Paths) > 0 {
		fcConfig.Include = cfg.TraceV3Paths
	}

	// Set file consumer encoding to "nop" since we'll handle decoding via the extension
	fcConfig.Encoding = "nop"
	// 64MiB. In practice, macOS rotates the file at a max of 10MiB.
	// The entire file needs to be read in at once to ensure binary data is decoded correctly.
	fcConfig.MaxLogSize = 1024 * 1024 * 64
	// Enable file path attributes so we can access the file paths in the consume function
	fcConfig.IncludeFilePath = true
	fcConfig.IncludeFileName = true

	return fcConfig
}

// Validate checks the receiver configuration is valid
func (cfg Config) Validate() error {
	if cfg.Encoding != "macosunifiedlogencoding" {
		return errors.New("encoding_extension must be macosunifiedlogencoding for macOS Unified Logging receiver")
	}

	return nil
}
