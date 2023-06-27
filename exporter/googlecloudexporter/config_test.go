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
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/collector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func TestCreateDefaultConfig(t *testing.T) {
	collectorVersion := "v1.2.3"

	cfg := createDefaultConfig(collectorVersion)()
	googleCfg, ok := cfg.(*Config)
	require.True(t, ok)

	expectedUserAgent := fmt.Sprintf("%s/%s", defaultUserAgent, collectorVersion)

	require.Equal(t, defaultMetricPrefix, googleCfg.GCPConfig.MetricConfig.Prefix)
	require.Equal(t, expectedUserAgent, googleCfg.GCPConfig.UserAgent)
	require.Len(t, googleCfg.GCPConfig.MetricConfig.ResourceFilters, 1)
	require.Equal(t, googleCfg.GCPConfig.MetricConfig.ResourceFilters[0].Prefix, "")
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

func TestSetProject(t *testing.T) {
	testCases := []struct {
		name            string
		config          *Config
		expectedProject string
		expectedErr     string
	}{
		{
			name: "With project already set",
			config: &Config{
				GCPConfig: &googlecloudexporter.Config{
					Config: collector.Config{
						ProjectID: "test",
					},
				},
			},
			expectedProject: "test",
		},
		{
			name: "With project in json credentials",
			config: &Config{
				Credentials: `{"project_id":"test"}`,
				GCPConfig:   &googlecloudexporter.Config{},
			},
			expectedProject: "test",
		},
		{
			name: "With missing json key",
			config: &Config{
				Credentials: `{"test":"value"}`,
				GCPConfig:   &googlecloudexporter.Config{},
			},
			expectedErr: "project id does not exist",
		},
		{
			name: "With invalid json",
			config: &Config{
				Credentials: `{`,
				GCPConfig:   &googlecloudexporter.Config{},
			},
			expectedErr: "failed to unmarshal credentials",
		},
		{
			name: "With invalid string",
			config: &Config{
				Credentials: `{"project_id":100}`,
				GCPConfig:   &googlecloudexporter.Config{},
			},
			expectedErr: "project id is not a string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.setProject()
			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}

			require.Equal(t, tc.expectedProject, tc.config.GCPConfig.ProjectID)
		})
	}
}
