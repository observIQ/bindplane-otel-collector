---
allowed-tools: Bash(gh:*), Read, Grep, Glob, WebFetch
description: Analyze OpenTelemetry release notes for breaking changes affecting the bindplane collector and enterprise resource types.
---

## User Input

```
$CONTRIB_VERSION $BDOT_REPO $BINDPLANE_REPO
```

- `$CONTRIB_VERSION`: should be a target OTel contrib version (e.g., `v0.148.0`).
- `$BDOT_REPO`: should be a path to local bindplane-otel-collector repository.
- `$BINDPLANE_REPO`: should be a path to local bindplane-op-enterprise repository.

If any arguments are missing ask for them before continuing.

## Context

- Current contrib version in collector go.mod: !`grep 'open-telemetry/opentelemetry-collector-contrib/connector/countconnector' $($BDOT_REPO)/go.mod`
- Current core stable version: !`grep 'go.opentelemetry.io/collector/component ' $($BDOT_REPO)/go.mod`
- Current core unstable version: !`grep 'go.opentelemetry.io/collector/otelcol ' $($BDOT_REPO)/go.mod`

Parse the version numbers (the second field, e.g., `v0.147.0`) from each grep line above in Step 1.

## Your Task

Analyze OpenTelemetry release notes between the current version and the target version to identify breaking changes, deprecations, and notable enhancements that affect the bindplane-otel-collector distribution and its enterprise resource type definitions.

### Step 1: Parse Versions and Compute Range

1. Extract the target version from `$ARGUMENTS`. It must be in the format `v0.X.0`.
2. Extract the current contrib version from the context above.
3. Validate the target is newer than the current version.
4. Compute the version range: all minor versions from `(current_minor + 1)` through `target_minor` inclusive.
   - Example: current `v0.147.0`, target `v0.149.0` -> analyze `v0.148.0` and `v0.149.0`.
5. Compute the core stable version offset from go.mod: if core stable is `v1.S.0` and core unstable is `v0.U.0`, offset = `U - S`. Apply this offset to derive core stable versions for display.

### Step 2: Inventory Imported Components

Read these factory files to build a complete set of upstream OTel components imported into the collector:

- `$BDOT_REPO/factories/receivers.go`
- `$BDOT_REPO/factories/exporters.go`
- `$BDOT_REPO/factories/processors.go`
- `$BDOT_REPO/factories/extensions.go`
- `$BDOT_REPO/factories/connectors.go`

Extract only the `open-telemetry/opentelemetry-collector-contrib` and `go.opentelemetry.io/collector` imports. Build a lookup mapping from release-note component identifiers to Go import paths:
- Release notes use: `receiver/kafka`, `processor/transform`, `exporter/datadog`
- Factory imports use: `receiver/kafkareceiver`, `processor/transformprocessor`, `exporter/datadogexporter`
- The mapping: strip the component-type suffix (receiver, exporter, processor, extension, connector) from the Go package directory name to get the release-note identifier.
- Some names don't follow this pattern exactly (e.g., `storage/filestorage`, `encoding/textencodingextension`). Handle these by also matching the full directory name.

### Step 3: Fetch Release Notes

For each version in the range, fetch release notes using the `gh` CLI:

1. **Core collector**: `gh release view v0.X.0 --repo open-telemetry/opentelemetry-collector --json body -q .body`
2. **Contrib**: `gh release view v0.X.0 --repo open-telemetry/opentelemetry-collector-contrib --json body -q .body`

Focus on the **"End User Changelog"** section. Ignore the "API Changelog" section unless it contains changes affecting component configuration or runtime behavior.

### Step 4: Extract and Filter Relevant Changes

From each release's End User Changelog, extract items from these sections:

| Section | Marker |
|---------|--------|
| Breaking changes | `🛑 Breaking changes 🛑` |
| Deprecations | `🚩 Deprecations 🚩` |
| Enhancements | `💡 Enhancements 💡` |
| Bug fixes | `🧰 Bug fixes 🧰` |

Each item starts with a backtick-quoted component identifier (e.g., `` `receiver/kafka` ``, `` `all` ``). Parse the component identifier and description from each item.

**Filter** to only items where:
- The component identifier matches a component in the imported set from Step 2, OR
- The component identifier is `all`, `pkg/confmap`, `service`, or other core infrastructure that affects all collectors

