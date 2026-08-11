# RabbitMQ Plugin

Log parser for RabbitMQ

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| daemon_log_paths | The absolute path to the RabbitMQ Daemon logs | []string | `[/var/log/rabbitmq/rabbit@*.log]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:BINDPLANE_COLLECTOR_STORAGE}` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/rabbitmq_logs.yaml
    parameters:
      daemon_log_paths: [/var/log/rabbitmq/rabbit@*.log]
      start_at: end
      offset_storage_dir: ${env:BINDPLANE_COLLECTOR_STORAGE}
      save_log_record_original: false
      parse: true
```
