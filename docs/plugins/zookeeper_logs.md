# Apache Zookeeper Plugin

Log parser for Apache Zookeeper

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| file_path | The absolute path to the Zookeeper logs | []string | `[/home/kafka/kafka/logs/zookeeper.log]` | false |  |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/zookeeper_logs.yaml
    parameters:
      file_path: [/home/kafka/kafka/logs/zookeeper.log]
      timezone: UTC
      start_at: end
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
      parse: true
```
