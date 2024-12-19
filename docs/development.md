# Development

## Initial Setup

Clone this repository, and run `make install-tools`

## Building

To create a build for your current machine, run `make collector`

To build for a specific architecture, see the [Makefile](../Makefile)

To build all targets, run `make build-all`

Build files will show up in the `./dist` directory

## Running Tests

Tests can be run with `make test`.

## Running CI checks locally

The CI runs the `ci-checks` make target, which includes linting, testing, and checking documentation for misspelling.
CI also does a build of all targets (`make build-all`)

## Updating to latest OTEL version

Most of the process for updating the OTEL dependency is automated with scripts. If at any point there is a failure, try running `make tidy` to see if updating the `go.mod` is able to resolve the issue.
The steps are as follows:

1. Run:
    ```sh
    ./scripts/update-otel.sh {COLLECTOR_VERSION} {CONTRIB_VERSION} {PDATA_VERSION}
    ```
    Grab the different versions from OTEL's GitHub by checking the latest release versions of [collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib) and the [collector](https://github.com/open-telemetry/opentelemetry-collector). They should be the same.
    The pdata version can be found by checking the latest release notes of the collector in a header with {PDATA_VERSION}/{COLLECTOR_VERSION}. All three version arguments should include the v - an example run of the script would look like this:
    ```sh
    ./scripts/update-otel.sh v0.114.0 v0.114.0 v1.20.0
    ```

2. Run `make tidy`

3. Run:
    ```sh
    ./scripts/update-docs.sh {COLLECTOR_VERSION} {CONTRIB_VERSION}
    ```
    These should be the same versions used in the first step.

4. Run `make install-tools`

5. Run `make generate`

6. Run `make tidy`

7. Run `make ci-checks`

If all was successful, the repo has had its OTEL dependencies updated to the latest version. 

There is potential for tests to fail, deprecation issues, code changes, or a variety of other problems to arise, but once the above steps are successful the repo can be updated.

## Releasing
To release the agent, see [releasing documentation](releasing.md).
