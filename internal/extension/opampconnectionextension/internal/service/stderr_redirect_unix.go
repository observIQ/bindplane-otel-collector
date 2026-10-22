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

//go:build aix || freebsd

package service

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// redirectServiceStderr sends stderr to $BINDPLANE_COLLECTOR_HOME/log/observiq_collector.err
// when the service manager discards it. AIX SRC leaves stderr on /dev/console
// and FreeBSD's daemon(8) on /dev/null. SRC's own mkssys -e is no substitute:
// SRC truncates that file on every start, including the restart after a crash.
func redirectServiceStderr() error {
	home, ok := os.LookupEnv("BINDPLANE_COLLECTOR_HOME")
	if !ok || !stderrDiscarded() {
		return nil
	}
	if home == "" {
		return errors.New("BINDPLANE_COLLECTOR_HOME is empty")
	}
	return redirectStderrToFile(filepath.Join(home, "log", "observiq_collector.err"), unix.Dup2)
}

// stderrDiscarded reports whether stderr is /dev/console or /dev/null, where
// nothing keeps it.
func stderrDiscarded() bool {
	stderrInfo, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	for _, dev := range []string{"/dev/console", "/dev/null"} {
		if info, err := os.Stat(dev); err == nil && os.SameFile(info, stderrInfo) {
			return true
		}
	}
	return false
}
