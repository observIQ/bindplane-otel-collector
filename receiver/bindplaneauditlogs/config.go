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

// Package bindplaneauditlogs provides a receiver that receives telemetry from an BindPlane audit logs.
package bindplaneauditlogs // import "github.com/observiq/bindplane-otel-collector/receiver/bindplaneauditlogs"

import (
	"errors"
	"time"
)

// Config defines the configuration for the BindPlane audit logs receiver
type Config struct {
	// BindplaneURL is the URL of the BindPlane instance
	BindplaneURL string `mapstructure:"bindplane_url"`

	// APIKey is the authentication key for accessing BindPlane audit logs
	APIKey string `mapstructure:"api_key"`

	// PollInterval is the interval at which the receiver polls for new audit logs
	// Default: 10 seconds
	PollInterval time.Duration `mapstructure:"poll_interval"`
}

// Validate ensures the config is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("api_key cannot be empty")
	}

	if c.BindplaneURL == "" {
		return errors.New("bindplane_url cannot be empty")
	}

	if c.PollInterval < 10*time.Second || c.PollInterval > 24*time.Hour {
		return errors.New("poll_interval must be between 10 seconds and 24 hours")
	}

	return nil
}
