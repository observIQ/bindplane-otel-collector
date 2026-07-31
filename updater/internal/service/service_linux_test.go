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

//go:build linux

package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSudoCommandNonInteractive(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires a non-root user so needsSudo() returns true")
	}

	// When running as non-root, sudo must be invoked non-interactively (-n) so a
	// missing or non-NOPASSWD sudoers rule fails fast instead of blocking on a
	// password prompt the updater has no TTY to answer.
	cmd := sudoCommand("systemctl", "start", "observiq-otel-collector")
	require.Equal(t, []string{"sudo", "-n", "systemctl", "start", "observiq-otel-collector"}, cmd.Args)
}
