# Apache Kafka Plugin

Log parser for Apache Kafka

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_server_log | Enable to collect Apache Kafka server logs | bool | `true` | false |  |
| server_log_path | Apache Kafka server log path | []string | `[/home/kafka/kafka/logs/server.log*]` | false |  |
| enable_controller_log | Enable to collect Apache Kafka controller logs | bool | `true` | false |  |
| controller_log_path | Apache Kafka controller log path | []string | `[/home/kafka/kafka/logs/controller.log*]` | false |  |
| enable_state_change_log | Enable to collect Apache Kafka state change logs | bool | `true` | false |  |
| state_change_log_path | Apache Kafka state-change log path | []string | `[/home/kafka/kafka/logs/state-change.log*]` | false |  |
| enable_log_cleaner_log | Enable to collect Apache Kafka log cleaner logs | bool | `true` | false |  |
| log_cleaner_log_path | Apache Kafka log-cleaner log path | []string | `[/home/kafka/kafka/logs/log-cleaner.log*]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/kafka_logs.yaml
    parameters:
      enable_server_log: true
      server_log_path: [/home/kafka/kafka/logs/server.log*]
      enable_controller_log: true
      controller_log_path: [/home/kafka/kafka/logs/controller.log*]
      enable_state_change_log: true
      state_change_log_path: [/home/kafka/kafka/logs/state-change.log*]
      enable_log_cleaner_log: true
      log_cleaner_log_path: [/home/kafka/kafka/logs/log-cleaner.log*]
      start_at: end
      timezone: UTC
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
