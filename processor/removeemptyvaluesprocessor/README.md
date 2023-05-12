# Remove Empty Values Processor

This processor removes empty values from telemetry's attributes and resource attributes, as well as from log record's body.

## Supported pipelines

- Logs
- Metrics
- Traces

## How it works

1. The user configures the processor in their pipeline, optionally configuring `empty_string_values` with a list of string values that are considered "empty".
2. For each piece of telemetry data, each entry in the resource attributes, the attributes, and the log record body (if it is a map) is visited.
3. Map entries are removed if the value is null, or if the value is one of the string values contained in `empty_string_values`. Optionally, empty maps and lists may be removed by configuring the `remove_empty_lists` and `remove_empty_maps` settings.
4. The telemetry data is then passed to the next component in the pipeline.

## Configuration

The following options may be configured:

| Field | Type | Default | Description |
| -- | -- | -- | -- |
| remove_nulls | bool | `true` | If true, entries with a value of null are removed. |
| remove_empty_lists | bool | `false` | If true, entries with a value of an empty list are removed. |
| remove_empty_maps | bool | `false` | If true, entries with a value of an empty map are removed. |
| empty_string_values | []string | `[]` | A list of case-insensitive string values considered "empty". |
| exclude_keys | []string | `[]` | A list of keys to exclude from removal. These keys are in the format of `<field>.<path-to-key>` (e.g. `resource.k8s.pod.id`). You may also just specify `<field>` to exclude the whole field. Valid fields are `body`, `resource`, and `attributes`. |

### Example Configuration

The following config is an example configuration of the `removeemptyvalues` processor with defaults in a logs pipeline sending to the `logging` exporter.

```yaml
receivers:
  windowseventlog:
    channel: application

processors:
  removeemptyvalues:

exporters:
  logging:

service:
  pipelines:
    logs:
      receivers: [windowseventlog]
      processors: [removeemptyvalues]
      exporters: [logging]
```

## How to

### Remove fields with empty string values
The following configuration removes fields that are the empty string ("") or null from Windows Event Logs:

```yaml
receivers:
  windowseventlog:
    channel: application

processors:
  removeemptyvalues:
    empty_string_values:
      # "" must be explicitly defined to remove empty strings
      - ""

exporters:
  logging:

service:
  pipelines:
    logs:
      receivers: [windowseventlog]
      processors: [removeemptyvalues]
      exporters: [logging]
```

### Remove empty fields from nginx logs

The following configuration removes empty fields from nginx logs, where empty fields have a value of "-".

```yaml
receivers:
  plugin:
    path: "./plugins/nginx_logs.yaml"

processors:
  removeemptyvalues:
    empty_string_values:
      # Remove fields with the value of "-"
      - "-"

exporters:
  logging:

service:
  pipelines:
    logs:
      receivers: [plugin]
      processors: [removeemptyvalues]
      exporters: [logging]
```

