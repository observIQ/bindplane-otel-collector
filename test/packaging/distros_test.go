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
	"strings"
)

// pkgFormat is a Linux package format produced by goreleaser (nfpm).
type pkgFormat string

const (
	formatDeb pkgFormat = "deb"
	formatRPM pkgFormat = "rpm"
)

// distro is one entry in the test matrix.
type distro struct {
	// Name is used for the subtest name and the image tag (bdot-pkgtest:<Name>).
	Name string
	// BaseImage is passed to the Dockerfile as the BASE_IMAGE build arg.
	BaseImage string
	// Dockerfile is the file under images/ that turns BaseImage into a
	// systemd-enabled image. It is selected by package manager (apt, dnf,
	// zypper, ...), not by package format.
	Dockerfile string
	// Format selects the package to install and the package manager
	// commands used by the scenarios.
	Format pkgFormat
}

// distros is the test matrix. Adding a distro that uses an existing package
// manager only requires a new entry here.
var distros = []distro{
	{Name: "debian-13", BaseImage: "debian:13", Dockerfile: "Dockerfile.apt", Format: formatDeb},
	{Name: "rocky-10", BaseImage: "rockylinux/rockylinux:10", Dockerfile: "Dockerfile.dnf", Format: formatRPM},
}

const (
	packageName = "observiq-otel-collector"

	// Package override files sourced by the package scripts.
	debOverridePath = "/etc/default/observiq-otel-collector"
	rpmOverridePath = "/etc/sysconfig/observiq-otel-collector"
)

// formatOps holds the package manager commands for a package format. Each
// func returns a shell snippet that is run inside the container.
type formatOps struct {
	// install installs or upgrades the package, using the same commands as
	// scripts/install/install_unix.sh. downgrade is set when the package
	// is older than the installed one.
	install func(pkg string, downgrade bool) string
	// remove uninstalls the package, matching install_unix.sh --uninstall.
	remove string
	// fileVersion prints the version recorded in a package file.
	fileVersion func(pkg string) string
	// installedVersions prints one line per installed instance of the
	// package, formatted the same as fileVersion. It prints nothing when
	// the package is not installed.
	installedVersions string
	// packageState prints the package manager's view of the package after
	// removal. See uninstallExpect.state.
	packageState string
	// payload lists the paths in a package file, one per line.
	payload func(pkg string) string
	// payloadPrefix is the stage directory prefix of payload entries.
	payloadPrefix string
	// older exits 0 when version a is older than version b.
	older func(a, b string) string
	// overridePath is the package override file sourced by the package
	// scripts. It is part of the expected state on upgrade and uninstall.
	overridePath string
	// uninstall is the per-format expected state after removal.
	uninstall uninstallExpect
}

var ops = map[pkgFormat]formatOps{
	formatDeb: {
		install: func(pkg string, _ bool) string {
			// dpkg allows downgrades, it only prints a warning.
			return "dpkg --force-confold -i " + shellQuote(pkg)
		},
		remove: "dpkg -r " + packageName,
		fileVersion: func(pkg string) string {
			return "dpkg-deb -f " + shellQuote(pkg) + " Version"
		},
		installedVersions: `dpkg-query -W -f='${db:Status-Abbrev}|${Version}\n' ` + packageName + ` 2>/dev/null | awk -F'|' '$1 ~ /^ii/ {print $2}'`,
		packageState:      `dpkg-query -W -f='${db:Status-Abbrev}' ` + packageName + ` 2>/dev/null | tr -d ' ' || true`,
		payload: func(pkg string) string {
			return "dpkg-deb --fsys-tarfile " + shellQuote(pkg) + " | tar -t"
		},
		payloadPrefix: "./usr/share/observiq-otel-collector/stage/observiq-otel-collector/",
		older: func(a, b string) string {
			return fmt.Sprintf("dpkg --compare-versions %s lt %s", shellQuote(a), shellQuote(b))
		},
		overridePath: debOverridePath,
		uninstall:    debUninstall,
	},
	formatRPM: {
		install: func(pkg string, downgrade bool) string {
			if downgrade {
				return "rpm -U --oldpackage " + shellQuote(pkg)
			}
			return "rpm -U " + shellQuote(pkg)
		},
		remove: "rpm -e " + packageName,
		fileVersion: func(pkg string) string {
			return "rpm -qp --qf '%{VERSION}-%{RELEASE}' " + shellQuote(pkg)
		},
		installedVersions: "rpm -q --qf '%{VERSION}-%{RELEASE}\\n' " + packageName + " 2>/dev/null || true",
		packageState:      "if rpm -q " + packageName + " >/dev/null 2>&1; then echo installed; fi",
		payload: func(pkg string) string {
			return "rpm -qlp " + shellQuote(pkg)
		},
		payloadPrefix: "/usr/share/observiq-otel-collector/stage/observiq-otel-collector/",
		older: func(a, b string) string {
			return fmt.Sprintf(`test "$(rpm --eval '%%{lua:print(rpm.vercmp(%q, %q))}')" = "-1"`, a, b)
		},
		overridePath: rpmOverridePath,
		uninstall:    rpmUninstall,
	},
}

// shellQuote single quotes s for use in a POSIX shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
