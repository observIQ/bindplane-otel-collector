# Throughput Measurement Processor

This processor samples OTLP payloads and measures the protobuf size as well as number of OTLP objects in that payload. These measurements are added to the following counter metrics that can be accessed via the collectors internal telemetry service. Units for each `data_size` counter are in Bytes.

Counters:

- `log_data_size` - The size of the log payload, including all attributes, headers, and metadata
- `log_raw_bytes` - The raw byte size of the log body payload
- `metric_data_size` - The size of the metric payload, including all attributes, headers, and metadata
- `trace_data_size` - The size of the trace payload, including all attributes, headers, and metadata
- `log_count` - The number of log records in the payload
- `metric_count` - The number of metric data points in the payload
- `trace_count` - The number of trace spans in the payload

## Minimum agent versions

- Introduced: [v1.8.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.8.0)

## Supported pipelines:

- Logs
- Metrics
- Traces

## Configuration

| Field                     | Type                    | Default | Description                                                                                                                                                                          |
| ------------------------- | ----------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `enabled`                 | bool                    | `true`  | When `true` signals that measurements are being taken of data passing through this processor. If false this processor acts as a no-op.                                               |
| `sampling_ratio`          | float                   | `0.5`   | The ratio of data payloads that are sampled. Values between `0.0` and `1.0`. Values closer to `1.0` mean any individual payload is more likely to have its size measured.            |
| `measure_log_raw_bytes`   | bool                    | `false` | When `true`, for logs, the processor will measure the raw bytes of the payload in addition to the protobuf size. This is more expensive but provides raw measurements if designated. |
| `measure_log_raw_fields`  | []RawFieldWithFallback  | `null`  | When set, for logs, the processor will measure the raw bytes of specific log fields. Each field configuration includes a primary field path and a fallback field path. If the primary field is not present, the fallback field will be measured instead. This is more expensive than standard protobuf measurements but provides granular raw byte measurements for specific log attributes or body content. |

### Example configuration

The example configuration below shows ingesting logs and sampling the size of 50% of the OTLP log objects.

```yaml
receivers:
  filelog:
    inclucde: ["/var/log/*.log"]

processors:
  throughputmeasurement:
    enabled: true
    sampling_ratio: 0.5

exporters:
  googlecloud:

service:
  pipelines:
    logs:
      receivers:
        - filelog
      processors:
        - throughputmeasurement
      exporters:
        - googlecloud
```

### Advanced configuration with raw field measurement

The example configuration below shows how to measure specific log fields in addition to the standard protobuf size measurements.

```yaml
receivers:
  filelog:
    include: ["/var/log/*.log"]

processors:
  throughputmeasurement:
    enabled: true
    sampling_ratio: 0.5
    measure_log_raw_bytes: true
    measure_log_raw_fields:
      - field: "attributes.log.record.original"
        fallback_field: "body"

exporters:
  googlecloud:

service:
  pipelines:
    logs:
      receivers:
        - filelog
      processors:
        - throughputmeasurement
      exporters:
        - googlecloud
```

The above configuration will add metrics to the collectors internal metrics service which can be scraped via the `http://localhost:8888/metrics` endpoint.

More info on the internal metric service can be found [here](https://opentelemetry.io/docs/collector/configuration/#service).
