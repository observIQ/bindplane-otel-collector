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

package runtime

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetLegacyEnvVars(t *testing.T) {
	testCases := []struct {
		name        string
		env         map[string]string
		wantHome    string
		wantStorage string
	}{
		{
			name: "derives legacy vars from generic vars",
			env: map[string]string{
				"BINDPLANE_COLLECTOR_HOME":    "/opt/observiq-otel-collector",
				"BINDPLANE_COLLECTOR_STORAGE": "/opt/observiq-otel-collector/storage",
			},
			wantHome:    "/opt/observiq-otel-collector",
			wantStorage: "/opt/observiq-otel-collector/storage",
		},
		{
			name: "explicitly set legacy var wins",
			env: map[string]string{
				"BINDPLANE_COLLECTOR_HOME": "/opt/observiq-otel-collector",
				"OIQ_OTEL_COLLECTOR_HOME":  "/custom/home",
			},
			wantHome: "/custom/home",
		},
		{
			name: "nothing set leaves legacy vars unset",
			env:  map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{
				"OIQ_OTEL_COLLECTOR_HOME", "OIQ_OTEL_COLLECTOR_STORAGE",
				"BINDPLANE_COLLECTOR_HOME", "BINDPLANE_COLLECTOR_STORAGE",
			} {
				t.Setenv(key, "")
				os.Unsetenv(key)
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			setLegacyEnvVars()

			require.Equal(t, tc.wantHome, os.Getenv("OIQ_OTEL_COLLECTOR_HOME"))
			require.Equal(t, tc.wantStorage, os.Getenv("OIQ_OTEL_COLLECTOR_STORAGE"))
		})
	}
}
