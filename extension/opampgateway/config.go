// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opampgateway

import (
	"errors"
	"net/url"

	"go.opentelemetry.io/collector/config/confighttp"
)

// Config holds the configuration for the OpAMP gateway extension.
type Config struct {
	UpstreamOpAMPAddress string                  `mapstructure:"upstream_opamp_address"`
	SecretKey            string                  `mapstructure:"secret_key"`
	UpstreamConnections  int                     `mapstructure:"upstream_connections"`
	OpAMPServer          confighttp.ServerConfig `mapstructure:"opamp_server"`
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.UpstreamOpAMPAddress == "" {
		return errors.New("upstream_opamp_address must be specified")
	}
	u, err := url.Parse(c.UpstreamOpAMPAddress)
	if err != nil {
		return errors.New("upstream_opamp_address is not a valid URL")
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return errors.New("upstream_opamp_address must use ws:// or wss:// scheme")
	}

	if c.UpstreamConnections < 1 {
		return errors.New("upstream_connections must be at least 1")
	}

	if c.OpAMPServer.NetAddr.Endpoint == "" {
		return errors.New("opamp_server endpoint must be specified")
	}

	return nil
}
