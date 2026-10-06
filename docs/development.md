# Development

## Initial Setup

Clone this repository, and run `make install-tools`

## Building

To create a build for your current machine, run `make agent`

To build for a specific architecture, see the [Makefile](../Makefile)

To build all targets, run `make build-all`

Build files will show up in the `./dist` directory

## Running Tests

Tests can be run with `make test`.

## Testing Linux packages

The deb and rpm packages can be tested in systemd-enabled distro containers. The suite covers fresh install, upgrade from a previous release, and uninstall. It requires Docker.

```sh
make release-test   # builds the packages into ./dist
make test-packages  # runs the package tests against ./dist
```

See [test/packaging](../test/packaging/README.md) for options and for how to add a distribution.

## Running CI checks locally

The CI runs the `ci-checks` make target, which includes linting, testing, and checking documentation for misspelling.
CI also does a build of all targets (`make build-all`)

## Updating to latest OTEL version

Read through the release notes for the versions of contrib and core that are being updated to. If doing multiple versions look at all version included, i.e. if going from v0.120.0 to v0.122.0 look at v0.121.0 as well. Look for any potential deprecations or breaking changes that could affect Bindplane resources (these are typically in the "End User Changelog" but check the "API Changelog" as well). 

If there are any potential issues with the update, raise concerns in the team channel and coordinate an update plan if necessary.

Most of the process for updating the OTEL dependency is automated with scripts. If at any point there is a failure, try running `make tidy` to see if updating the `go.mod` is able to resolve the issue.
The steps are as follows:

1. Run:
    ```sh
    ./scripts/update-otel.sh {COLLECTOR_VERSION} {CONTRIB_VERSION} {PDATA_VERSION} {CONTRIB_STABLE_VERSION}
    ```
    Grab the different versions from OTEL's GitHub by checking the latest release versions of [collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib) and the [collector](https://github.com/open-telemetry/opentelemetry-collector). They should be the same.
    The pdata version can be found by checking the latest release notes of the collector in a header with {PDATA_VERSION}/{COLLECTOR_VERSION}. The `CONTRIB_STABLE_VERSION` is the version of contrib's stable (v1.0.0+) modules, which are versioned independently of the main contrib release (e.g. the k8sattributes processor). All version arguments should include the v - an example run of the script would look like this:
    ```sh
    ./scripts/update-otel.sh v0.114.0 v0.114.0 v1.20.0 v1.0.0
    ```
    The stable contrib modules are tracked in an allowlist near the top of `update-otel.sh` (and `update-docs.sh`). When a new contrib component reaches v1.0.0, add its module path to both lists. Note: these scripts update the `go.mod` files and docs; bump the matching versions in `manifests/observIQ/manifest.yaml` by hand.

2. Run `make tidy`

3. Run:
    ```sh
    ./scripts/update-docs.sh {COLLECTOR_VERSION} {CONTRIB_VERSION} {BDOT_CONTRIB_VERSION} {CONTRIB_STABLE_VERSION}
    ```
    The collector and contrib versions should be the same as in step 1. The `BDOT_CONTRIB_VERSION` is the latest release version of [dynatrace-bindplane-otel-contrib](https://github.com/dynatrace/dynatrace-bindplane-otel-contrib). The `CONTRIB_STABLE_VERSION` is the same value passed in step 1.

4. Run `make install-tools`

5. Run `make generate`

6. Run `make tidy`

7. Run `make ci-checks`

If all was successful, the repo has had its OTEL dependencies updated to the latest version. 

There is potential for tests to fail, deprecation issues, code changes, or a variety of other problems to arise, but once the above steps are successful the repo can be updated.

## Updating bindplane-otel-contrib dependencies

When [dynatrace-bindplane-otel-contrib](https://github.com/dynatrace/dynatrace-bindplane-otel-contrib) publishes a new release, update the Go module dependencies in this repo:

1. Run:
    ```sh
    make update-contrib BDOT_CONTRIB_VERSION=vx.x.x
    ```
    where `vx.x.x` is the new version of bindplane-otel-contrib.

2. Run `make ci-checks` to verify the update.

## Releasing
To release the agent, see [releasing documentation](releasing.md).
