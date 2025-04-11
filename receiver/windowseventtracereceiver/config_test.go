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

//go:build windows

package windowseventtracereceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestConfig() *Config {
	return &Config{
		SessionName:       "TestSession",
		SessionBufferSize: 64,
		Providers: []Provider{
			{Name: "TestProvider", Level: LevelInformational},
		},
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "providers cannot be empty")
}
