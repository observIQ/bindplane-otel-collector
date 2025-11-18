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

package kandjireceiver // import "github.com/observiq/bindplane-otel-collector/receiver/m365receiver"

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

type LogsConfig struct {
	PollInterval time.Duration `mapstructure:"poll_interval"`
}

type Config struct {
	ClientConfig confighttp.ClientConfig `mapstructure:",squash"`

	Region         string                    `mapstructure:"region"`
	SubDomain      string                    `mapstructure:"subdomain"`
	ApiKey         string                    `mapstructure:"api_key"`
	BaseHost       string                    `mapstructure:"base_host"`
	EndpointParams map[string]map[string]any `mapstructure:"endpoint_params"`

	Logs      LogsConfig    `mapstructure:"logs"`
	StorageID *component.ID `mapstructure:"storage_id"`
}

// Validate validates the configuration by checking for missing or invalid fields.
func (c *Config) Validate() error {

	if strings.TrimSpace(c.SubDomain) == "" {
		return fmt.Errorf("missing subdomain; required (e.g. 'acme' in https://acme.kandji.io)")
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(c.SubDomain) {
		return fmt.Errorf(
			"invalid subdomain %q; must contain only letters, numbers, and hyphens",
			c.SubDomain,
		)
	}

	if c.Region != "" && c.Region != "US" && c.Region != "EU" {
		return fmt.Errorf(`invalid region %q; must be "US", "EU", or empty`, c.Region)
	}

	if strings.TrimSpace(c.ApiKey) == "" {
		return fmt.Errorf("missing api_key; required Kandji API token")
	}

	if len(c.ApiKey) < 20 {
		return fmt.Errorf("api_key appears too short to be a valid Kandji API token")
	}

	if c.BaseHost != "" {
		if !regexp.MustCompile(`^[a-zA-Z0-9.-]+$`).MatchString(c.BaseHost) {
			return fmt.Errorf("base_host %q is not a valid hostname", c.BaseHost)
		}
	}

	for epStr, params := range c.EndpointParams {
		ep := KandjiEndpoint(epStr)

		spec, ok := EndpointRegistry[ep]
		if !ok {
			return fmt.Errorf(
				"endpoint_params: unknown endpoint %q; must match a KandjiEndpoint key in the registry",
				epStr,
			)
		}

		if err := ValidateParams(ep, params); err != nil {
			return fmt.Errorf(
				"endpoint_params validation failed for endpoint %q: %w",
				spec.Path,
				err,
			)
		}
	}

	return nil
}
