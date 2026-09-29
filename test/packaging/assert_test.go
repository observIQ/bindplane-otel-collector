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

//go:build packaging

package packaging

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// snapshotTree returns every entry under dir, keyed by path relative to dir.
func (b *box) snapshotTree(dir string) map[string]fileEntry {
	b.t.Helper()
	out := b.mustRun("find " + shellQuote(dir) + ` -printf '%P|%y|%u|%g|%m\n'`)
	tree := map[string]fileEntry{}
	for line := range strings.SplitSeq(out, "\n") {
		parts := strings.Split(line, "|")
		require.Len(b.t, parts, 5, "unexpected find output %q", line)
		tree[parts[0]] = fileEntry{Type: parts[1], Owner: parts[2], Group: parts[3], Mode: parts[4]}
	}
	return tree
}

// assertTree compares a snapshot of dir against the expected entries. Paths
// for which allowExtra returns true may be present without being expected.
func (b *box) assertTree(dir string, want map[string]fileEntry, allowExtra func(path string) bool) {
	b.t.Helper()
	got := b.snapshotTree(dir)

	var problems, extras []string
	for p, w := range want {
		g, ok := got[p]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("missing:    %s/%s (want %s)", dir, p, w))
		case g != w:
			problems = append(problems, fmt.Sprintf("mismatch:   %s/%s got %s, want %s", dir, p, g, w))
		}
	}
	for p, g := range got {
		if _, ok := want[p]; ok {
			continue
		}
		if allowExtra != nil && allowExtra(p) {
			extras = append(extras, fmt.Sprintf("%s/%s (%s)", dir, p, g))
			continue
		}
		problems = append(problems, fmt.Sprintf("unexpected: %s/%s (%s)", dir, p, g))
	}
	sort.Strings(problems)
	sort.Strings(extras)
	if len(extras) > 0 {
		b.t.Logf("allowed extra entries under %s:\n  %s", dir, strings.Join(extras, "\n  "))
	}
	assert.Empty(b.t, problems, "unexpected state under %s:\n  %s", dir, strings.Join(problems, "\n  "))
}

func (e fileEntry) String() string {
	return fmt.Sprintf("%s %s:%s %s", e.Type, e.Owner, e.Group, e.Mode)
}

// stat returns the entry for path, and false when it does not exist.
func (b *box) stat(path string) (fileEntry, bool) {
	b.t.Helper()
	res := b.run("stat -c '%F|%U|%G|%a' " + shellQuote(path))
	if res.code != 0 {
		return fileEntry{}, false
	}
	parts := strings.Split(strings.TrimSpace(res.stdout), "|")
	require.Len(b.t, parts, 4, "unexpected stat output %q", res.stdout)
	typ := parts[0]
	switch parts[0] {
	case "regular file", "regular empty file":
		typ = "f"
	case "directory":
		typ = "d"
	case "symbolic link":
		typ = "l"
	}
	return fileEntry{Type: typ, Owner: parts[1], Group: parts[2], Mode: parts[3]}, true
}

// exists reports whether path exists, including dangling symlinks.
func (b *box) exists(path string) bool {
	b.t.Helper()
	p := shellQuote(path)
	return b.run("test -e "+p+" || test -L "+p).code == 0
}

func (b *box) assertFiles(want map[string]fileEntry) {
	b.t.Helper()
	for path, w := range want {
		got, ok := b.stat(path)
		if assert.True(b.t, ok, "%s should exist", path) {
			assert.Equal(b.t, w, got, "%s", path)
		}
	}
}

func (b *box) assertPresent(paths ...string) {
	b.t.Helper()
	for _, p := range paths {
		assert.True(b.t, b.exists(p), "%s should exist", p)
	}
}

func (b *box) assertAbsent(paths ...string) {
	b.t.Helper()
	for _, p := range paths {
		assert.False(b.t, b.exists(p), "%s should not exist", p)
	}
}

// sha256sums returns the sha256sum output for paths, which must exist.
func (b *box) sha256sums(paths ...string) string {
	b.t.Helper()
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = shellQuote(p)
	}
	return b.mustRun("sha256sum " + strings.Join(quoted, " "))
}

// payloadPlugins lists the plugin files in the package payload, relative to
// the plugins directory.
func (b *box) payloadPlugins(o formatOps, pkg string) []string {
	b.t.Helper()
	var plugins []string
	for line := range strings.SplitSeq(b.mustRun(o.payload(pkg)), "\n") {
		rel, ok := strings.CutPrefix(strings.TrimSpace(line), o.payloadPrefix)
		if !ok {
			continue
		}
		name, ok := strings.CutPrefix(strings.TrimSuffix(rel, "/"), "plugins/")
		if ok && name != "" && !slices.Contains(plugins, name) {
			plugins = append(plugins, name)
		}
	}
	require.NotEmpty(b.t, plugins, "no plugins found in the payload of %s", pkg)
	return plugins
}

// assertPackageVersion checks that exactly one instance of the package is
// installed, at the version recorded in the package file.
func (b *box) assertPackageVersion(o formatOps, pkg string) {
	b.t.Helper()
	want := b.mustRun(o.fileVersion(pkg))
	got := b.mustRun(o.installedVersions)
	assert.Equal(b.t, want, got, "installed package version(s)")
}

