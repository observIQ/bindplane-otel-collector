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
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scenario runs the test scenarios for one distro. Each scenario starts a
// fresh container.
type scenario struct {
	env    *suiteEnv
	distro distro
	ops    formatOps
	// pkg is the host path of the package under test.
	pkg string
	// image is the systemd-enabled image for the distro.
	image string
}

// install installs the package on a clean system, starts the service the
// way install_unix.sh does, and then removes the package.
func (s *scenario) install(t *testing.T) {
	b := startBox(t, s.env, s.image)
	pkg := b.copyPackage(s.pkg)

	b.installPackage(s.ops, pkg, false)
	s.assertInstalled(b, pkg, nil, nil)

	// The package configures the service but does not enable or start it.
	assert.Equal(t, "disabled", b.unitState("is-enabled"), "service should not be enabled by the package")
	assert.Equal(t, "inactive", b.unitState("is-active"), "service should not be started by the package")
	b.assertAbsent(wantsPath)

	b.mustRun("systemctl enable --now " + packageName)
	b.assertServiceRunning(1)

	s.uninstall(t, b)
}

// upgrade installs a released package, starts it, modifies the files a user
// would, upgrades to the package under test, and then removes the package.
func (s *scenario) upgrade(t *testing.T, from string) {
	oldHostPkg := releasePackage(t, from, s.env.arch, s.distro.Format)
	b := startBox(t, s.env, s.image)
	oldPkg := b.copyPackage(oldHostPkg)
	newPkg := b.copyPackage(s.pkg)

	b.writeOverride(s.ops)
	b.installPackage(s.ops, oldPkg, false)
	b.mustRun("systemctl enable --now " + packageName)
	require.True(t, poll(time.Minute, func() bool { return b.startupCount() >= 1 }), "release %s should start", from)
	oldPID := b.unitProps("MainPID")["MainPID"]

	// User state that the upgrade must preserve.
	b.writeUserFiles(true)
	preserved := []string{
		installDir + "/config.yaml",
		installDir + "/logging.yaml",
		installDir + "/manager.yaml",
		installDir + "/storage/marker",
		s.ops.overridePath,
	}
	before := b.sha256sums(preserved...)

	// Upgrading to a snapshot of the release it is based on is a downgrade
	// as far as the package manager is concerned (1.2.3~SNAPSHOT < 1.2.3).
	oldVersion := b.mustRun(s.ops.fileVersion(oldPkg))
	newVersion := b.mustRun(s.ops.fileVersion(newPkg))
	downgrade := b.run(s.ops.older(newVersion, oldVersion)).code == 0
	if downgrade {
		t.Logf("%s is older than %s, installing it as a downgrade", newVersion, oldVersion)
	}
	b.installPackage(s.ops, newPkg, downgrade)

	assert.Equal(t, before, b.sha256sums(preserved...), "the upgrade should not modify user files")
	userFiles := map[string]fileEntry{
		"manager.yaml":   {"f", runtimeUser, runtimeGrp, "640"},
		"storage/marker": {"f", runtimeUser, runtimeGrp, "640"},
	}
	// Plugins removed from the new release are not cleaned up on upgrade.
	oldPlugins := func(p string) bool { return strings.HasPrefix(p, "plugins/") }
	s.assertInstalled(b, newPkg, userFiles, oldPlugins)

	// The package does not restart the service; the old binary keeps
	// running until it is restarted.
	assert.Equal(t, "enabled", b.unitState("is-enabled"), "service should stay enabled")
	assert.Equal(t, "active", b.unitState("is-active"), "service should keep running")
	assert.Equal(t, oldPID, b.unitProps("MainPID")["MainPID"], "the package should not restart the service")

	// Restart like install_unix.sh does after an upgrade. manager.yaml is
	// removed first because it switches the collector to managed mode.
	b.mustRun("rm " + installDir + "/manager.yaml")
	starts := b.startupCount()
	b.mustRun("systemctl restart " + packageName)
	pid := b.assertServiceRunning(starts + 1)
	assert.NotEqual(t, oldPID, pid, "service should have a new process after restart")

	s.uninstall(t, b)
}

