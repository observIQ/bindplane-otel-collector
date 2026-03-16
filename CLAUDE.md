# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is `bindplane-otel-contrib`, a multi-module Go repository containing contrib components (receivers, processors, exporters, extensions) for the [BindPlane OpenTelemetry Collector](https://github.com/observIQ/bindplane-otel-collector). There is **no root go.mod** and **no main binary** — this repo is a library of components consumed by the collector repo.

## Development Commands

### Setup
- `make install-tools` — Install all required development tools
- `./scripts/generate-gowork.sh` — Generate a `go.work` file for IDE/local dev (gitignored)

### Testing
- `make test` — Run all tests with race detection across all modules
- `make test-receivers` / `make test-processors` / `make test-exporters` / `make test-extensions` / `make test-other` — Run tests for a specific component type
- `make test-no-race` — Run all tests without race detection
- `make test-with-cover` — Run tests with coverage reports
- `make bench` — Run benchmarks

### Code Quality
- `make ci-checks` — Run all CI checks (format, license, mod paths, dependabot, misspell, lint, gosec, test)
- `make lint` — Run revive linter
- `make fmt` — Format code with goimports
- `make check-fmt` — Check code formatting
- `make gosec` — Run security scanner
- `make misspell` — Check for misspellings in documentation
- `make check-license` — Check license headers
- `make add-license` — Add license headers to source files

### Module Management
- `make tidy` — Tidy go modules across all submodules
- `make generate` — Run go generate across all modules
- `make check-mod-paths` — Verify all go.mod module paths match directory structure
- `make check-dependabot` — Verify all modules have dependabot entries

### Building the Collector Locally
- `make build-collector` — Build the collector binary using local contrib + collector repos
  - Requires a `.local.env` file with `COLLECTOR_PATH=../bindplane-otel-collector` (auto-created on first run)
  - Creates a temporary `go.work` in the collector repo to use local contrib modules

### Release
- `make release version=vX.X.X` — Tag root + all submodules and push
- `make update-modules NEW_VERSION=vX.X.X` — Update all inter-module version references
- `make update-otel OTEL_VERSION=vX.X.X CONTRIB_VERSION=vX.X.X PDATA_VERSION=vX.X.X` — Update OTel dependencies

## Project Architecture

### Component Organization
- **receiver/** — Custom receivers (AWS S3, M365, Okta, SAP NetWeaver, etc.)
- **processor/** — Custom processors (metric extraction, sampling, masking, etc.)
- **exporter/** — Custom exporters (Azure Blob, Chronicle, Google Cloud, Snowflake, etc.)
- **extension/** — Custom extensions (AWS S3 event, Bindplane extension, OpAMP gateway, etc.)
- **internal/** — Shared internal packages (aws, azureblob, measurements, testutils, etc.)

### Key Architectural Patterns

1. **Multi-module repo** — Each component is a separate Go module with its own `go.mod`. There is no root `go.mod`.
2. **No binary builds** — This repo produces libraries, not executables. The collector repo imports these.
3. **`.local.env` pattern** — Environment overrides (like `COLLECTOR_PATH`) are stored in `.local.env`, which is gitignored.
4. **`go.work` for local dev** — Run `./scripts/generate-gowork.sh` to generate a `go.work` file for IDE support. This file is gitignored.

## Module Management

Each component is its own Go module. When adding or modifying dependencies:
- Run `make tidy` to tidy all modules
- Ensure module paths follow `github.com/observiq/bindplane-otel-contrib/<path>` convention
- Add new modules to `.github/dependabot.yml`
