# Log Count Processor
This processor is used to convert the number of logs received during an interval into a metric.

## Supported pipelines
- Logs

## How It Works
1. The user configures the log count processor in their logs pipeline and a route receiver in their desired metrics pipeline.
2. If any incoming logs match the `match` expression, they are counted and dimensioned by their `attributes`. Regardless of match, all logs are sent to the next component in the logs pipeline.
3. After each configured interval, the observed log counts are converted into gauge metrics. These metrics are sent to the configured route receiver.


## Configuration
| Field        | Type     | Default | Description |
| ---          | ---      | ---     | ---         |
| match        | string   | `true`  | A boolean [expression](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) used to match which logs to count. By default, all logs are counted. |
| route        | string   | ` `      | The name of the [route receiver](../../receiver/routereceiver/README.md) to send metrics to. |
| interval     | duration | `1m`    | The interval at which metrics are created. The counter will reset after each interval. |
| metric_name  | string   | `log.count` | The name of the metric created. |
| metric_unit  | string   | `{logs}`    | The unit of the metric created. |
| attributes   | map      | `{}`        | The mapped attributes of the metric created. Each key is an attribute name. Each value is an [expression](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) that extracts data from the log. |

### Example Config
The following config is an example configuration of the log count processor using default values. In this example, logs are collected from a file, sent to the processor to be counted, and then consumed by the logging exporter. After each minute, the log counts are converted to metrics and sent to the route receiver in the metrics pipeline, which then forwards to the Google Cloud exporter.
```yaml
receivers:
    filelog:
        include: [./example/apache.log]
    route/example:
processors:
    batch:
    logcount:
        route: example
exporters:
    googlecloud:
    logging:

service:
    pipelines:
        logs:
            receivers: [filelog]
            processors: [batch, logcount]
            exporters: [logging]
        metrics:
            receivers: [route/example]
            processors: [batch]
            exporters: [googlecloud]
```

## Expression Language
In order to match or extract values from logs, the following `keys` are reserved and can be used to traverse the logs data model.

| Key               | Description |
| ---               | ---   |
| `body`            | Used to access the body of the log. |
| `attributes`      | Used to access the attributes of the log. |
| `resource`        | Used to access the resource of the log. |
| `severity_enum`   | Used to access the severity enum of the log. |
| `severity_number` | Used to access the severity number of the log. |

In order to access embedded values, use JSON dot notation. For example, `body.example.field` can be used to access a field two levels deep on the log body. 

However, if a key already possesses a literal dot, users will need to use bracket notation to access that field. For example, when the field `service.name` exists on the log's resource, users will need to use `resource["service.name"]` to access this value.

For more information about syntax and available operators, see the [Expression Language Definition](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md).

## How To
### Match specific logs
The following configuration adds a match expression that will count only logs with a 4xx status code. In this example, incoming logs are expected to have a `status` attribute.
```yaml
processors:
    logcount:
        match: attributes.status startsWith "4"
```

### Extract metric attributes
The following configuration extracts the status and endpoint values from the body of a log. These values are used as metric attributes. For each unique combination observed, a unique metric count is created.
```yaml
processors:
    logcount:
        attributes:
            status_code: body.status
            endpoint: body.endpoint
```
