# OCSF Standardization Processor

This processor is used to create JSON OCSF compliant log bodies from OTEL logs.

## Supported pipelines

- Logs

## Limitations

- OCSF profile validation is not currently supported. Profile fields can still be mapped, but they will not be validated against profile schemas.

## How it works

The processor transforms OpenTelemetry log records into [OCSF](https://schema.ocsf.io/) compliant JSON bodies. Each `event_mapping` defines a filter to match incoming logs, an OCSF `class_id` to assign, and a set of `field_mappings` that map source log fields to OCSF fields. Fields can be populated from the source log (`from`) or set to a static value (`default`).

## Configuration

The following options may be configured:

| Field | Type | Default | Required | Description |
| -- | -- | -- | -- | -- |
| `ocsf_version` | string | | Yes | The OCSF schema version. Supported: `1.0.0` through `1.7.0`. |
| `event_mappings` | []EventMapping | `[]` | No | List of event mappings that define how logs are transformed. |
| `runtime_validation` | bool | `true` | No | Enables runtime OCSF validation of mapped log bodies. When enabled, logs that do not conform to the OCSF schema (missing required fields, invalid enum values, regex/range constraint violations) are dropped. |

### EventMapping

| Field | Type | Default | Required | Description |
| -- | -- | -- | -- | -- |
| `filter` | string | | No | A boolean [expression](https://github.com/expr-lang/expr) to match incoming logs. If empty, all logs match. |
| `class_id` | int | | Yes | The OCSF event class ID. Must be non-zero. |
| `field_mappings` | []FieldMapping | `[]` | No | List of field mappings for the event. |

### FieldMapping

| Field | Type | Default | Required | Description |
| -- | -- | -- | -- | -- |
| `from` | string | | No | An [expression](https://github.com/expr-lang/expr) referencing a source log field. Required if `default` is not set. |
| `to` | string | | Yes | The target OCSF field name. |
| `default` | any | | No | A static default value. Required if `from` is not set. |

### Example Configuration

```yaml
processors:
  ocsf_standardization:
    ocsf_version: "1.3.0"
    event_mappings:
      - filter: 'attributes["event.type"] == "authentication"'
        class_id: 3002
        field_mappings:
          - from: 'body["src_ip"]'
            to: "src_endpoint.ip"
          - from: 'body["user"]'
            to: "actor.user.name"
          - to: "severity_id"
            default: 1
      - filter: 'attributes["event.type"] == "file_system"'
        class_id: 1001
        field_mappings:
          - from: 'body["src_ip"]'
            to: "src_endpoint.ip"
          - from: 'body["dst_ip"]'
            to: "dst_endpoint.ip"
          - from: 'body["src"]'
            to: "src_file.path"
          - from: 'body["dst"]'
            to: "dst_file.path"
          - from: 'body["operation"]'
            to: "operation"
          - from: 'body["action"]'
            to: "action"
```
