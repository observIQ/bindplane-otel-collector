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
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
)

// URLConfig is a wrapper for url.URL that implements proper unmarshaling
type URLConfig struct {
	URL *url.URL `mapstructure:"url"`
}

// Config defines the configuration for the Bindplane audit logs receiver
type Config struct {
	// BindplaneURLString is the URL string of the Bindplane instance
	BindplaneURLString string `mapstructure:"bindplane_url_string"`

	// APIKey is the authentication key for accessing BindPlane audit logs
	APIKey string `mapstructure:"api_key"`

	// BindplaneURL is the URL of the Bindplane instance, taken from BindplaneURLString
	BindplaneURL URLConfig `mapstructure:"bindplane_url"`

	// PollInterval is the interval at which the receiver polls for new audit logs
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// ClientConfig is the configuration for the HTTP client
	confighttp.ClientConfig `mapstructure:",squash"`
}

// Validate ensures the config is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("api_key cannot be empty")
	}

	if c.BindplaneURLString == "" {
		return errors.New("bindplane_url_string cannot be empty")
	}

	// parse the string into a url
	bindplaneURL, err := url.Parse(c.BindplaneURLString)
	if err != nil {
		return fmt.Errorf("error parsing bindplane_url_string: %w", err)
	}

	if bindplaneURL.Host == "" || bindplaneURL.Scheme == "" {
		return errors.New("bindplane_url_string must contain a host and scheme")
	}

	// Set the BindplaneURL
	c.BindplaneURL = URLConfig{URL: bindplaneURL}
	if c.BindplaneURL.URL == nil {
		return errors.New("bindplane_url must be initialized")
	}

	if c.PollInterval == 0 {
		c.PollInterval = 10 * time.Second
	} else if c.PollInterval < 10*time.Second || c.PollInterval > 24*time.Hour {
		return errors.New("poll_interval must be between 10 seconds and 24 hours")
	}

	return nil
}
