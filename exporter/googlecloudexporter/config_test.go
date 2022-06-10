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

package googlecloudexporter

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
	"google.golang.org/api/option"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	googleCfg, ok := cfg.(*Config)
	require.True(t, ok)

	require.Equal(t, config.NewComponentID(typeStr), googleCfg.ID())
	require.Equal(t, defaultMetricPrefix, googleCfg.GCPConfig.MetricConfig.Prefix)
	require.Equal(t, defaultUserAgent, googleCfg.GCPConfig.UserAgent)
	require.Nil(t, googleCfg.Validate())
}

func TestSetClientOptionsWithCredentials(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
		opts   []option.ClientOption
	}{
		{
			name: "With no credentials",
			config: &Config{
				GCPConfig: &googlecloudexporter.Config{},
			},
			opts: []option.ClientOption{},
		},
		{
			name: "With credentials json",
			config: &Config{
				Credentials: "testjson",
				GCPConfig:   &googlecloudexporter.Config{},
			},
			opts: []option.ClientOption{
				option.WithCredentialsJSON([]byte("testjson")),
			},
		},
		{
			name: "With credentials file",
			config: &Config{
				CredentialsFile: "testfile",
				GCPConfig:       &googlecloudexporter.Config{},
			},
			opts: []option.ClientOption{
				option.WithCredentialsFile("testfile"),
			},
		},
		{
			name: "With both credentials json and credentials file",
			config: &Config{
				Credentials:     "testjson",
				CredentialsFile: "testfile",
				GCPConfig:       &googlecloudexporter.Config{},
			},
			opts: []option.ClientOption{
				option.WithCredentialsJSON([]byte("testjson")),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Nil(t, tc.config.GCPConfig.MetricConfig.ClientConfig.GetClientOptions)
			require.Nil(t, tc.config.GCPConfig.LogConfig.ClientConfig.GetClientOptions)
			require.Nil(t, tc.config.GCPConfig.TraceConfig.ClientConfig.GetClientOptions)

			tc.config.setClientOptions()
			require.NotNil(t, tc.config.GCPConfig.MetricConfig.ClientConfig.GetClientOptions)
			require.NotNil(t, tc.config.GCPConfig.LogConfig.ClientConfig.GetClientOptions)
			require.NotNil(t, tc.config.GCPConfig.TraceConfig.ClientConfig.GetClientOptions)

			opts := tc.config.GCPConfig.MetricConfig.ClientConfig.GetClientOptions()
			require.Equal(t, tc.opts, opts)
		})
	}
}
