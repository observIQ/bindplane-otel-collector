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

package windowseventtracereceiver

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Config is the configuration for the windows event trace receiver.
type Config struct {
	// SessionName is the name for the ETW session.
	SessionName string `mapstructure:"session_name"`

	// Providers is a list of providers to create subscriptions for.
	Providers []Provider `mapstructure:"providers"`

	// Attributes is a list of attributes to add to the logs.
	Attributes map[string]string `mapstructure:"attributes"`

	// BufferSize is the size of the buffer to use for the ETW session.
	BufferSize int `mapstructure:"buffer_size"`
}

// Provider is a provider to create a session
type Provider struct {
	Name string `mapstructure:"name"`
}

func createDefaultConfig() component.Config {
	return &Config{
		SessionName: "OtelCollectorETW",
		BufferSize:  64,
		Providers:   []Provider{},
	}
}

// Validate validates the config.
func (cfg *Config) Validate() error {
	if cfg.SessionName == "" {
		return fmt.Errorf("session_name cannot be empty")
	}

	if len(cfg.Providers) == 0 {
		return fmt.Errorf("providers cannot be empty")
	}

	if cfg.BufferSize <= 0 {
		return fmt.Errorf("buffer_size must be greater than 0")
	}

	return nil
}
