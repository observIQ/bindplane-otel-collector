# Linux package tests

This suite tests the deb and rpm packages built by goreleaser. It uses
[testcontainers-go](https://golang.testcontainers.org/) to install the packages
into systemd-enabled distro containers, and checks the result from Go.

Scenarios, one fresh container each:

| Scenario | What it does |
|---|---|
| `install` | Installs the package, checks files, ownership, modes, the `bdot` user, the systemd unit, and versions. Then it runs `systemctl enable --now`, the same as `install_unix.sh`, and checks that the collector runs. |
| `upgrade_from_<version>` | Installs a released package and starts it. It edits `config.yaml` and `logging.yaml`, and creates `manager.yaml`, a storage file, and the package override file. Then it upgrades to the package under test, checks that those files are preserved, restarts the service, and checks that it runs. |
| `uninstall` | Installs and starts the package, removes it, and checks what is removed and what is kept. |

## Usage

Build the packages, then run the suite:

```sh
make release-test
make test-packages
```

Run a single distro or scenario with `RUN`, which is passed to `go test -run`:

```sh
make test-packages RUN=TestPackages/debian-13
make test-packages RUN=TestPackages/rocky-10/uninstall
```

`PKGTEST_TIMEOUT` (default `45m`) and `PKGTEST_PARALLEL` (default `4`) set the
`go test` timeout and parallelism. Each running container uses about 1 GB of
disk, because the collector binary is copied twice during install.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `BDOT_PKGTEST_DIST` | `../../dist` (`make` passes `./dist`) | goreleaser output directory. It must contain `metadata.json` and `artifacts.json`. |
| `BDOT_PKGTEST_ARCH` | Docker engine architecture | Package architecture to test (`amd64`, `arm64`). Running a different architecture than the Docker engine relies on emulation and is not supported. |
| `BDOT_PKGTEST_UPGRADE_FROM` | `previous_tag` from `metadata.json` | Comma separated release versions to upgrade from, e.g. `1.90.0,1.108.0`. Each one is a separate scenario. |
| `BDOT_PKGTEST_PULL` | unset | When set, pulls the base images again before building the test images. |
| `BDOT_PKGTEST_KEEP` | unset | When set, containers are not removed after the run. Also set `TESTCONTAINERS_RYUK_DISABLED=true`, otherwise Ryuk removes them when the test binary exits. |

Released packages for the upgrade scenario are downloaded from
`https://bdot.bindplane.com` and cached in the user cache directory, e.g.
`~/Library/Caches/bdot-pkgtest` on macOS and `~/.cache/bdot-pkgtest` on Linux.

The upgrade scenario defaults to the previous release because a snapshot build
is older than the release it is based on, according to the package managers.
`1.109.0~SNAPSHOT-abc` sorts before `1.109.0`. If a version passed with
`BDOT_PKGTEST_UPGRADE_FROM` is newer than the package under test, the suite
installs the package as a downgrade (`rpm -U --oldpackage`).

## Requirements

- Docker with **cgroup v2**. Docker Desktop, Colima, OrbStack, and current
  Linux distributions all use cgroup v2. Rootless Docker and Podman are not
  supported.
- **Privileged containers**. systemd needs a writable cgroup filesystem. The
  containers use a private cgroup namespace and do not mount the host's
  `/sys/fs/cgroup`.
- Packages for the Docker engine's architecture in `dist`. `make release-test`
  builds every architecture.
- On amd64 hosts, EL10 based images (Rocky, Alma, RHEL) need an x86-64-v3 (AVX2)
  capable CPU.
- `TESTCONTAINERS_HUB_IMAGE_NAME_PREFIX` must not be set. The test images are
  built locally and must not be pulled from a registry.

With Colima, set `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock` if
the Ryuk container fails to start. See
[using Colima](https://golang.testcontainers.org/system_requirements/using_colima/).

## Adding a distribution

Add an entry to `distros` in [distros_test.go](distros_test.go):

```go
{Name: "almalinux-10", BaseImage: "almalinux:10", Dockerfile: "Dockerfile.dnf", Format: formatRPM},
```

`Dockerfile` selects the image recipe in [images](images) by package manager.
`Format` selects the package and the `dpkg`/`rpm` commands. A distro with a new
package manager, such as openSUSE with zypper, needs a new `images/Dockerfile.<manager>`.
It should install systemd and the tools the package scripts use, following the
existing files.

## Expected state

The expected state is defined in [expect_test.go](expect_test.go), based on
`.goreleaser.yml` and `scripts/package/*.sh`. When a package change is
intended, update the expectations in the same change.

deb and rpm currently behave differently on uninstall. Those entries are marked
with `divergence:` in `expect_test.go`. The table records current behavior, so a
change in either direction fails the suite.

| Path | `dpkg -r` | `rpm -e` |
|---|---|---|
| package state | `rc` (config remains) | not installed |
| `config.yaml`, `logging.yaml` | kept | removed (`%ghost`) |
| `LICENSE` | kept | removed (`%ghost`) |
| systemd unit file | kept, disabled | removed (`%ghost`) |
| package override file | kept (`/etc/default`) | removed (`/etc/sysconfig`, `%ghost`) |

## Debugging

When a scenario fails, the suite logs the container's systemd output, the unit
status and journal, the tail of `collector.log`, and a listing of the install
directory. It also logs the output of every package manager command, including
the package scripts.

To inspect a container after a run, keep it and exec into it:

```sh
BDOT_PKGTEST_KEEP=1 TESTCONTAINERS_RYUK_DISABLED=true make test-packages RUN=TestPackages/debian-13/install
docker exec -it <container id from the test log> bash
```

Remove kept containers with `docker rm -f`.

The test images are kept to reuse the Docker build cache. Remove them with:

```sh
docker image rm bdot-pkgtest:debian-13 bdot-pkgtest:rocky-10
```
