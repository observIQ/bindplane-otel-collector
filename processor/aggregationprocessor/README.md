# Aggregation Processor
This processor aggregates metrics over a configurable interval, allowing for metrics to be sampled at a higher rate, or to reduce the volume of metric data from push-based sources.

## Minimum collector versions
- Introduced: [v1.19.0](https://github.com/observIQ/observiq-otel-collector/releases/tag/v1.19.0)

## Supported pipelines
- Metrics

## How it works
1. The user configures the aggregation processor in the desired metrics pipeline.
2. Every metric that flows through the pipeline is matched against the provided `include` regex.
3. If the metric name does not match the `include` regex, the metric passes through the processor.
4. If the metric matches, but is not a gauge or cumulative sum, the metric passes through the processor.
5. If the metric name does match, and the metric is a gauge or cumulative sum, the metric is added to an aggregate based on its attributes. The metric does not continue down the pipeline.
6. After the configured `interval` has passed, all aggregate metrics are emitted. Aggregate metrics are emitted with "${metric_name}.${aggregation}" e.g. if you take the average of the metric `system.cpu.utilization`, the aggregated metric would be `system.cpu.utilization.avg`.
7. All aggregations are cleared, and will not be emitted on the next interval, unless another matching metric enters the pipeline.

## Configuration
| Field          | Type     | Default                | Description                                                                                              |
|----------------|----------|------------------------|----------------------------------------------------------------------------------------------------------|
| `interval`     | duration | `1m`                   | The interval on which to emit aggregate metrics.                                                         |
| `include`      | regexp   | `".*"`                 | A regex that specifies which metrics to consider for aggregation. The default regex matches all metrics. |
| `aggregations` | []string | `["min", "max, "avg"]` | A list of aggregations to perform on each. Valid values are: `min`, `max`, `avg`, `first`, `last`.       |

### Example configuration


#### Reduce volume of log-based metrics

In this example, the throughput of log-based metrics is limited, by aggregating them using the "last" aggregation. The last datapoint received from the log will be emitted every minute at a maximum.

```yaml
receivers:
  filelog:
    include:
    - $HOME/example.log
    operators:
    - type: regex_parser
      regex: "^(?P<timestamp>[^ ]+) (?P<number>.*)$$"
      timestamp:
      parse_from: attributes.timestamp
      layout: "%d-%m-%YT%H:%M:%S.%LZ"

  route/extract:

processors:
  aggregation:
    interval: 1m
    include: '^.*$$'
    aggregations: ["last"]
  metricextract:
    route: extract
    extract: attributes.number
    metric_name: 'log.count'
    metric_unit: '{count}'
    metric_type: gauge_int

exporters:
  nop:
  googlecloud:

service:
  pipelines:
    logs:
      receivers: [filelog]
      processors: [metricextract]
      exporters: [nop]
    metrics:
      receivers: [route/extract]
      processors: [aggregation]
      exporters: [googlecloud]
```

This configuration extracts metrics from a log file, and passes them through the aggregation processor. The aggregation processor will hold the last data point it receives, then emit it after a one minute interval as `log.count.last`, sending the metric to Google Cloud Monitoring. This limits the throughput to 1 metric per minute.

#### Sample CPU utilization at a higher rate

In this example, we sample CPU utilization once per second, but only emit aggregate metrics every minute. This allows for a higher effective sample rate of the CPU utilization.

```yaml
receivers:
  hostmetrics:
    collection_interval: 1s
    scrapers:
      cpu:
        metrics:
          system.cpu.time:
            enabled: false
          system.cpu.utilization:
            enabled: true

processors:
  aggregation:
    interval: 1m
    include: '^.*$$'
    aggregations: ["avg", "min", "max"]

exporters:
  googlecloud:


service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [aggregation]
      exporters: [googlecloud]
```

This configuration will emit a "system.cpu.utilization.max", "system.cpu.utilization.avg", "system.cpu.utilization.min" metric every minute, and sends them to Google Cloud Monitoring.
