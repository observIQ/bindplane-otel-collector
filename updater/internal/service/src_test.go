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

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSRCArgs(t *testing.T) {
	s := srcService{
		subsystem:  "observiq-otel-collector",
		group:      "oiqcollector",
		installDir: "/opt/observiq-otel-collector",
		envFile:    "/etc/observiq-otel-collector.env",
	}

	t.Run("mkssys runs the collector as root with absolute config paths", func(t *testing.T) {
		require.Equal(t, []string{
			"-s", "observiq-otel-collector",
			"-G", "oiqcollector",
			"-p", "/opt/observiq-otel-collector/observiq-otel-collector",
			"-u", "0",
			"-S", "-n", "15", "-f", "9", "-R",
			"-a", "--config /opt/observiq-otel-collector/config.yaml --manager /opt/observiq-otel-collector/manager.yaml --logging /opt/observiq-otel-collector/logging.yaml",
		}, s.mkssysArgs())
	})

	t.Run("startsrc starts the group with the env file's variables", func(t *testing.T) {
		require.Equal(t, []string{"-g", "oiqcollector", "-e", "A=1 B=/x"}, s.startsrcArgs([]byte("A=1 B=/x\n")))
	})

	t.Run("startsrc without an env file passes no environment", func(t *testing.T) {
		require.Equal(t, []string{"-g", "oiqcollector"}, s.startsrcArgs(nil))
	})

	t.Run("inittab reads the env file at boot", func(t *testing.T) {
		require.Equal(t,
			`oiqcollector:23456789:once:startsrc -g oiqcollector -e "$(cat /etc/observiq-otel-collector.env)"`,
			s.inittabEntry())
	})
}

func TestSRCActive(t *testing.T) {
	header := "Subsystem         Group            PID          Status \n"
	require.True(t, srcActive([]byte(header+" observiq-otel-collector oiqcollector     11272516     active\n")))
	require.False(t, srcActive([]byte(header+" observiq-otel-collector oiqcollector                  inoperative\n")))
	require.False(t, srcActive([]byte(header+" observiq-otel-collector oiqcollector     11272516     stopping\n")))
	require.False(t, srcActive(nil))
}
