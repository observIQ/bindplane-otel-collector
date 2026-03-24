# Bindplane ProcessorType – Threat Enrichment

`threat-enrichment-processor-type.yaml` defines a **ProcessorType** for Bindplane that matches the threat enrichment processor in this repo.

## Aligns with

- **Component:** `github.com/observiq/bindplane-otel-collector/processor/threatenrichmentprocessor`
- **Config:** `filter` (default algorithm + params) + `rules[]` (name, indicator_file, lookup_fields) + optional `storage`

## Using in your other repo

1. Copy `threat-enrichment-processor-type.yaml` into your Bindplane/agent management repo (e.g. wherever you keep other `ProcessorType` CRs).
2. Adjust `metadata.labels` and `spec.supportedPlatforms` if needed.
3. If your template engine does not support the `quote` filter, remove `| quote` from the template and use plain `{{ .rule_1_name }}` etc. (YAML accepts unquoted values for simple strings).

## Parameters summary

| Section        | Parameters |
|----------------|------------|
| Storage        | `enable_storage`, `storage_id` (when enabled) |
| Default filter | `filter_kind` (bloom / cuckoo / scalable_cuckoo / vacuum); bloom: `filter_estimated_count`, `filter_false_positive_rate`, optional `filter_max_estimated_count`; cuckoo/vacuum: `filter_capacity` / `filter_capacity_vacuum`; scalable_cuckoo: `filter_initial_capacity`, `filter_load_factor`; vacuum: `filter_fingerprint_bits` |
| Rule 1         | `rule_1_name`, `rule_1_indicator_file`, `rule_1_lookup_fields` (`strings`: attribute keys or `body`) |
| Rule 2         | `rule_2_enabled`, then `rule_2_name`, `rule_2_indicator_file`, `rule_2_lookup_fields` |
| Rule 3         | `rule_3_enabled`, then `rule_3_name`, `rule_3_indicator_file`, `rule_3_lookup_fields` |

Parameter types match Bindplane (`float` not `double`, `strings` for `lookup_fields`). The Bindplane UI for this ProcessorType is limited to **three** rules (rule 1 plus optional rules 2 and 3); the collector `threatenrichment` processor accepts an arbitrary-length `rules` list if you author YAML manually. The collector config also supports an optional per-rule `filter` override; this ProcessorType does not expose that (use a custom processor block if needed).

---

## Threat Detection Rule (`threat-detection-rule-processor-type.yaml`)

`threat-detection-rule-processor-type.yaml` defines a **ProcessorType** that renders one **detection rule** as a contrib **transform** processor: an OTTL `conditions` block gates `statements` that set `detection.*` attributes (and optionally severity).

- **Component:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor` (already included in this distribution; see `factories/processors.go`).
- **Match:** `transformprocessor` `>= 0.0.0` in the ProcessorType spec aligns with the collector’s pinned contrib version.

Copy this YAML into the repo where you store Bindplane `ProcessorType` resources, same as threat enrichment.

### Validating Bindplane output (checklist)

After you add a Threat Detection Rule processor in Bindplane and render or deploy configuration:

1. **Unique transform component IDs** — Under `processors:`, each rule instance must have a **distinct** key (e.g. `transform/p-…__processor0`). If every instance emitted the same literal `transform/threat_detection`, the collector would reject duplicate IDs; adjust the template to use your platform’s per-instance name if needed.
2. **OTTL from `bpConditionOTTL .condition`** — The rendered `conditions:` entry must be a **single valid log-context OTTL boolean**. Smoke-test with a trivial condition, then your real rule; collector startup will fail on parse errors.
3. **Structured `log_statements`** — Expect `context: log`, `conditions:`, and `statements:` (structured style), not a flat-only list of statement strings.
4. **Template helpers** — Confirm `join` and `| quote` exist in your Bindplane template environment (tags/references and string safety).

A **reference fragment** (not a full collector config) lives at [`examples/threat-detection-rule-rendered.example.yaml`](examples/threat-detection-rule-rendered.example.yaml).

### Pipeline order: Threat Enrichment vs Threat Detection Rule

| Goal | Suggested order |
|------|-----------------|
| Detection OTTL should see IOC enrichment (`threat.matched`, `threat.rule`, etc.) | **Threat Enrichment** → **Threat Detection Rule** (transform) instances |
| Rules only on raw / pre-enrichment fields | **Threat Detection Rule** → **Threat Enrichment** |

Threat enrichment uses file-backed indicator sets (bloom/cuckoo/etc.) at startup; detection rules use arbitrary OTTL. They are complementary.

### Multiple detection rules on one pipeline

Each Threat Detection Rule instance targets the same **flat** attribute keys (`detection.rule.name`, `detection.matched`, …). If **more than one** rule’s condition matches the same log, **the later processor in the pipeline overwrites** the earlier rule’s `detection.*` values (last writer wins). Plan routing or downstream logic accordingly, or use separate pipelines if you need to preserve every hit separately.
