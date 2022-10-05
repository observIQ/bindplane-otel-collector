# Throughput Measurement Processor

Supported pipelines: logs, metrics, traces

This processor samples OTLP payloads and measures the protobuf size as well as number of OTLP objects in that payload. These measurements are added to the following counter metrics that can be accessed via internal telemetry. Units for each `data_size` counter are in Bytes.

Counters:
- `log_data_size`
- `metric_data_size`
- `trace_data_size`
- `log_count`
- `metric_count`
- `trace_count`

**NOTE**: This processor can be expensive and time consuming to run, especially at high throughput rates. It is recommended to run only for short periods of time with a modest `sampling_ratio` value.

## Configuration

The following options may be configured:
- `enabled` (default: true): When `true` signals that measurements are being taken of data passing through this processor. If false this processor acts as a no-op.
- `sampling_ratio` (default: 0.5): The ratio of data payloads that are sampled. Values between `0.0` and `1.0`. Values closer to `1.0` mean any individual payload is more likely to have its size measured.

### Example configuration

```yaml
processors:
  throughputmeasurement:
    enabled: true
    sampling_ratio: 0.5
```

