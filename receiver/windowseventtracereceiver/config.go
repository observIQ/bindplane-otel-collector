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

package windowseventtracereceiver

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	// SessionName is the name for the ETW session.
	SessionName string `mapstructure:"session_name"`

	// Providers is a list of providers to create subscriptions for.
	Providers []Provider `mapstructure:"providers"`
}

type Provider struct {
	Name string `mapstructure:"name"`
}

func createDefaultConfig() component.Config {
	return &Config{
		SessionName: "OtelCollectorETW",
		Providers: []Provider{
			// Microsoft-Windows-Kernel-File
			{Name: "{EDD08927-9CC4-4E65-B970-C2560FB5C289}"},
			// Microsoft Paint
			{Name: "{1d75856d-36a7-4ecb-a3f5-b13152222d29}"},
		},
	}
}

func (cfg *Config) Validate() error {
	if cfg.SessionName == "" {
		return fmt.Errorf("session_name cannot be empty")
	}

	if len(cfg.Providers) == 0 {
		return fmt.Errorf("providers cannot be empty")
	}

	return nil
}
