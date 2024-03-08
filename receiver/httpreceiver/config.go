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

// Package httpreceiver is a default HTTP receiver for log ingestion
package httpreceiver

import (
	"errors"
	"net"
	"path"

	"go.opentelemetry.io/collector/config/confighttp"
)

// Config defines the configuration for an HTTP receiver
type Config struct {
	Path                    string `mapstructure:"path"`
	confighttp.ServerConfig `mapstructure:",squash"`
}

var (
	errNoEndpoint  = errors.New("an endpoint must be specified")
	errBadEndpoint = errors.New("unable to split endpoint into 'host:port' pair")
	errBadPath     = errors.New("given path is malformed")
	errNoCert      = errors.New("tls was configured, but no cert file was specified")
	errNoKey       = errors.New("tls was configured, but no key file was specified")
)

// Validate ensures an HTTP receiver config is correct
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errNoEndpoint
	}

	if _, _, err := net.SplitHostPort(c.Endpoint); err != nil {
		return errBadEndpoint
	}
	if c.TLSSetting != nil {
		if c.TLSSetting.CertFile == "" && c.TLSSetting.CertPem == "" {
			return errNoCert
		}
		if c.TLSSetting.KeyFile == "" && c.TLSSetting.KeyPem == "" {
			return errNoKey
		}
	}
	if c.Path != "" {
		clean := path.Clean(c.Path)
		if c.Path != clean {
			return errBadPath
		}
	}

	return nil
}
