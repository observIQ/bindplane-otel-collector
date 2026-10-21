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

// srcActive reports whether lssrc output shows the subsystem as active.
func srcActive(lssrcOut []byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(lssrcOut))
	sc.Scan() // header
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) > 0 && f[len(f)-1] == "active" {
			return true
		}
	}
	return false
}
