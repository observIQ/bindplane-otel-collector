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

// Package oktareceiver provides a receiver that receives telemetry from an Okta domain.
package oktareceiver // import "github.com/observiq/bindplane-agent/receiver/oktareceiver"

import (
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/collector/config/configopaque"
)

var (
	defaultPollInterval = time.Minute

	// OktaTimeFormat ISO 8601 Format
	OktaTimeFormat = "2006-01-02T15:04:05Z"
)

// Config defines the configuration for an Okta receiver
type Config struct {
	// Domain Okta Domain (no https://  -  ex: observiq.okta.com)
	Domain string `mapstructure:"okta_domain"`

	// APIToken Okta API Token
	APIToken configopaque.String `mapstructure:"api_token"`

	// PollInterval The interval at which the Okta API is scanned for Logs
	// Must be in the range [1 second - 24 hours]
	PollInterval time.Duration `mapstructure:"poll_interval"`
}

var (
	errNoDomain            = errors.New("okta_domain must be specified")
	errInvalidDomain       = errors.New("invalid okta_domain, do not include https://")
	errNoAPIToken          = errors.New("api_token must be specified")
	errInvalidPollInterval = errors.New("invalid poll_interval, it must be within the range of [1 second - 24 hours]")
)

// Validate ensures an Okta receiver config is correct
func (c *Config) Validate() error {
	if c.Domain == "" {
		return errNoDomain
	}

	if strings.HasPrefix(c.Domain, "https://") || strings.HasPrefix(c.Domain, "http://") {
		return errInvalidDomain
	}

	if string(c.APIToken) == "" {
		return errNoAPIToken
	}

	if c.PollInterval < time.Second || c.PollInterval > 24*time.Hour {
		return errInvalidPollInterval
	}

	return nil
}
