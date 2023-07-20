# Throughput Measurement Processor
This processor samples OTLP payloads and measures the protobuf size as well as number of OTLP objects in that payload. These measurements are added to the following counter metrics that can be accessed via the collectors internal telemetry service. Units for each `data_size` counter are in Bytes.

Counters:
- `log_data_size`
- `metric_data_size`
- `trace_data_size`
- `log_count`
- `metric_count`
- `trace_count`

## Minimum agent versions
- Introduced: [v1.8.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.8.0)

## Supported pipelines: 
- Logs
- Metrics
- Traces

## Configuration
| Field               | Type   | Default | Description                                                                                                                                                               |
|---------------------|--------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `enabled`           | bool   | `true`  | When `true` signals that measurements are being taken of data passing through this processor. If false this processor acts as a no-op.                                    |
| `sampling_ratio`   | float  | `0.5`   | The ratio of data payloads that are sampled. Values between `0.0` and `1.0`. Values closer to `1.0` mean any individual payload is more likely to have its size measured. |


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

The above configuration will add metrics to the collectors internal metrics service which can be scraped via the `http://localhost:8888/metrics` endpoint. 

More info on the internal metric service can be found [here](https://opentelemetry.io/docs/collector/configuration/#service).


