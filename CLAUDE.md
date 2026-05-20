# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

The Bindplane Distro for OpenTelemetry Collector (BDOT Collector) is Bindplane's distribution of the upstream OpenTelemetry Collector. This is a Go-based project that implements the Open Agent Management Protocol (OpAMP) and supports both standalone and managed modes.

## Development Commands

### Build Commands
- `make agent` - Build the collector via the OTel Collector Builder, using `manifests/observIQ/manifest.yaml` as the source of truth for components. Overwrites the ocb-generated `main.go` with `internal/extension/opampconnectionextension/cmd/main/main.go` so the managed/standalone runtime is wired in. Requires `builder` (`go install go.opentelemetry.io/collector/cmd/builder@v0.151.0`).
- `make verify-manifest` - CI gate: regenerates sources from the manifest and compiles them. Fails if the manifest references unresolvable modules or has version drift. Cheap to run on every PR.
- `make agent-clean` - Wipe the ocb-generated `./build/` tree.
- `make updater` - Build just the updater binary for current OS/architecture
- `make build-binaries` - Build both collector and updater for current OS/architecture (default target)
- `make build-all` - Build for all supported platforms (Linux, Darwin, Windows)
- `make build-linux`, `make build-darwin`, `make build-windows` - Build for specific platforms

### Testing Commands
- `make test` - Run all tests with race detection
- `make test-no-race` - Run all tests without race detection
- `make test-with-cover` - Run tests with coverage reports
- `make test-updater-integration` - Run updater integration tests
- `make bench` - Run benchmarks

### Code Quality Commands
- `make ci-checks` - Run all CI checks (format, license, misspell, lint, gosec, test)
- `make lint` - Run revive linter
- `make fmt` - Format code with goimports
- `make check-fmt` - Check code formatting
- `make gosec` - Run security scanner
- `make misspell` - Check for misspellings in documentation
- `make misspell-fix` - Fix misspellings automatically

### Setup Commands
- `make install-tools` - Install all required development tools
- `make tidy` - Tidy go modules across all submodules
- `make generate` - Run go generate across all modules
- `make add-license` - Add license headers to source files
- `make check-license` - Check license headers

### Release Commands
- `make release version=vX.X.X` - Create and push release tags
- `make release-test` - Test release process locally
- `make release-prep` - Prepare release dependencies

## Project Architecture

### Core Components

The project is structured as an OpenTelemetry Collector distribution with custom components:

- **manifests/observIQ/manifest.yaml** - Canonical source of truth for components and their versions. Drives the `make agent` build.
- **internal/extension/opampconnectionextension/cmd/main/main.go** - Template `main.go` copied over ocb's generated `main.go`; calls into `internal/extension/opampconnectionextension/runtime` for managed/standalone dispatch.
- **internal/extension/opampconnectionextension/** - The OPaMP connection extension and its full managed-mode runtime cluster (collector lifecycle, OPaMP client, package state, report manager, measurements). Its own Go module; entry point `runtime.Run(Options)`.
- **internal/processor/snapshotprocessor/** - Bindplane snapshot processor. Its own internal Go module.

### Component Organization

Custom components are organized by type:
- **receiver/** - Custom receivers (AWS S3, M365, Okta, SAP NetWeaver, etc.)
- **processor/** - Custom processors (metric extraction, sampling, masking, etc.)
- **exporter/** - Custom exporters (Azure Blob, Chronicle, Google Cloud, Snowflake, etc.)
- **extension/** - Custom extensions (AWS S3 event, Bindplane extension)

### Key Architectural Patterns

1. **Dual Mode Operation**: The collector can run in standalone mode (using local config) or managed mode (via OpAMP)
2. **Factory Pattern**: All components are registered through factory functions in the factories package
3. **Module Structure**: Each component is a separate Go module with its own go.mod
4. **Interface Abstraction**: Core collector functionality is abstracted behind interfaces for testability

### Configuration Management

- **Standalone Mode**: Uses local `config.yaml` file
- **Managed Mode**: Configuration delivered via OpAMP from Bindplane server
- **Environment Variables**: Support for OpAMP configuration via environment variables
- **Rollback Support**: Automatic rollback of configurations on startup failures

### Build System

The project uses a Makefile-based build system with:
- **`make agent` runs ocb** against `manifests/observIQ/manifest.yaml`, overlays `internal/extension/opampconnectionextension/cmd/main/main.go`, and compiles. Build flag: `-tags bindplane` (enables Bindplane registry wiring in `topologyprocessor` and `throughputmeasurementprocessor`).
- `make verify-manifest` is the CI gate for manifest correctness — regenerate sources + compile to `/dev/null`.
- There's no top-level `go.mod`. ocb generates a per-build `go.mod` inside `./build/`.
- Multi-platform cross-compilation support
- Separate binaries for collector and updater
- Goreleaser for automated releases
- License scanning and security checks

## Testing Strategy

- Unit tests for all components with race detection
- Integration tests for complex components (updater, receivers)
- Mocking using mockery for interface testing
- Coverage reporting available
- Security scanning with gosec

## Module Management

The project uses a multi-module structure where each custom component is its own Go module. The `make tidy` command operates across all modules. When updating dependencies, use the provided scripts for OTEL version updates.