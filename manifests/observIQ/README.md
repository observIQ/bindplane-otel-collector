# observIQ manifest

`manifest.yaml` is the canonical source of truth for the components in the BDOT Collector build. Dependency versions, replaces, and the component graph all live here. Edit this file to add/remove/bump components.

## Build

Install ocb (matching the OTel core version pinned in this manifest):

```
go install go.opentelemetry.io/collector/cmd/builder@v0.151.0
```

Then from the repo root:

```
make agent-ocb
```

The target runs `builder --skip-compilation` against this manifest, overwrites the generated `build/main.go` with `internal/extension/opampconnectionextension/cmd/main/main.go` (so the managed/standalone runtime in `collector/`, `opamp/`, and `internal/service/` is wired in), then `go build`s the binary into `./dist/`. `./build/` is gitignored ocb output; the final binary is `./dist/collector_<os>_<arch>`.

To run ocb directly without the make target:

```
builder --config manifests/observIQ/manifest.yaml
```

That produces a standalone-mode-only binary (ocb's default `main.go`, no OPaMP wiring). Useful for verifying the manifest itself; not the shipping flow.

## Editing the manifest

- **Add a component**: append a `gomod:` entry under the right section. Format: `<module path> <version>`.
- **Bump a dependency**: change the version in `gomod:`. Re-run `builder`. Commit the manifest change; `build/` stays gitignored.
- **Pin a transitive dep or redirect a fork**: add to `replaces:`. These are not local-dev paths — they apply to release builds too.

## Internal components

Two components currently live under `internal/` of the top-level module: `opampconnectionextension` and `snapshotprocessor`. The manifest references them via the parent module path plus a local `replaces:` entry (`github.com/observiq/bindplane-otel-collector => ../`). When these packages are relocated to their own Go modules (Phase 1 of the [ocb-canonical build spec](../../docs/specs/ocb-canonical-build.md)), the local replace will be replaced with per-component module pins.
