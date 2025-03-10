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

// Config defines the configuration for the Bindplane audit logs receiver
type Config struct {

	// APIKey is the authentication key for accessing BindPlane audit logs
	APIKey string `mapstructure:"api_key"`

	// Scheme is the URL scheme used to connect to the BindPlane audit logs API
	Scheme string `mapstructure:"scheme"`

	// ClientConfig is the configuration for the HTTP client
	confighttp.ClientConfig `mapstructure:",squash"`

	// PollInterval is the interval at which the receiver polls for new audit logs
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// bindplaneURL is the URL to the BindPlane audit logs API. Taken from the client config endpoint.
	bindplaneURL *url.URL
}

// Validate ensures the config is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("api_key cannot be empty")
	}

	if c.Scheme == "" {
		return errors.New("scheme cannot be empty")
	}

	if c.Endpoint == "" {
		return errors.New("endpoint cannot be empty")
	}

	// parse the string into a url
	var err error
	c.bindplaneURL, err = url.Parse(c.Endpoint)
	if err != nil {
		return fmt.Errorf("error parsing endpoint: %w", err)
	}

	if c.bindplaneURL == nil {
		return errors.New("bindplaneURL cannot be nil")
	}

	if c.bindplaneURL.Host == "" || c.bindplaneURL.Scheme == "" {
		return errors.New("endpoint must contain a host and scheme")
	}

	if c.PollInterval < 10*time.Second || c.PollInterval > 24*time.Hour {
		return errors.New("poll_interval must be between 10 seconds and 24 hours")
	}

	return nil
}
