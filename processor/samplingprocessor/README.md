# Sampling Processor

This processor samples incoming OTLP objects and drops those objects based on a configured `drop_ratio`.

## Supported pipelines

- Logs
- Metrics
- Traces

## How it works

1. The user configures the processor in their pipeline with a `drop_ratio` that is the desired.
2. A number between 0 and 1 will be randomly generated for each piece incoming telemetry data.
3. If the generated number is less than or equal to the `drop_ratio`, then the telemetry data is dropped.
4. If the generated number is greater than the `drop_ratio`, then the telemetry data makes it further in the pipeline.

## Configuration

The following options may be configured:

| Field | Type | Default | Description |
| -- | -- | -- | -- |
| drop_ratio | float | 0.5 | The ratio of payload objects that are dropped. Values between `0.0` and `1.0`. Values closer to `1.0` mean any individual object in a payload is more likely to be dropped. |

### Example Configuration

The following config is an example configuration of the `sampling` processor with defaults in a logs pipeline sending to the `logging` exporter.

```yaml
receivers:
  filelog:
    include: [/tmp/example/apache.log]
processors:
  sampling:
    drop_ratio: 0.5
exporters:
  logging:

service:
  pipelines:
    logs:
      receivers: [filelog]
      processors: [sampling]
      exporters: [logging]
```

## How to

### Sample 75% of incoming telemetry

The following configuration will drop 75% of incoming `metrics`, `logs` or `traces`.

```yaml
processors:
  sampling:
    drop_ratio: 0.75
```

### Drop all incoming telemetry

The following configuration will drop 100% of incoming `metrics`, `logs`, or `traces`. Essentially dropping all data.

```yaml
processors:
  sampling:
    drop_ratio: 1.0
```
