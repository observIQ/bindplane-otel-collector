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

package sapnetweaverreceiver // import "github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver"

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.uber.org/multierr"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		endpoint    string
		username    string
		password    string
		errExpected bool
		errText     string
	}{
		{
			desc:        "Missing username and password and invalid hostname",
			endpoint:    "localhost:50013",
			errExpected: true,
			errText:     multierr.Combine(ErrNoUsername, ErrNoPwd, ErrInvalidHostname).Error(),
		},
		{
			desc:        "Missing username and password",
			endpoint:    "http://localhost:50013",
			errExpected: true,
			errText:     multierr.Combine(ErrNoUsername, ErrNoPwd).Error(),
		},
		{
			desc:        "Missing username and invalid hostname, no protocol",
			endpoint:    "localhost:50013",
			password:    "password",
			errExpected: true,
			errText:     multierr.Combine(ErrNoUsername, ErrInvalidHostname).Error(),
		},
		{
			desc:        "Missing password and invalid hostname, no protocol",
			endpoint:    "localhost:50013",
			username:    "root",
			errExpected: true,
			errText:     multierr.Combine(ErrNoPwd, ErrInvalidHostname).Error(),
		},
		{
			desc:        "Missing username",
			endpoint:    "http://localhost:50013",
			password:    "password",
			errExpected: true,
			errText:     multierr.Combine(ErrNoUsername).Error(),
		},
		{
			desc:        "Missing password",
			endpoint:    "http://localhost:50013",
			username:    "root",
			errExpected: true,
			errText:     multierr.Combine(ErrNoPwd).Error(),
		},
		{
			desc:        "custom_host",
			username:    "root",
			password:    "password",
			endpoint:    "http://123.123.123.123:50013",
			errExpected: false,
		},
		{
			desc:        "custom_port",
			username:    "root",
			password:    "password",
			endpoint:    "http://123.123.123.123:9090",
			errExpected: false,
		},
		{
			desc:        "example config",
			username:    "root",
			password:    "password",
			endpoint:    "http://localhost:50013",
			errExpected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := NewFactory().CreateDefaultConfig().(*Config)
			cfg.Endpoint = tc.endpoint
			cfg.Username = tc.username
			cfg.Password = tc.password
			err := component.ValidateConfig(cfg)
			if tc.errExpected {
				require.EqualError(t, err, tc.errText)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(component.NewIDWithName(metadata.Type, "").String())
	require.NoError(t, err)
	require.NoError(t, component.UnmarshalConfig(sub, cfg))

	expected := factory.CreateDefaultConfig().(*Config)
	expected.Endpoint = "http://localhost:50013"
	expected.Password = "password"
	expected.Username = "root"
	expected.CollectionInterval = 60 * time.Second

	require.Equal(t, expected, cfg)
}