// assertVersionFiles checks VERSION.txt and the collector's --version output
// against the dist metadata.
func (b *box) assertVersionFiles(md distMetadata) {
	b.t.Helper()
	assert.Equal(b.t, "v"+md.Version, b.mustRun("cat "+installDir+"/VERSION.txt"), "VERSION.txt")

	out := b.mustRun(installDir + "/observiq-otel-collector --version")
	assert.Contains(b.t, out, "observiq-otel-collector version "+md.Tag, "--version output")
	assert.Contains(b.t, out, "commit: "+md.Commit, "--version output")
}

// assertRuntimeUser checks the user and group created by preinstall.
func (b *box) assertRuntimeUser() {
	b.t.Helper()
	fields := strings.Split(b.mustRun("getent passwd "+runtimeUser), ":")
	require.Len(b.t, fields, 7, "getent passwd output")
	assert.Equal(b.t, "/sbin/nologin", fields[6], "login shell of %s", runtimeUser)

	// useradd --system allocates a UID below the distro's UID_MIN.
	uid, err := strconv.Atoi(fields[2])
	require.NoError(b.t, err)
	uidMin, err := strconv.Atoi(b.mustRun(`awk '$1 == "UID_MIN" {print $2}' /etc/login.defs`))
	require.NoError(b.t, err)
	assert.Less(b.t, uid, uidMin, "%s should be a system user", runtimeUser)

	gid := b.mustRun("getent group " + runtimeGrp + " | cut -d: -f3")
	assert.Equal(b.t, gid, fields[3], "primary group of %s should be %s", runtimeUser, runtimeGrp)
}

func (b *box) unitState(verb string) string {
	b.t.Helper()
	return strings.TrimSpace(b.run("systemctl " + verb + " " + packageName).stdout)
}

// unitProps returns `systemctl show` properties of the unit.
func (b *box) unitProps(props ...string) map[string]string {
	b.t.Helper()
	args := ""
	for _, p := range props {
		args += " -p " + p
	}
	out := b.mustRun("systemctl show" + args + " " + packageName)
	m := map[string]string{}
	for line := range strings.SplitSeq(out, "\n") {
		k, v, _ := strings.Cut(line, "=")
		m[k] = v
	}
	return m
}

// assertUnit checks the systemd unit written by postinstall.
func (b *box) assertUnit() {
	b.t.Helper()
	props := []string{"ExecStart", "Environment"}
	for p := range installUnitProps {
		props = append(props, p)
	}
	got := b.unitProps(props...)
	for p, want := range installUnitProps {
		assert.Equal(b.t, want, got[p], "unit property %s", p)
	}
	assert.Contains(b.t, got["ExecStart"], "argv[]="+installDir+"/observiq-otel-collector --config config.yaml ;", "unit ExecStart")
	env := strings.Fields(got["Environment"])
	assert.Contains(b.t, env, "BINDPLANE_COLLECTOR_HOME="+installDir, "unit Environment")
	assert.Contains(b.t, env, "BINDPLANE_COLLECTOR_STORAGE="+installDir+"/storage", "unit Environment")
}

// poll calls cond every second until it returns true or timeout elapses. It
// runs cond on the calling goroutine (unlike require.Eventually) so that cond
// may use the box helpers, which fail the test on errors.
func poll(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Second)
	}
}

// startupCount returns how many times the collector logged its standalone
// startup line.
func (b *box) startupCount() int {
	b.t.Helper()
	out := b.run("grep -c " + shellQuote(standaloneLogLine) + " " + installDir + "/log/collector.log").stdout
	n, _ := strconv.Atoi(strings.TrimSpace(out))
	return n
}

// assertServiceRunning checks that the service is enabled and running the
// installed binary with the expected identity, that it logged startup n
// times in total, and that it stays up. It returns the service's main PID.
func (b *box) assertServiceRunning(n int) string {
	b.t.Helper()
	require.Equal(b.t, "enabled", b.unitState("is-enabled"), "service should be enabled")
	require.Equal(b.t, "active", b.unitState("is-active"), "service should be active")
	b.assertPresent(wantsPath)

	pid := b.unitProps("MainPID")["MainPID"]
	require.NotContains(b.t, []string{"", "0"}, pid, "service main PID")
	assert.Equal(b.t, []string{"root", runtimeGrp}, strings.Fields(b.mustRun("ps -o euser=,egroup= -p "+pid)), "service process user and group")
	assert.Equal(b.t, installDir+"/observiq-otel-collector", b.mustRun("readlink /proc/"+pid+"/exe"), "service process executable")

	require.True(b.t, poll(time.Minute, func() bool { return b.startupCount() >= n }),
		"collector.log should contain %q %d time(s)", standaloneLogLine, n)
	got, ok := b.stat(installDir + "/log/collector.log")
	require.True(b.t, ok)
	assert.Equal(b.t, runtimeUser+":"+runtimeGrp, got.Owner+":"+got.Group, "collector.log owner")

	// Outlast RestartSec=5s to catch a crash loop.
	time.Sleep(6 * time.Second)
	props := b.unitProps("ActiveState", "NRestarts", "MainPID")
	assert.Equal(b.t, map[string]string{"ActiveState": "active", "NRestarts": "0", "MainPID": pid}, props, "service should stay up")
	return pid
}
