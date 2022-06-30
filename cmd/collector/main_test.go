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

package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/observiq/observiq-otel-collector/opamp"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCheckManagerNoConfig(t *testing.T) {
	exec.Command("rm", "-r", "./manager.yaml").Run()
	manager := "./manager.yaml"
	err := checkManagerConfig(&manager)
	require.ErrorIs(t, err, os.ErrNotExist)

	tmp := "\000"
	err = checkManagerConfig(&tmp)
	require.Error(t, err)
}

func TestCheckManagerConfigNoFile(t *testing.T) {
	exec.Command("rm", "-r", "./manager.yaml").Run()
	manager := "./manager.yaml"
	err := checkManagerConfig(&manager)
	require.Error(t, err)

	os.Setenv(endpointENV, "0.0.0.0")
	defer os.Unsetenv(endpointENV)

	os.Setenv(agentNameENV, "agent name")
	defer os.Unsetenv(agentNameENV)

	os.Setenv(secretkeyENV, "secretKey")
	defer os.Unsetenv(secretkeyENV)

	os.Setenv(labelsENV, "this is a label")
	defer os.Unsetenv(labelsENV)
	defer os.Unsetenv(agentIdENV)

	manager = "./manager.yaml"
	err = checkManagerConfig(&manager)
	require.NoError(t, err)

	dat, err := os.ReadFile("./manager.yaml")
	out := &opamp.Config{}
	err = yaml.Unmarshal(dat, out)
	require.Equal(t,
		&opamp.Config{
			Endpoint: "0.0.0.0",
		},
		&opamp.Config{
			Endpoint: out.Endpoint,
		})
}

func TestCheckManagerConfig(t *testing.T) {
	manager := "./manager.yaml"
	err := checkManagerConfig(&manager)
	require.NoError(t, err)
}
