# Manifests

This folder contains pre-defined manifests that can be built using the [OCB](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder).

Use and modify them as you see fit. Make sure you run `make install-tools` in the root of the project before building any manifests to ensure the builder and supervisor are installed. The [observIQ manifest](./observIQ/README.md) is built by default when running `make agent`. Other manifests can be built running `make distro MANIFEST="/path/to/manifest"`. For example the [minimal manifest](./minimal/README.md) can be built running this:

```
make distro MANIFEST="./manifests/minimal/manifest.yaml"
```

## Pre-defined Manifests

- [observIQ](./observIQ/README.md) -- All components available in this repo, OpenTelemetry-Collector, and OpenTelemetry-Collector-Contrib.
- [minimal](./minimal/README.md) -- The minimal components needed to run the collector with the supervisor and connect to BindPlane OP.