// uninstall removes the installed and running package from b, and checks
// what is removed and what is kept. It runs as the "uninstall" subtest of
// each action.
func (s *scenario) uninstall(t *testing.T, b *box) {
	t.Run("uninstall", func(t *testing.T) {
		b := b.withT(t)

		// User state that removal keeps, or removes where the per-format
		// expectations say so. The package scripts source the override
		// file on removal too.
		b.writeOverride(s.ops)
		b.writeUserFiles(false)

		b.removePackage(s.ops)

		assert.NotEqual(t, "active", b.unitState("is-active"), "service should be stopped")
		assert.NotEqual(t, "enabled", b.unitState("is-enabled"), "service should be disabled")
		// The bracket keeps pgrep from matching the shell running it.
		assert.NotZero(t, b.run("pgrep -f '[/]opt/observiq-otel-collector/observiq-otel-collector'").code, "collector process should be stopped")

		b.assertPresent(uninstallShared.present...)
		b.assertAbsent(uninstallShared.absent...)
		b.assertPresent(s.ops.uninstall.present...)
		b.assertAbsent(s.ops.uninstall.absent...)
		assert.Equal(t, s.ops.uninstall.state, b.mustRun(s.ops.packageState), "package state after removal")
		assert.Zero(t, b.run("id "+runtimeUser).code, "user %s should be kept", runtimeUser)
	})
}

// assertInstalled checks the state of a system with pkg installed.
// extraTree adds expected entries under installDir, and allowExtra permits
// unexpected ones, see assertTree.
func (s *scenario) assertInstalled(b *box, pkg string, extraTree map[string]fileEntry, allowExtra func(string) bool) {
	b.t.Helper()
	want := installTree(b.payloadPlugins(s.ops, pkg))
	maps.Copy(want, extraTree)
	b.assertTree(installDir, want, allowExtra)
	b.assertFiles(installSystemFiles)
	b.assertAbsent(installAbsent...)
	b.assertPackageVersion(s.ops, pkg)
	b.assertVersionFiles(s.env.dist.metadata)
	b.assertRuntimeUser()
	b.assertUnit()
}

func (b *box) installPackage(o formatOps, pkg string, downgrade bool) {
	b.t.Helper()
	b.pkgCommand(o.install(pkg, downgrade))
}

func (b *box) removePackage(o formatOps) {
	b.t.Helper()
	b.pkgCommand(o.remove)
}

// pkgCommand runs a package manager command and logs its output, which
// includes the output of the package scripts.
func (b *box) pkgCommand(cmd string) {
	b.t.Helper()
	res := b.run(cmd)
	b.t.Logf("$ %s\n%s", cmd, res)
	require.Zero(b.t, res.code, "%s failed", cmd)
}

// writeOverride writes a package override file that keeps the defaults. The
// install scripts only source it when it is written before the install.
func (b *box) writeOverride(o formatOps) {
	b.t.Helper()
	b.mustRun("mkdir -p \"$(dirname " + o.overridePath + ")\" && echo '# written by the packaging test suite' > " + o.overridePath)
}

// writeUserFiles creates files a user or install_unix.sh would create after
// install. When editConfig is set, the config files are modified too.
func (b *box) writeUserFiles(editConfig bool) {
	b.t.Helper()
	script := `set -e
cd ` + installDir + `
install -o ` + runtimeUser + ` -g ` + runtimeGrp + ` -m 0640 /dev/null storage/marker
echo 'written by the packaging test suite' > storage/marker
install -o ` + runtimeUser + ` -g ` + runtimeGrp + ` -m 0640 /dev/null manager.yaml
echo 'endpoint: "ws://127.0.0.1:3001/v1/opamp"' > manager.yaml
`
	if editConfig {
		script += `echo '# edited by the packaging test suite' >> config.yaml
echo '# edited by the packaging test suite' >> logging.yaml
`
	}
	b.mustRun(script)
}
