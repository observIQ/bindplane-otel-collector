# Apache Cassandra Plugin

Log parser for Apache Cassandra

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_system_logs | Enable to collect apache system logs | bool | `true` | false |  |
| system_log_path | Path to apache system logs | []string | `[/var/log/cassandra/system.log]` | false |  |
| enable_debug_logs | Enable to collect apache debug logs | bool | `true` | false |  |
| debug_log_path | Path to apache debug logs | []string | `[/var/log/cassandra/debug.log]` | false |  |
| enable_gc_logs | Enable to collect apache garbage collection logs | bool | `true` | false |  |
| gc_log_path | Path to apache garbage collection logs | []string | `[/var/log/cassandra/gc.log]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false |  |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/apache_cassandra_logs.yaml
    parameters:
      enable_system_logs: true
      system_log_path: [/var/log/cassandra/system.log]
      enable_debug_logs: true
      debug_log_path: [/var/log/cassandra/debug.log]
      enable_gc_logs: true
      gc_log_path: [/var/log/cassandra/gc.log]
      start_at: end
      timezone: UTC
      save_log_record_original: false
      parse: true
```
