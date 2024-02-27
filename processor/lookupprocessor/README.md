# Lookup Processor
This processor is used to lookup values in a csv file.

## Supported pipelines
- Logs
- Metrics
- Traces

## How It Works
1. This processor will load and periodically refresh a CSV file in memory.
2. When telemetry is received, the processor checks if the configured `field` exists in the configured `context`.
3. If the field exists and the CSV contains a matching value, all other values in that record are added to the `context` of the telemetry. The header of each value is used when adding these fields.

## Configuration
| Field        | Type     | Default | Description |
| ---          | ---      | ---     | ---         |
| csv          | string   | ` `     | The location of the CSV file used for lookups. The processor will periodically reload this in memory every minute. |
| context      | string   | ` `     | The context of the telemetry to check and use when performing lookups. Supported values are `attributes`, `body`, `resource`. |
| field        | string   | ` `     | The field to match when performing a lookup. For a lookup to succeed, the field name and value must be the same in the CSV file and telemetry. |

### Example Config
The following is an example configuration of the lookup processor. In this example, this processor will check if incoming logs contain an `ip` field on their body. If they do, the processor will use the value of `ip` to lookup additional fields in the `example.csv` file. If a match is found, all other defined values in the csv will be added to the body of the log.
```yaml
receivers:
    otlp:
        protocols:
            grpc:
processors:
    lookup:
        csv: ./example.csv
        context: body
        field: ip
exporters:
    logging:
service:
    pipelines:
        logs:
            receivers: [otlp]
            processors: [lookup]
            exporters: [logging]
```
The following is an example CSV file that might be used in this scenario. In this example, if a log contains an `ip` field on its body with a value of `0.0.0.0`, the corresponding values for `host`, `region`, and `env` will also be added to the body of the log.
```csv
ip,host,region,env
0.0.0.0,host-1,us-west,prod
1.1.1.1,host-2,us-east,dev
```