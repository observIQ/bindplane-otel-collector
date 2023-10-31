# Datapoint Count Processor
This processor is used to convert the number of datapoints received during an interval into a metric.

## Supported pipelines
- Metrics

## How It Works
1. The user configures the datapoint count processor in their metrics pipeline and a route receiver in their target metrics pipeline.
2. If any incoming metrics match the `ottl_match` expression, they are counted and dimensioned by their `ottl_attributes`. Regardless of match, all metrics are sent to the next component in the metrics pipeline.
3. After each configured interval, the observed metric counts are converted into gauge metrics. These metrics are sent to the configured route receiver.


## Configuration
| Field           | Type     | Default           | Description                                                                                                                                                                                                                                                               |
|-----------------|----------|-------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ottl_match      | string   | `true`            | An [OTTL] expression used to match which datapoints to count. All paths in the [datapoint context] are available to reference. All [converters] are available to use.                                                                                                     |
| match           | string   | ``                | **DEPRECATED** use `ottl_match` instead. A boolean [expression](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) used to match which datapoints to count. By default, all datapoints are counted.                                               |
| route           | string   | ` `               | The name of the [route receiver](../../receiver/routereceiver/README.md) to send metrics to.                                                                                                                                                                              |
| interval        | duration | `1m`              | The interval at which count metrics are created. The counter will reset after each interval.                                                                                                                                                                              |
| metric_name     | string   | `datapoint.count` | The name of the metric created.                                                                                                                                                                                                                                           |
| metric_unit     | string   | `{datapoints}`    | The unit of the metric created.                                                                                                                                                                                                                                           |
| ottl_attributes | map      | `{}`              | The mapped attributes of the metric created. Each key is an attribute name. Each value is an [OTTL] expression. All paths in the [datapoint context] are available to reference. All [converters] are available to use.                                                   |
| attributes      | map      | `{}`              | **DEPRECATED** use `ottl_attributes` instead. The mapped attributes of the metric created. Each key is an attribute name. Each value is an [expression](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) that extracts data from the datapoint. |

[OTTL]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.88.0/pkg/ottl#readme
[converters]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.88.0/pkg/ottl/ottlfuncs/README.md#converters
[datapoint context]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.88.0/pkg/ottl/contexts/ottldatapoint/README.md

### Example Config
The following config is an example configuration of the log count processor using default values. In this example, host metrics are scraped, sent to the processor to be counted, and then consumed by the logging exporter. After each minute, the datapoint counts are converted to metrics and sent to the route receiver in the metrics pipeline, which then forwards to the Google Cloud exporter.
```yaml
receivers:
    hostmetrics:
        collection_interval: 30s
        scrapers:
            load:
            filesystem:
            memory:
            network:
    route/example:
processors:
    batch:
    datapointcount:
        route: example
exporters:
    googlecloud:
    logging:

service:
    pipelines:
        metrics/host:
            receivers: [hostmetrics]
            processors: [datapointcount, batch]
            exporters: [logging]
        metrics/count:
            receivers: [route/example]
            processors: [batch]
            exporters: [googlecloud]
```


## Expression Language
**DEPRECATED**
The expression language has been deprecated in favor of [OTTL]. Use the `ottl_match` and `ottl_attributes` options instead of `match` and `attributes` for OTTL based expressions.

--- 
In order to match or extract values from metrics, the following `keys` are reserved and can be used to traverse the metrics data model.

| Key               | Description                                                             |
|-------------------|-------------------------------------------------------------------------|
| `attributes`      | Used to access the attributes of the log.                               |
| `resource`        | Used to access the resource of the log.                                 |
| `metric_name`     | Used to access the metric name for the metric containing the datapoint. |
| `datapoint_value` | Used to access the value of gauge and sum datapoints.                   |

In order to access embedded values, use JSON dot notation. For example, `attributes.example.field` can be used to access a field two levels deep on the metric attributes. 

However, if a key already possesses a literal dot, users will need to use bracket notation to access that field. For example, when the field `service.name` exists on the metric's resource, users will need to use `resource["service.name"]` to access this value.

For more information about syntax and available operators, see the [Expression Language Definition](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md).

## How To
### Match specific datapoints
The following configuration adds a match expression that will only match process metrics.
```yaml
processors:
    datapointcount:
        ottl_match: IsMatch(metric.name, "^process\.")
```

### Extract metric attributes
The following configuration extracts the metric name for each datapoint. This value is used as a metric attribute. For each unique combination observed, a unique metric count is created. This configuration counts the number of observed datapoints for each metric.
```yaml
processors:
    datapointcount:
        ottl_attributes:
            metric: metric.name
```