For enhancements and bug fixes, only include items that describe a **behavioral change**, a **default value change**, or a **feature gate graduation**. Skip purely additive enhancements (new optional features with no change to existing behavior).

### Step 5: Categorize Each Finding

Assign each finding a **type** and **severity**:

**Types:**
- `CONFIG_REMOVED` — Configuration option removed or renamed
- `CONFIG_DEFAULT_CHANGED` — Default value for a config option changed
- `BEHAVIORAL_CHANGE` — Component behavior changed (data placement, format, semantics)
- `COMPONENT_REMOVED` — Entire component removed from the distribution
- `FEATURE_GATE` — Feature gate graduating (alpha->beta = on by default; beta->stable = cannot disable; stable->removed)
- `DEPRECATION` — Something deprecated but still functional
- `BUG_FIX_BEHAVIORAL` — Bug fix that changes observable behavior users may depend on

**Severity:**
- `CRITICAL` — Config option removed with no backward-compatible fallback, component removed, feature gate removed
- `HIGH` — Config option renamed/moved, default value changed, behavioral change affecting data output or placement
- `MEDIUM` — Feature gate graduating but still configurable, deprecation warning
- `LOW` — Informational enhancement, minor behavioral fix

### Step 6: Cross-Reference with Enterprise Resource Types

For each `CRITICAL` or `HIGH` finding, search the enterprise resource type definitions to determine user-facing impact.

1. Use Grep to search for the affected component's Go module path across these directories:
   - `$BINDPLANE_REPO/resources/source-types/`
   - `$BINDPLANE_REPO/resources/destination-types/`
   - `$BINDPLANE_REPO/resources/processor-types/`
   - `$BINDPLANE_REPO/resources/extension-types/`
   - `$BINDPLANE_REPO/resources/connector-types/`

   The YAML files contain `match` blocks with `component:` fields using full Go module paths, e.g.:
   ```yaml
   match:
     - component: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/apachereceiver"
   ```

2. For each matching resource type YAML file:
   - Read the file's template sections to check if the changed/removed config option is referenced
   - Note the resource type `metadata.name` and `metadata.displayName`
   - Check if any parameter defaults need updating

3. Record which resource types are affected and what specific template changes may be needed.

### Step 7: Generate Report

Output the report in this format:

```
# OTel Upgrade Analysis: vCURRENT -> vTARGET

## Summary
- **Version range**: v0.X.0 through v0.Y.0 (N releases)
- **Core versions**: v1.A.0 -> v1.B.0 (stable), v0.X.0 -> v0.Y.0 (unstable)
- **Critical findings**: N
- **High findings**: N
- **Medium findings**: N
- **Low findings**: N

## Critical Findings

### [CRITICAL] <Type>: <component> (v0.X.0)
- **Change**: <description> (#PR)
- **Affected resource types**:
  - `<name>` (<kind>, displayName: "<displayName>")
    - Template impact: <what specifically in the template is affected>
- **Action required**: <concrete action the release engineer must take>

## High Severity Findings

### [HIGH] <Type>: <component> (v0.X.0)
- **Change**: <description> (#PR)
- **Affected resource types**: <list or "None — used directly in collector config only">
- **Action required**: <concrete action>

## Feature Gate Changes

| Gate | Transition | Version | Component | Impact |
|------|-----------|---------|-----------|--------|

## Deprecations

| Component | What | Version | Details |
|-----------|------|---------|---------|

## Notable Enhancements

| Component | Enhancement | Version |
|-----------|------------|---------|

## Action Items Checklist

- [ ] [CRITICAL] <specific action>
- [ ] [HIGH] <specific action>
- [ ] ...
```

## Constraints

- ONLY report on components actually imported in the collector factory files. Do not report on components we do not use.
- Items tagged `all` in the release notes ALWAYS apply — include them regardless.
- ALWAYS fetch release notes using `gh release view`, not WebFetch.
- ALWAYS check BOTH core AND contrib release notes for each version in the range. Do not assume contrib notes include all core changes.
- For the cross-reference step, use Grep first to find matching files, then Read only the matching files. Do not read all resource type YAMLs.
- Do NOT modify any files. This command is read-only analysis.
- Present findings in descending severity order (CRITICAL first).
- If there are no breaking changes affecting imported components, state that explicitly.
- For each finding with an affected resource type, provide a CONCRETE action item — not just "review this".
- When a finding's description is multi-line in the release notes, include the full context — these details matter for understanding the migration path.
