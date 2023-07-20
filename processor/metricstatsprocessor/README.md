# Metric Stats Processor
This processor calculates statistics from metrics over a configurable interval, allowing for metrics to be sampled at a higher rate, or to reduce the volume of metric data from push-based sources.

## Minimum agent versions
- Introduced: [v1.19.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.19.0)

## Supported pipelines
- Metrics

## How it works
1. The user configures the metricstats processor in the desired metrics pipeline.
2. Every metric that flows through the pipeline is matched against the provided `include` regex.
3. If the metric name does not match the `include` regex, the metric passes through the processor.
4. If the metric matches, but is not a gauge or cumulative sum, the metric passes through the processor.
5. If the metric name does match, and the metric is a gauge or cumulative sum, the metric is added to a statistic based on its attributes. The metric does not continue down the pipeline.
6. After the configured `interval` has passed, all calculated metrics are emitted. Calculated metrics are emitted with a name of `${metric_name}.${statistic_type}` e.g. if you take the average of the metric `system.cpu.utilization`, the calculated metric would be `system.cpu.utilization.avg`.
7. All calculations are cleared, and will not be emitted on the next interval, unless another matching metric enters the pipeline.

## Configuration
| Field      | Type     | Default                | Description                                                                                               |
|------------|----------|------------------------|-----------------------------------------------------------------------------------------------------------|
| `interval` | duration | `1m`                   | The interval on which to emit calculated metrics.                                                         |
| `include`  | regexp   | `".*"`                 | A regex that specifies which metrics to consider for calculation. The default regex matches all metrics.  |
| `stats`    | []string | `["min", "max, "avg"]` | A list of statistics to calculate on each metric. Valid values are: `min`, `max`, `avg`, `first`, `last`. |

### Example configuration


#### Reduce volume of log-based metrics

In this example, the throughput of log-based metrics is limited, by calculating the "last" statistic. The last datapoint received from the log will be emitted every minute at a maximum.

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
  metricstats:
    interval: 1m
    include: '^.*$$'
    stats: ["last"]
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
      processors: [metricstats]
      exporters: [googlecloud]
```

This configuration extracts metrics from a log file, and passes them through the metricstats processor. The metricstats processor will hold the last data point it receives, then emit it after a one minute interval as `log.count.last`, sending the metric to Google Cloud Monitoring. This limits the throughput to 1 metric per minute.

#### Sample CPU utilization at a higher rate

In this example, we sample CPU utilization once per second, but only emit calculated metrics every minute. This allows for a higher effective sample rate of the CPU utilization.

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
  metricstats:
    interval: 1m
    include: '^.*$$'
    stats: ["avg", "min", "max"]

exporters:
  googlecloud:


service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [metricstats]
      exporters: [googlecloud]
```

This configuration will emit a "system.cpu.utilization.max", "system.cpu.utilization.avg", "system.cpu.utilization.min" metric every minute, and sends them to Google Cloud Monitoring.
