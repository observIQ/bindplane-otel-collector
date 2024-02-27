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

package snowflakeexporter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		cfg         *Config
		expectedErr error
	}{
		{
			desc: "default pass",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
		},
		{
			desc: "Missing account identifier",
			cfg: &Config{
				AccountIdentifier: "",
				Username:          "user",
				Password:          "pass",
				Warehouse:         "wh",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("account_identifier is required"),
		},
		{
			desc: "Missing username",
			cfg: &Config{
				AccountIdentifier: "id",
				Username:          "",
				Password:          "pass",
				Warehouse:         "wh",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("username is required"),
		},
		{
			desc: "Missing password",
			cfg: &Config{
				AccountIdentifier: "id",
				Username:          "user",
				Password:          "",
				Warehouse:         "wh",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("password is required"),
		},
		{
			desc: "Missing warehouse",
			cfg: &Config{
				AccountIdentifier: "id",
				Username:          "user",
				Password:          "pass",
				Warehouse:         "",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("warehouse is required"),
		},
		{
			desc: "empty database",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          "",
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("database cannot be set as empty"),
		},
		{
			desc: "empty logs schema",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: "",
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("logs schema cannot be set as empty"),
		},
		{
			desc: "empty logs table",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  "",
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("logs table cannot be set as empty"),
		},
		{
			desc: "empty metrics schema",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: "",
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("metrics schema cannot be set as empty"),
		},
		{
			desc: "empty metrics table",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  "",
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("metrics table cannot be set as empty"),
		},
		{
			desc: "empty traces schema",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: "",
					Table:  defaultTable,
				},
			},
			expectedErr: errors.New("traces schema cannot be set as empty"),
		},
		{
			desc: "empty traces table",
			cfg: &Config{
				AccountIdentifier: "accountID",
				Username:          "username",
				Password:          "password",
				Warehouse:         "warehouse",
				Database:          defaultDatabase,
				Logs: TelemetryConfig{
					Schema: defaultLogsSchema,
					Table:  defaultTable,
				},
				Metrics: TelemetryConfig{
					Schema: defaultMetricsSchema,
					Table:  defaultTable,
				},
				Traces: TelemetryConfig{
					Schema: defaultTracesSchema,
					Table:  "",
				},
			},
			expectedErr: errors.New("traces table cannot be set as empty"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}
