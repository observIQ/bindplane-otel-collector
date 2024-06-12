# Snapshot Processor

The snapshot processor is used in custom distributions of the collector to provide snapshot functionality in BindPlane. It is not currently included in the official `bindplane-agent`.
## Supported pipelines

- Logs
- Metrics
- Traces

## How it works

1. The user configures the processor in one or more pipelines.
2. Whenever telemetry passes through the processor, it is copied and stored temporarily in an internal buffer.
3. An OpAMP server is able to use a custom message to request the contents of the internal buffer, in order to view a snapshot of the telemetry flowing through the collector.

## Configuration

| Field   | Type   | Default | Required | Description                                                            |
|---------|--------|---------|----------|------------------------------------------------------------------------|
| enabled | bool   | `true`  | `false`  | Whether the snapshot processor is enabled or not.                      |
| opamp   | string | `opamp` | `true`   | Specifies the name of the opamp extension for sending custom messages. |


## Examples

### Usage in pipelines

The snapshot processor may be used in a pipeline in order to temporarily catch telemetry data in a buffer, which an opamp server may request:
```yaml
receivers:
  filelog:
    include: [/var/log/logfile.txt]

processors:
  snapshotprocessor:
    enabled: true
    opamp: opamp

exporters:
  nop:

extensions:
  bindplane:
    labels: "labelA=valueA,labelB=valueB"
  opamp:
    endpoint: "https://localhost:3001/v1/opamp"

service:
  extensions: [bindplane, opamp]
  pipelines:
    logs:
      receivers: [filelog]
      processors: [snapshotprocessor]
      exporters: [nop]
```

In this instance, the OpAMP server can now request a snapshot using the `com.bindplane.snapshot` capability (see [request.go](./request.go) for more information on the payload).
