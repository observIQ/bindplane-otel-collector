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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
)

func TestConfigValidate(t *testing.T) {
	validConfig := func() *Config {
		return &Config{
			Server: ServerConfig{
				Endpoint:    "ws://localhost:4320/v1/opamp",
				Connections: 1,
			},
			Listener: confighttp.ServerConfig{
				NetAddr: confignet.AddrConfig{Endpoint: "0.0.0.0:4321"},
			},
		}
	}

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, validConfig().Validate())
	})

	t.Run("valid wss scheme", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Endpoint = "wss://localhost:4320/v1/opamp"
		require.NoError(t, cfg.Validate())
	})

	t.Run("empty upstream address", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Endpoint = ""
		err := cfg.Validate()
		require.ErrorContains(t, err, "opamp_client endpoint must be specified")
	})

	t.Run("invalid upstream address scheme", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Endpoint = "http://localhost:4320"
		err := cfg.Validate()
		require.ErrorContains(t, err, "opamp_client endpoint must use ws:// or wss:// scheme")
	})

	t.Run("upstream connections zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Connections = 0
		err := cfg.Validate()
		require.ErrorContains(t, err, "opamp_client connections must be at least 1")
	})

	t.Run("upstream connections negative", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Connections = -1
		err := cfg.Validate()
		require.ErrorContains(t, err, "opamp_client connections must be at least 1")
	})

	t.Run("empty server endpoint", func(t *testing.T) {
		cfg := validConfig()
		cfg.Listener.NetAddr.Endpoint = ""
		err := cfg.Validate()
		require.ErrorContains(t, err, "opamp_server endpoint must be specified")
	})
}
