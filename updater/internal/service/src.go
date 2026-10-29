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
	"bufio"
	"bytes"
	"fmt"
	"path"
	"strings"
	"time"
)

const (
	// srcWaitTimeout bounds the wait for startsrc and stopsrc to finish. AIX
	// can be slow to start or stop a subsystem.
	srcWaitTimeout  = 120 * time.Second
	srcPollInterval = time.Second
)

// srcService holds what the AIX System Resource Controller needs to run the collector.
// SRC has no working directory, so every path the collector reads is absolute.
type srcService struct {
	subsystem  string
	group      string
	installDir string
	envFile    string
}

// mkssysArgs defines the subsystem: run as root like the Linux unit, stop with
// SIGTERM then SIGKILL, and restart on abnormal exit.
func (s srcService) mkssysArgs() []string {
	dir := path.Clean(s.installDir)
	return []string{
		"-s", s.subsystem,
		"-G", s.group,
		"-p", path.Join(dir, "observiq-otel-collector"),
		"-u", "0",
		"-S", "-n", "15", "-f", "9", "-R",
		"-a", fmt.Sprintf("--config %s --manager %s --logging %s",
			path.Join(dir, "config.yaml"), path.Join(dir, "manager.yaml"), path.Join(dir, "logging.yaml")),
	}
}

// startsrcArgs starts the group with the env file's variables, which stand in
// for the Linux unit's Environment lines. Like v2's bdot.env, the file is one
// line of space-separated KEY=value pairs, since startsrc -e splits only on spaces.
func (s srcService) startsrcArgs(envFileContents []byte) []string {
	args := []string{"-g", s.group}
	if env := strings.TrimSpace(string(envFileContents)); env != "" {
		args = append(args, "-e", env)
	}
	return args
}

// inittabEntry starts the group once at boot with the same env file.
func (s srcService) inittabEntry() string {
	return fmt.Sprintf(`%s:23456789:once:startsrc -g %s -e "$(cat %s)"`, s.group, s.group, s.envFile)
}

// srcStatus returns the subsystem's status from lssrc output, or "" when lssrc lists none.
func srcStatus(lssrcOut []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(lssrcOut))
	sc.Scan() // header
	for sc.Scan() {
		if f := strings.Fields(sc.Text()); len(f) > 0 {
			return f[len(f)-1]
		}
	}
	return ""
}

// srcActive reports whether lssrc output shows the subsystem running. A
// subsystem that is still stopping counts as running.
func srcActive(lssrcOut []byte) bool {
	status := srcStatus(lssrcOut)
	return status == "active" || status == "stopping"
}

// srcPoller waits for the subsystem to reach a status. startsrc and stopsrc
// return before SRC finishes, so their exit codes don't show the outcome.
type srcPoller struct {
	lssrc func() ([]byte, error)
	now   func() time.Time
	sleep func(time.Duration)
}

// waitFor polls lssrc until the subsystem reports want, or returns an error
// once timeout passes.
func (p srcPoller) waitFor(want string, timeout time.Duration) error {
	deadline := p.now().Add(timeout)
	var last string
	var lastErr error
	for {
		out, err := p.lssrc()
		if err == nil {
			last = srcStatus(out)
			if last == want {
				return nil
			}
		}
		lastErr = err
		if !p.now().Before(deadline) {
			if lastErr != nil {
				return fmt.Errorf("subsystem did not become %s within %s (last status %q): %w", want, timeout, last, lastErr)
			}
			return fmt.Errorf("subsystem did not become %s within %s (last status %q)", want, timeout, last)
		}
		p.sleep(srcPollInterval)
	}
}
