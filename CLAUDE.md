# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

The Bindplane Distro for OpenTelemetry Collector (BDOT Collector) is Bindplane's distribution of the upstream OpenTelemetry Collector. This is a Go-based project that implements the Open Agent Management Protocol (OpAMP) and supports both standalone and managed modes.

## Development Commands

### Build Commands
- `make agent` - Build just the collector binary for current OS/architecture
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

- **cmd/collector** - Main collector entry point that handles both standalone and managed modes
- **collector/** - Core collector wrapper that manages OTel collector lifecycle
- **factories/** - Component factory registration for receivers, processors, exporters, extensions, and connectors
- **opamp/** - OpAMP client implementation for remote management

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