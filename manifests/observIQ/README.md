# observIQ manifests

Three manifests live here, all driving ocb builds:

| File | Build target | Output binary | Shape |
|---|---|---|---|
| `manifest.yaml` | `make agent` | `dist/collector_<os>_<arch>` | v1 — in-process managed runtime. `buildscripts/main.go` is overlaid onto ocb's generated `main.go`; the binary owns the OpAMP client and restarts the collector on remote config push. |

## Installing ocb

```
make install-ocb
```

The ocb version is pinned via `OCB_VERSION` in the root Makefile and matches the OTel core version pinned in all three manifests.

## Building

From the repo root:

```
make agent          # v1
```

`make agent` runs `builder --skip-compilation` against `manifest.yaml`, overwrites the generated `build/main.go` with `internal/extension/opampconnectionextension/cmd/main/main.go` (so the managed/standalone runtime is wired in), then `go build`s into `./dist/`. `./build/` is gitignored ocb output; the final binary is `./dist/collector_<os>_<arch>`.

`make verify-manifest` regenerates sources from `manifest.yaml` and compiles to `/dev/null` — the CI gate.

`make agent-clean` wipes `./build/` (v1 ocb output) and `./builder/` (v2 ocb output).

## Editing a manifest

- **Add a component**: append a `gomod:` entry under the right section. Format: `<module path> <version>`.
- **Bump a dependency**: change the version in `gomod:`. Re-run the build. Commit the manifest change; ocb output dirs are gitignored.
- **Pin a transitive dep or redirect a fork**: add to `replaces:`. These apply to release builds too.

## v1 manifest internals

Two components are internal modules under this repo:

- `internal/extension/opampconnectionextension` — bindplane's OpAMP connection extension and the full v1 managed-mode runtime cluster (collector lifecycle, OpAMP client, package state, report manager, measurements, service dispatch).
- `internal/processor/snapshotprocessor` — bindplane snapshot processor.

The v1 manifest references each by its own `gomod:` entry and a narrow local `replace:` pointing at the on-disk path. They stay under `internal/` until they're ready to be published.

See [`docs/specs/ocb-canonical-build.md`](../../docs/specs/ocb-canonical-build.md) for the broader design context.
