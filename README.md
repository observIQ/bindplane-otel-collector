# bindplane-otel-contrib

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Contrib components (receivers, processors, exporters, extensions) for the [BindPlane OpenTelemetry Collector](https://github.com/observIQ/bindplane-otel-collector).

## Directory Structure

```
receiver/       Custom receivers
processor/      Custom processors
exporter/       Custom exporters
extension/      Custom extensions
internal/       Shared internal packages
pkg/            Public utility packages
  counter/      Counter module
  expr/         Expression module
  snapshot/     Snapshot module
  version/      Version module
```

## Development

### Prerequisites

- Go (version specified in `internal/tools/go.mod`)

### Setup

```bash
# Install development tools
make install-tools

# Generate a go.work file for IDE support
./scripts/generate-gowork.sh
```

### Common Commands

```bash
# Run all tests
make test

# Run all CI checks (format, license, lint, gosec, test)
make ci-checks

# Build the collector using the local collector repo
make build-collector

# Lint
make lint

# Format code
make fmt

# Tidy all modules
make tidy
```

### Local Development with Collector

Create a `.local.env` file to configure the path to your local collector repo:

```bash
COLLECTOR_PATH=../bindplane-otel-collector
```

Then run `make build-collector` to build the collector binary with your local contrib changes.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
