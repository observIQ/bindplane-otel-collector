# Logs Receiver

Allows configuration of a raw pipeline, using components from [opentelemetry-log-collection](https://github.com/open-telemetry/opentelemetry-log-collection).

Supported pipeline types: logs

> :construction: This receiver is in alpha and configuration fields are subject to change.

## Configuration

| Field        | Default | Description                                                                                                        |
| ---          | ---     | ---                                                                                                                |
| `pipeline`   | []      | An array of [operators](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/README.md#what-operators-are-available). See below for more details. |
| `plugin_dir` | ""      | A path to a directory that contains [plugins](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/plugins.md#defining-plugins). |

### Pipeline

A pipeline is made up of `operators`. The last operator in a pipeline will automatically emit logs from this receiver. Each operator performs a simple responsibility, such as parsing a timestamp or JSON. Chain together operators to process logs into a desired format.

- Every operator has a `type`.
- Every operator can be given a unique `id`. If you use the same type of operator more than once in a pipeline, you must specify an `id`. Otherwise, the `id` defaults to the value of `type`.
- Operators will output to the next operator in the pipeline. The last operator in the pipeline will emit from the receiver. Optionally, the `output` parameter can be used to specify the `id` of another operator to which logs will be passed directly.
- Only parsers and general purpose operators should be used.

### Statefulness

Some `operators` are able to persist state across subsequent executions of the collector. To enable this, simply configure a [storage extension](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/storage), such as the [filestorage](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/storage/filestorage) extension.

## Additional Terminology and Features

- An [entry](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/entry.md) is the base representation of log data as it moves through a pipeline. All operators either create, modify, or consume entries.
- A [field](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/field.md) is used to reference values in an entry.
- A common [expression](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/expression.md) syntax is used in several operators. For example, expressions can be used to [filter](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/filter.md) or [route](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/router.md) entries.
- [timestamp](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/timestamp.md) parsing is available as a block within all parser operators, and also as a standalone operator. Many common timestamp layouts are supported.
- [severity](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/severity.md) parsing is available as a block within all parser operators, and also as a standalone operator. Stanza uses a flexible severity representation which is automatically interpreted by the stanza receiver.


## Example - Tailing a simple json file

Receiver Configuration
```yaml
receivers:
  stanza:
    plugin_dir: ./local/plugins
    pipeline:    
      - type: file_input
        include: [ ./local/test/myfile.log ]
      - type: json_parser
        timestamp:
          parse_from: time
          layout: '%Y-%m-%d %H:%M:%S'
```
