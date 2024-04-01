# Span Count Processor
This processor is used to convert the number of spans received during an interval into a metric.

## Supported pipelines
- Traces

## How It Works
1. The user configures the span count processor in their traces pipeline and a route receiver in their target metrics pipeline.
2. If any incoming spans match the `ottl_match` expression, they are counted and dimensioned by their `ottl_attributes`. Regardless of match, all spans are sent to the next component in the traces pipeline.
3. After each configured interval, the observed metric counts are converted into gauge metrics. These metrics are sent to the configured route receiver.


## Configuration
| Field           | Type     | Default      | Description                                                                                                                                                                                                                                                          |
|-----------------|----------|--------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ottl_match      | string   | `true`       | An [OTTL] expression used to match which datapoints to count. All paths in the [span context] are available to reference. All [converters] are available to use.                                                                                                     |
| match           | string   | ``           | **DEPRECATED** use `ottl_match` instead. A boolean [expression](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) used to match which spans to count. By default, all spans are counted.                                                    |
| route           | string   | ` `          | The name of the [route receiver](../../receiver/routereceiver/README.md) to send metrics to.                                                                                                                                                                         |
| interval        | duration | `1m`         | The interval at which count metrics are created. The counter will reset after each interval.                                                                                                                                                                         |
| metric_name     | string   | `span.count` | The name of the metric created.                                                                                                                                                                                                                                      |
| metric_unit     | string   | `{spans}`    | The unit of the metric created.                                                                                                                                                                                                                                      |
| ottl_attributes | map      | `{}`         | The mapped attributes of the metric created. Each key is an attribute name. Each value is an [OTTL] expression. All paths in the [span context] are available to reference. All [converters] are available to use.                                                   |
| attributes      | map      | `{}`         | **DEPRECATED** use `ottl_attributes` instead. The mapped attributes of the metric created. Each key is an attribute name. Each value is an [expression](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) that extracts data from the span. |

[OTTL]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.97.0/pkg/ottl#readme
[converters]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.97.0/pkg/ottl/ottlfuncs/README.md#converters
[span context]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.97.0/pkg/ottl/contexts/ottlspan/README.md

### Example Config
The following config is an example configuration of the span count processor using default values. In this example, spans are collected from a file, sent to the processor to be counted, and then consumed by the logging exporter. After each minute, the span counts are converted to metrics and sent to the route receiver in the metrics pipeline, which then forwards to the Google Cloud exporter.
```yaml
receivers:
    otlp:
    route/example:
processors:
    batch:
    spancount:
        route: example
exporters:
    googlecloud:
    logging:

service:
    pipelines:
        traces:
            receivers: [otlp]
            processors: [spancount, batch]
            exporters: [logging]
        metrics:
            receivers: [route/example]
            processors: [batch]
            exporters: [googlecloud]
```

## Expression Language
**DEPRECATED**
The expression language has been deprecated in favor of [OTTL]. Use the `ottl_match` and `ottl_attributes` options instead of `match` and `attributes` for OTTL based expressions.

---
In order to match or extract values from spans, the following `keys` are reserved and can be used to traverse the spans data model.

| Key                    | Description                                                                                                                       |
|------------------------|-----------------------------------------------------------------------------------------------------------------------------------|
| `attributes`           | Used to access the attributes of the span.                                                                                        |
| `resource`             | Used to access the resource of the span.                                                                                          |
| `trace_status_message` | Used to access the status message of the span.                                                                                    |
| `trace_status_code`    | Used to access the status code enum of the span. Values may be "ok", "error", or "unset".                                         |
| `trace_kind`           | Used to access the kind enum of the span. Values may be "unspecified", "internal", "client", "server", "consumer", or "producer". |
| `span_duration_ms`     | Used to access the duration of the span, in milliseconds.                                                                         |
In order to access embedded values, use JSON dot notation. For example, `attributes.example.field` can be used to access a field two levels deep on the span attributes. 

However, if a key already possesses a literal dot, users will need to use bracket notation to access that field. For example, when the field `service.name` exists on the log's resource, users will need to use `resource["service.name"]` to access this value.

For more information about syntax and available operators, see the [Expression Language Definition](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md).

## How To
### Match specific spans
The following configuration adds a match expression that will count only spans with longer than one second.
```yaml
processors:
    spancount:
        ottl_match: end_time_unix_nano - start_time_unix_nano > 1000000000
```

### Extract metric attributes
The following configuration extracts the status code and kind values from traces. These values are used as metric attributes. For each unique combination observed, a unique metric count is created.
```yaml
processors:
    spancount:
        ottl_attributes:
            status_code: status.code
            kind: kind
```
