# Unroll Processor

This is an experimental processor that will take a log record with slice bodies and expand each element of the slice into its own log record within the slice.

## Important Note

This is an experimental processor and is expected that this functionality would eventually be moved to an [OTTL function](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl).

## Supported pipelines

- Logs


## How it works

1. The user configures the `unroll` processor in their desired logs pipeline
2. Logs that go into this pipeline with a pcommon.Slice body will have each element of that body be expanded into its own log record


## Configuration
| Field     | Type   | Default | Description                                                                                                |
| --------- | ------ | ------- | ---------------------------------------------------------------------------------------------------------- |
| field     | string | body    | note: body is currently the only available value for unrolling; making this configuration currently static |
| recursive | bool   | false   | whether to recursively unroll body slices of slices                                                        |


### Example configuration

```yaml
unroll:
    recursive: false
```



## How To

### Split a log record into multiple via a delimiter: ","

The following configuration utilizes the [transformprocessor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor) to first split the original string body and then the unroll processor can create multiple events

```yaml
receivers:
  filelog:
    include: [ ./test.txt ]
    start_at: beginning
processors:
  transform:
    log_statements:
      - context: log
        statements:
          - set(body, Split(body, ","))
  unroll:
exporters:
  file:
    path: ./test/output.json
service:
  pipelines:
    logs:
      receivers: [filelog]
      processors: [transform, unroll]
      exporters: [file]
```

<details>
<summary> Sample Data </summary>

```txt
1,2,3
```

```json
{
  "resourceLogs": [
    {
      "resource": {},
      "scopeLogs": [
        {
          "scope": {},
          "logRecords": [
            {
              "observedTimeUnixNano": "1733240156591852000",
              "body": { "stringValue": "1" },
              "attributes": [
                {
                  "key": "log.file.name",
                  "value": { "stringValue": "test.txt" }
                },
              ],
              "traceId": "",
              "spanId": ""
            },
            {
              "observedTimeUnixNano": "1733240156591852000",
              "body": { "stringValue": "2" },
              "attributes": [
                {
                  "key": "log.file.name",
                  "value": { "stringValue": "test.txt" }
                },
              ],
              "traceId": "",
              "spanId": ""
            },
            {
              "observedTimeUnixNano": "1733240156591852000",
              "body": { "stringValue": "3" },
              "attributes": [
                {
                  "key": "log.file.name",
                  "value": { "stringValue": "test.txt" }
                },
              ],
              "traceId": "",
              "spanId": ""
            }
          ]
        }
      ]
    }
  ]
}
```
</details>
