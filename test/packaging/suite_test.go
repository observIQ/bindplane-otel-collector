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
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// Actions are the scenarios the suite can run. Each one ends with the
// package being removed, see scenario.uninstall.
const (
	actionInstall = "install"
	actionUpgrade = "upgrade"
)

var actions = []string{actionInstall, actionUpgrade}

var (
	distroFlag = flag.String("distro", "", "comma separated distros to test, from distros_test.go (default all)")
	actionFlag = flag.String("action", "", "comma separated actions to run: "+strings.Join(actions, ", ")+" (default all)")
)

// suiteEnv is the state shared by every distro and scenario.
type suiteEnv struct {
	docker *testcontainers.DockerClient
	// arch is the goreleaser architecture (amd64, arm64, ...) of the
	// packages under test. It matches the Docker engine so that containers
	// run natively.
	arch string
	dist *dist
	// upgradeFrom are the released versions the upgrade scenario starts
	// from, or upgradeErr when they could not be determined.
	upgradeFrom []string
	upgradeErr  error
}

// TestPackages runs the selected actions against the packages in dist on
// the selected distros. The -distro and -action flags select them. Run it
// with `make test-packages`.
func TestPackages(t *testing.T) {
	names := make([]string, len(distros))
	for i, d := range distros {
		names[i] = d.Name
	}
	runDistros, err := selectNames("distro", *distroFlag, names)
	require.NoError(t, err)
	runActions, err := selectNames("action", *actionFlag, actions)
	require.NoError(t, err)

	env := setupSuite(t)

	for _, d := range distros {
		if !runDistros[d.Name] {
			continue
		}
		t.Run(d.Name, func(t *testing.T) {
			t.Parallel()

			pkg, err := env.dist.linuxPackage(env.arch, d.Format)
			require.NoError(t, err)
			s := &scenario{env: env, distro: d, ops: ops[d.Format], pkg: pkg, image: buildImage(t, d)}

			if runActions[actionInstall] {
				t.Run(actionInstall, func(t *testing.T) {
					t.Parallel()
					s.install(t)
				})
			}
			if runActions[actionUpgrade] {
				if env.upgradeErr != nil {
					t.Run(actionUpgrade, func(t *testing.T) { t.Fatal(env.upgradeErr) })
				}
				for _, from := range env.upgradeFrom {
					t.Run(actionUpgrade+"_from_"+from, func(t *testing.T) {
						t.Parallel()
						s.upgrade(t, from)
					})
				}
			}
		})
	}
}

// selectNames parses the comma separated value of flag name into a set. An
// empty value selects every valid name.
func selectNames(name, value string, valid []string) (map[string]bool, error) {
	set := map[string]bool{}
	if value == "" {
		for _, v := range valid {
			set[v] = true
		}
		return set, nil
	}
	for v := range strings.SplitSeq(value, ",") {
		v = strings.TrimSpace(v)
		if !slices.Contains(valid, v) {
			return nil, fmt.Errorf("-%s: unknown value %q, valid values are: %s", name, v, strings.Join(valid, ", "))
		}
		set[v] = true
	}
	return set, nil
}

// setupSuite validates the environment before any container is started.
func setupSuite(t *testing.T) *suiteEnv {
	t.Helper()

	distDir, err := filepath.Abs(envOr("BDOT_PKGTEST_DIST", filepath.Join("..", "..", "dist")))
	require.NoError(t, err)
	d, err := loadDist(distDir)
	require.NoError(t, err)
	md := d.metadata
	t.Logf("testing %s %s (tag %s, commit %s, built %s) from %s", md.ProjectName, md.Version, md.Tag, md.Commit, md.Date, distDir)

	docker, err := testcontainers.NewDockerClientWithOpts(t.Context())
	require.NoError(t, err, "connect to Docker")
	info, err := docker.Info(t.Context(), client.InfoOptions{})
	require.NoError(t, err, "get Docker info")
	// systemd 256 and newer refuse to boot on cgroup v1.
	require.Equal(t, "2", info.Info.CgroupVersion, "the Docker engine must use cgroup v2 to run systemd containers")

	arch := os.Getenv("BDOT_PKGTEST_ARCH")
	if arch == "" {
		arch = goArch(info.Info.Architecture)
	}
	t.Logf("Docker engine architecture %s, testing linux/%s packages", info.Info.Architecture, arch)

	env := &suiteEnv{docker: docker, arch: arch, dist: d}
	env.upgradeFrom, env.upgradeErr = upgradeVersions(os.Getenv("BDOT_PKGTEST_UPGRADE_FROM"), md.PreviousTag)
	return env
}

// goArch maps a Docker engine architecture (uname -m) to a goreleaser
// architecture.
func goArch(dockerArch string) string {
	switch dockerArch {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	case "armv7l", "armv6l":
		return "arm"
	default:
		return dockerArch
	}
}

var releaseVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// upgradeVersions returns the versions to upgrade from: the comma separated
// override when set, otherwise the previous release tag of the build.
func upgradeVersions(override, previousTag string) ([]string, error) {
	src := "previous_tag in dist/metadata.json"
	list := previousTag
	if override != "" {
		src = "BDOT_PKGTEST_UPGRADE_FROM"
		list = override
	}

	var versions []string
	for v := range strings.SplitSeq(list, ",") {
		// Accept "1.2.3", "v1.2.3", and module tags such as "updater/v1.2.3".
		v = strings.TrimPrefix(path.Base(strings.TrimSpace(v)), "v")
		if !releaseVersionRe.MatchString(v) {
			return nil, fmt.Errorf("%s: %q is not a release version (x.y.z); set BDOT_PKGTEST_UPGRADE_FROM", src, list)
		}
		versions = append(versions, v)
	}
	if len(versions) == 0 {
		return nil, errors.New("no versions to upgrade from; set BDOT_PKGTEST_UPGRADE_FROM")
	}
	return versions, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
