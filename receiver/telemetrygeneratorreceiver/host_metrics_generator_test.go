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

package telemetrygeneratorreceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostMetricsDefaultConfig(t *testing.T) {

	// validate the default config
	cfg := GeneratorConfig{

		// type is intentionally "metrics" because that's what the host_metrics generator is
		// using under the hood. This is to ensure that the default configuration is valid,
		// since it's not validated at runtime.
		Type: "metrics",
		ResourceAttributes: map[string]any{
			"host.name": "2ed77de7e4c1",
			"os.type":   "linux",
		},
	}

	cfg, err := fillHostMetricsConfig(cfg)
	require.NoError(t, err)

	err = cfg.Validate()
	require.NoError(t, err)
}
