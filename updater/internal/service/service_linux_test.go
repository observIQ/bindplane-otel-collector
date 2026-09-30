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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
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

func TestRunWithOutput(t *testing.T) {
	t.Run("Success returns nil", func(t *testing.T) {
		require.NoError(t, runWithOutput(exec.Command("sh", "-c", "echo started")))
	})

	t.Run("Failure includes the output", func(t *testing.T) {
		// sudo writes the reason it refused a command to stderr.
		err := runWithOutput(exec.Command("sh", "-c", "echo 'sudo: a password is required' >&2; exit 1"))
		require.EqualError(t, err, "exit status 1: sudo: a password is required")

		var exitErr *exec.ExitError
		require.True(t, errors.As(err, &exitErr))
	})

	t.Run("Failure without output", func(t *testing.T) {
		err := runWithOutput(exec.Command("sh", "-c", "exit 1"))
		require.EqualError(t, err, "exit status 1")
	})
}

func TestLinuxSystemdServiceUnprivileged(t *testing.T) {
	// The unprivileged field is set directly, so this runs the same for any uid.
	installDir := t.TempDir()
	installedServiceFilePath := filepath.Join(t.TempDir(), "observiq-otel-collector.service")
	installedContent := []byte("[Unit]\nDescription=installed unit\n")
	require.NoError(t, os.WriteFile(installedServiceFilePath, installedContent, 0600))

	l := &linuxSystemdService{
		newServiceFilePath:       filepath.Join(t.TempDir(), "missing.service"),
		serviceName:              "bdot-nonexistent-test.service",
		installedServiceFilePath: installedServiceFilePath,
		installDir:               installDir,
		logger:                   zaptest.NewLogger(t),
		unprivileged:             true,
	}

	t.Run("Update does nothing", func(t *testing.T) {
		require.NoError(t, l.Update())

		content, err := os.ReadFile(installedServiceFilePath)
		require.NoError(t, err)
		require.Equal(t, installedContent, content)
	})

	t.Run("Backup does nothing", func(t *testing.T) {
		require.NoError(t, l.Backup())
		require.NoFileExists(t, path.BackupServiceFile(installDir))
	})
}

// sudoersHeredocStart is the line in install_sudoers (postinstall.sh) that
// starts the heredoc holding the sudoers drop-in.
const sudoersHeredocStart = `cat << EOF > "$tmp_file"`

// sudoersHeredoc returns the lines of the sudoers drop-in heredoc in script.
func sudoersHeredoc(script string) ([]string, error) {
	lines := strings.Split(script, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != sudoersHeredocStart {
			continue
		}
		if start != -1 {
			return nil, fmt.Errorf("%q appears more than once", sudoersHeredocStart)
		}
		start = i
	}
	if start == -1 {
		return nil, fmt.Errorf("%q not found", sudoersHeredocStart)
	}
	for i := start + 1; i < len(lines); i++ {
		// An unquoted heredoc ends only at a line that is exactly EOF.
		if lines[i] == "EOF" {
			return lines[start+1 : i], nil
		}
	}
	return nil, errors.New("sudoers heredoc has no EOF line")
}

// TestSudoersMatchesUpdaterCommands pins the whole sudoers drop-in written by
// postinstall, comment lines included, because sudoers reads some # lines
// (#include, #includedir, #<uid>) as syntax. A failure means the grant changed.
// The runtime user must not be able to escape to root, so the grant stays at
// systemctl start and stop of the collector service. The only Defaults line
// allowed is the one that turns off requiretty, which grants no commands.
// Any update to expected needs a security review.
func TestSudoersMatchesUpdaterCommands(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "package", "postinstall.sh"))
	require.NoError(t, err)
	script := string(content)

	svc := filepath.Base(path.SystemdFilePath)
	expected := []string{
		`# Sudoers drop-in for the Bindplane Distribution for OpenTelemetry Collector.`,
		`# Generated by the package for runtime user "${BDOT_USER}". It lets the updater`,
		`# stop and start the collector service while it replaces the collector's files.`,
		`# Any process running as "${BDOT_USER}" can run these commands. Don't add others.`,
		`# The updater has no TTY, so requiretty is off for this user. It grants nothing.`,
		`Defaults:${BDOT_USER} !requiretty`,
		`${BDOT_USER} ALL=(root) NOPASSWD: ${systemctl_path} start ` + svc,
		`${BDOT_USER} ALL=(root) NOPASSWD: ${systemctl_path} stop ` + svc,
	}

	got, err := sudoersHeredoc(script)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	// Catches a grant written outside the heredoc, such as an appended echo.
	require.Equal(t, 2, strings.Count(script, "NOPASSWD"))
	// Catches a Defaults line written outside the heredoc.
	require.Equal(t, 1, strings.Count(script, "Defaults:"))

	t.Run("detects widened grants", func(t *testing.T) {
		mutations := []struct {
			name     string
			old, new string
		}{
			{
				name: "SETENV on start",
				old:  "NOPASSWD: ${systemctl_path} start",
				new:  "NOPASSWD:SETENV: ${systemctl_path} start",
			},
			{
				name: "SETENV on stop",
				old:  "NOPASSWD: ${systemctl_path} stop",
				new:  "NOPASSWD:SETENV: ${systemctl_path} stop",
			},
			{
				name: "broader run-as list",
				old:  "ALL=(root)",
				new:  "ALL=(ALL)",
			},
			{
				name: "Defaults line",
				old:  "stop " + svc + "\nEOF",
				new:  "stop " + svc + "\nDefaults:${BDOT_USER} !env_reset\nEOF",
			},
			{
				name: "changed Defaults line",
				old:  "!requiretty",
				new:  "!env_reset",
			},
			{
				name: "include line",
				old:  sudoersHeredocStart + "\n",
				new:  sudoersHeredocStart + "\n#includedir /etc/bdot-sudoers.d\n",
			},
		}
		for _, m := range mutations {
			t.Run(m.name, func(t *testing.T) {
				mutated := strings.Replace(script, m.old, m.new, 1)
				require.NotEqual(t, script, mutated, "mutation did not apply")

				got, err := sudoersHeredoc(mutated)
				require.NoError(t, err)
				require.NotEqual(t, expected, got)
			})
		}
	})
}
