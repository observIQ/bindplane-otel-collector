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

package randomfailureprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.Equal(t, 0.5, cfg.FailureRate)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		err  error
	}{
		{name: "default", cfg: createDefaultConfig().(*Config), err: nil},
		{name: "negative_failure_rate", cfg: &Config{FailureRate: -0.1}, err: errInvalidFailureRate},
		{name: "greater_than_one_failure_rate", cfg: &Config{FailureRate: 1.1}, err: errInvalidFailureRate},
		{name: "zero_failure_rate", cfg: &Config{FailureRate: 0}, err: nil},
		{name: "one_failure_rate", cfg: &Config{FailureRate: 1}, err: nil},
		{name: "intermediate_failure_rate", cfg: &Config{FailureRate: 0.6758}, err: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.Validate()
			require.Equal(t, test.err, err)
		})
	}
}
