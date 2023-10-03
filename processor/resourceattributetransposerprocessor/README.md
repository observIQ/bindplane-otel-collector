# Resource Attribute Transposer Processor
This processor copies a resource level attribute to all individual logs or metric data points associated with the resource.
If the key already exists, no action is taken (the attribute _**IS NOT**_ overwritten)

## Minimum agent versions
- Introduced: [v0.0.12](https://github.com/observIQ/bindplane-agent/releases/tag/v0.0.12)
- Introduced support for logging pipelines: [v1.0.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.0.0)

## Supported pipelines
- Logs
- Metrics

## How it works
1. The user configures the resource attribute transposer processor in the desired logs/metrics pipeline.
2. For every log/metric datapoint, all resource attributes are copied from the resource attribute specified in the "from" field, to the log or datapoint attribute specified in the "to" field.
3. If the attribute specified by the "to" field already exists, it is not overwritten.

## Configuration
| Field               | Type   | Default | Description                                                            |
|---------------------|--------|---------|------------------------------------------------------------------------|
| `operations`        | []map  | `[]`    | A list of operations to apply to each metric or log resource.          |
| `operations[].from` | string | `""`    | The attribute to copy off of the resource.                             |
| `operations[].to`   | string | `""`    | The destination attribute on each individual metric data point or log. |

### Example configuration

This example configuration shows how you can use the resource attribute transposer to copy the resource attributes for mongodbatlas logs to labels in GCP.

```yaml
receivers:
  mongodbatlas:
    public_key: $MONGODB_ATLAS_PUBLIC_KEY
    private_key: $MONGODB_ATLAS_PRIVATE_KEY
    logs:
      enabled: true
      projects:
        - name: "MyProject"
          collect_audit_logs: true

exporters: 
  googlecloud:

processors:
  resourceattributetransposer:
    operations:
      # Log resource attributes
      - from: "mongodb_atlas.org"
        to: "mongodb_atlas.org"
      - from: "mongodb_atlas.project"
        to: "mongodb_atlas.project"
      - from: "mongodb_atlas.cluster"
        to: "mongodb_atlas.cluster"
      - from: "mongodb_atlas.host.name"
        to: "mongodb_atlas.host.name"

service:
  pipelines:
    logs:
      receivers:
      - mongodbatlas
      processors:
      - resourceattributetransposer
      exporters:
      - googlecloud
```

The configuration above copies the `mongodb_atlas` prefixed resource attributes from the mongodb logs to the attributes of the log entry.
This allows the resource attributes to be mapped to log labels in GCP.

## Limitations

Currently, this assumes that the resources attributes is a flat map. This means that you cannot move a single resource attribute if it is under a nested map. You can, however, move a whole nested map.

