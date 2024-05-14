# BindPlane Extension

This extension is used by BindPlane in custom distributions to store BindPlane specific information. It is not currently included in the official `bindplane-agent`.

## Configuration

| Field  | Type   | Default | Required | Description                                              |
|--------|--------|---------|----------|----------------------------------------------------------|
| labels | string |         | `false`  | Labels for the agent, formatted in `k1=v1,k2=v2` format. |


## Examples

### Setting labels for BindPlane

BindPlane expects a single unnamed bindplane extension in the configuration. It may be used to specify labels:
```yaml
receivers:
  nop:

exporters:
  nop:

extensions:
  bindplane:
    labels: "labelA=valueA,labelB=valueB"

service:
  extensions: [bindplane]
  pipelines:
    logs:
      receivers: [nop]
      exporters: [nop]
```

In this configuration, two labels are specified - `labelA`, with a value of `valueA`, and `labelB`, with a value of `valueB`
