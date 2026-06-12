d# ocb-Canonical Build — as built

This describes the BDOT Collector build as it stands today. The OTel Collector Builder (ocb) compiles every binary from a manifest. The v1 runtime cluster (OpAMP client, collector lifecycle, package state, report manager, measurements) lives in a single internal Go module.

## What ocb owns

1. Reads `manifests/observIQ/manifest.yaml` (or one of the v2 variants).
2. Generates `./build/components.go` and `./build/go.mod` (v1) or `./builder/...` (v2).
3. For v1: a Make step copies `internal/extension/opampconnectionextension/cmd/main/main.go` over ocb's generated `main.go`, runs `go mod tidy`, then `go build`.
4. For v2: ocb's generated `main.go` is used as-is; ocb runs `go build` itself.

`./build/` and `./builder/` are gitignored; the compiled binary lands in `./dist/`.

## Manifests

| File | Build target | Output binary | Shape |
|---|---|---|---|
| `manifest.yaml` | `make agent` | `dist/collector_<os>_<arch>` | v1 — in-process managed runtime; `internal/extension/opampconnectionextension/cmd/main/main.go` overlay. |
| `manifest-v2.yaml` | `make agent-v2` | `dist/collector_v2_<os>_<arch>` | v2 — vanilla collector; remote management lives in the external `opampsupervisor`. Verbatim from the v2.0.1-beta.3 release. |
| `manifest-v2-aix.yaml` | `GOOS=aix GOARCH=ppc64 make agent-v2-aix` | `dist/collector_v2_aix_ppc64` | v2 trimmed for AIX/ppc64. Excludes components that don't build on big-endian ppc64. |

`make verify-manifest` regenerates from `manifest.yaml` and compiles to `/dev/null` — the CI gate against manifest breakage.

`make agent-clean` wipes `./build/` and `./builder/`.

## File layout

```
bindplane-otel-collector/
├── manifests/
│   └── observIQ/
│       ├── manifest.yaml
│       ├── manifest-v2.yaml
│       ├── manifest-v2-aix.yaml
│       └── README.md
│
├── internal/
│   ├── extension/
│   │   └── opampconnectionextension/    # own Go module
│   │       ├── go.mod
│   │       ├── extension.go             # OTel extension factory
│   │       ├── factory.go
│   │       ├── config.go
│   │       ├── registry.go              # Client interface + GetRegistry
│   │       ├── available_components.go
│   │       │
│   │       ├── cmd/main/                # v1 entry point overlaid onto ocb's build/main.go
│   │       │   ├── main.go              # the actual main; copied to build/main.go
│   │       │   └── components_stub.go   # source-tree stub of components() (not copied)
│   │       │
│   │       ├── runtime/                 # entry point used by main.go
│   │       │   ├── run.go               # Options + Run
│   │       │   └── bootstrap.go         # TLS env, manager.yaml init, rollback
│   │       │
│   │       ├── collector/               # otelcol lifecycle wrapper
│   │       ├── opamp/                   # OpAMP types + client interface
│   │       │   └── observiq/            # default Bindplane client implementation
│   │       ├── packagestate/            # package-install state machine
│   │       ├── report/                  # snapshot/measurements/topology senders
│   │       ├── measurements/            # throughput measurements registry
│   │       ├── service/                 # managed/standalone runnable wrapper
│   │       └── logging/                 # zap config loader
│   │
│   └── processor/
│       └── snapshotprocessor/           # own Go module
│           └── go.mod
│
├── updater/                             # own Go module
├── version/, expr/, counter/            # untouched sub-modules
│
├── build/                               # v1 ocb output (gitignored)
├── builder/                             # v2 ocb output (gitignored)
└── dist/                                # compiled binaries (gitignored)
```

No top-level `go.mod`. No `cmd/collector/`. No `factories/`. The legacy `opamp/`, `collector/`, `packagestate/`, `internal/{logging,service,report,measurements}/` directories are gone — that code now lives under `internal/extension/opampconnectionextension/`.

## v1 entry point

`internal/extension/opampconnectionextension/cmd/main/main.go` is ~100 lines: parse flags, build factories from ocb's `components()`, call `runtime.Run(Options)`. Everything else lives in the extension module.

```go
package runtime

type Options struct {
    Factories            otelcol.Factories
    Version              string
    CollectorConfigPaths []string
    ManagerConfigPath    string
    LoggingConfigPath    string
    FeatureGates         []string
}

func Run(opts Options)
```

`Run` configures logging from `LoggingConfigPath`, sets feature gates, builds a collector, checks `ManagerConfigPath` (creates one from `OPAMP_*` env vars if absent), then launches either the managed (OpAMP) or standalone runnable service. Terminal errors go through `log.Fatal` / `logger.Fatal`.

The v1 binary dispatches managed vs standalone at runtime: `manager.yaml` exists → managed; absent and no `OPAMP_*` env vars → standalone.

## v2 entry point

ocb's default `main.go`, no overlay. The binary is a vanilla otel-collector with the upstream `opampextension`. Remote management is the supervisor's responsibility, out of process.

## Build flag

`-tags bindplane` is required on the v1 build. Two contrib processors — `topologyprocessor` and `throughputmeasurementprocessor` — gate their Bindplane registry wiring on this tag. Without it those processors no-op on Bindplane state. The v1 `make agent` target sets it. v2 doesn't need it (no in-process Bindplane registry).

## Replaces in manifests

`manifest.yaml` carries:

- Two narrow local replaces pointing at the on-disk paths of the internal modules: `internal/extension/opampconnectionextension` and `internal/processor/snapshotprocessor`.
- Six pinning replaces inherited from the legacy `go.mod`: `mattn/go-ieproxy v0.0.1`, `cilium/ebpf v0.11.0`, `DataDog/datadog-agent/comp/core/delegatedauth v0.78.2`, plus three observIQ forks (`windowseventlogreceiver`, `pkg/stanza`, `azureblobreceiver`).

`manifest-v2.yaml` and `manifest-v2-aix.yaml` are unmodified copies of v2.0.1-beta.3 and use a different replace set (notably `cockroachdb/errors v1.13.0` to fix a sentry-go API break).

## Open items

Future work:

- **Goreleaser config.** Still references `cmd/collector` and the legacy go-build flow. Needs to invoke `make agent` (or run ocb directly with the same recipe) per platform, and to do `make agent-v2` / `agent-v2-aix` for the v2 builds.
- **CI workflows in `.github/workflows/`.** Build jobs that compiled from the top-level `go.mod` need to compile via the manifest. `verify-manifest` should run on every PR that touches a manifest or any sub-module.
- **Dependabot.** Currently bumps `go.mod` files; needs to also bump versions inside `manifest.yaml` and the v2 variants.
- **Docker image.** `docker/` references the compiled binary path; the `dist/collector_*` naming hasn't changed, but the end-to-end image build hasn't been re-verified.
- **Build-info ldflags parity.** Today the build only stamps `version`. Legacy/v2 release flows likely also wanted git hash and build date.
