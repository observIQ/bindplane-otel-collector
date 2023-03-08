// Copyright  observIQ, Inc.
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

package sapnetweaverreceiver // import "github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver"

import (
	"errors"
	"fmt"
	"net/url"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/multierr"

	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/metadata"
)

// Errors for missing required config parameters.
var (
	ErrNoUsername      = errors.New("invalid config: missing username")
	ErrNoPwd           = errors.New("invalid config: missing password")
	ErrInvalidHostname = errors.New("invalid config: invalid hostname")
	ErrInvalidEndpoint = errors.New("invalid config: invalid endpoint")
)

var (
	defaultProtocol = "http://"
	defaultHost     = "localhost"
	defaultPort     = "50013"
	defaultEndpoint = fmt.Sprintf("%s%s:%s", defaultProtocol, defaultHost, defaultPort)
)

// Config defines configuration for SAP Netweaver metrics receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	configtls.TLSClientSetting              `mapstructure:"tls,omitempty"`
	confighttp.HTTPClientSettings           `mapstructure:"tls,omitempty,squash"`
	// Metrics defines which metrics to enable for the scraper
	MetricsBuilderConfig metadata.MetricsBuilderConfig `mapstructure:",squash"`
	// Endpoint string `mapstructure:"endpoint"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Profile  string `mapstructure:"profile,omitempty"`
}

// Validate validates the configuration by checking for missing or invalid fields
func (cfg *Config) Validate() error {
	var errs error
	if cfg.Username == "" {
		errs = multierr.Append(errs, ErrNoUsername)
	}

	if cfg.Password == "" {
		errs = multierr.Append(errs, ErrNoPwd)
	}

	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		errs = multierr.Append(errs, ErrInvalidEndpoint)
	}

	if u.Hostname() == "" {
		errs = multierr.Append(errs, ErrInvalidHostname)
	}

	return errs
}
