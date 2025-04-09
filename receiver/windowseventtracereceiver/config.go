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

// Package windowseventtracereceiver implements a receiver that uses the Windows Event Trace (ETW) API to collect events.
package windowseventtracereceiver

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// TraceLevelString is a string representation of the trace level.
type TraceLevelString string

const (
	// LevelVerbose is the verbose trace level.
	LevelVerbose TraceLevelString = "verbose"
	// LevelInformational is the informational trace level.
	LevelInformational TraceLevelString = "informational"
	// LevelWarning is the warning trace level.
	LevelWarning TraceLevelString = "warning"
	// LevelError is the error trace level.
	LevelError TraceLevelString = "error"
	// LevelCritical is the critical trace level.
	LevelCritical TraceLevelString = "critical"
	// LevelNone is the none trace level.
	LevelNone TraceLevelString = "none"
)

// Config is the configuration for the windows event trace receiver.
type Config struct {
	// SessionName is the name for the ETW session.
	SessionName string `mapstructure:"session_name"`

	// Providers is a list of providers to create subscriptions for.
	Providers []Provider `mapstructure:"providers"`

	// Attributes is a list of attributes to add to the logs.
	Attributes map[string]string `mapstructure:"attributes"`

	// BufferSize is the size of bytes buffer to use for creating the ETW session
	BufferSize int `mapstructure:"buffer_size"`

	// RequireAllProviders is a flag to fail if not all providers are able to be enabled.
	RequireAllProviders bool `mapstructure:"require_all_providers"`

	// RawEvents is a flag to enable raw event logging.
	Raw bool `mapstructure:"raw"`
}

// Provider is a provider to create a session
type Provider struct {
	Name            string           `mapstructure:"name"`
	Level           TraceLevelString `mapstructure:"level"`
	MatchAnyKeyword uint64           `mapstructure:"match_any_keyword"`
	MatchAllKeyword uint64           `mapstructure:"match_all_keyword"`
}

func createDefaultConfig() component.Config {
	return &Config{
		SessionName:         "OtelCollectorETW",
		BufferSize:          256,
		Providers:           []Provider{},
		RequireAllProviders: true,
		Raw:                 false,
	}
}

// Validate validates the config.
func (cfg *Config) Validate() error {
	if cfg.SessionName == "" {
		return fmt.Errorf("session_name cannot be empty")
	}

	if len(cfg.Providers) < 1 {
		return fmt.Errorf("providers cannot be empty")
	}

	for _, provider := range cfg.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider name cannot be empty; it must be a valid ETW provider name or GUID")
		}
	}

	if cfg.BufferSize <= 0 {
		return fmt.Errorf("buffer_size must be greater than 0")
	}

	return nil
}
