# CockroachDB Plugin

Log parser for CockroachDB

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_health_log | Enable to collect health logs | bool | `true` | false |  |
| health_log_path | The absolute path to the CockroachDB health logs | []string | `[/var/log/cockroach-data/logs/cockroach-health.log]` | false |  |
| enable_dev_log | Enable to collect general Dev logs. | bool | `true` | false |  |
| dev_log_path | The absolute path to the CockroachDB Dev Logs | []string | `[/var/log/cockroach-data/logs/cockroach.log]` | false |  |
| enable_error_log | Enable to collect stderr logs. | bool | `true` | false |  |
| error_log_path | The absolute path to the CockroachDB stderr logs | []string | `[/var/log/cockroach-data/logs/cockroach-stderr.log]` | false |  |
| enable_sql_schema_log | Enable to collect sql schema logs. | bool | `true` | false |  |
| sql_schema_log_path | The absolute path to the CockroachDB sql schema logs | []string | `[/var/log/cockroach-data/logs/cockroach-sql-schema.log]` | false |  |
| enable_telemetry_log | Enable to collect telemetry logs. | bool | `true` | false |  |
| telemetry_log_path | The absolute path to the CockroachDB telemetry logs | []string | `[/var/log/cockroach-data/logs/cockroach-telemetry.log]` | false |  |
| enable_kv_distribution_log | Enable to collect kv distribution logs. | bool | `true` | false |  |
| kv_distribution_log_path | The absolute path to the CockroachDB kv distribution logs | []string | `[/var/log/cockroach-data/logs/cockroach-kv-distribution.log]` | false |  |
| enable_pebble_log | Enable to collect cockroachdb pebble logs. | bool | `true` | false |  |
| pebble_log_path | The absolute path to the CockroachDB pebble logs | []string | `[/var/log/cockroach-data/logs/cockroach-pebble.log]` | false |  |
| start_at | At startup, where to start reading logs from the file ('beginning' or 'end') | string | `end` | false | `beginning`, `end` |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| retain_raw_logs | When enabled will preserve the original log message in a `raw_log` key. This will either be in the `body` or `attributes` depending on how `parse_to` is configured. | bool | `false` | false |  |
| parse_to | Where to parse structured log parts | string | `body` | false | `body`, `attributes` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/cockroachdb_logs.yaml
    parameters:
      enable_health_log: true
      health_log_path: [/var/log/cockroach-data/logs/cockroach-health.log]
      enable_dev_log: true
      dev_log_path: [/var/log/cockroach-data/logs/cockroach.log]
      enable_error_log: true
      error_log_path: [/var/log/cockroach-data/logs/cockroach-stderr.log]
      enable_sql_schema_log: true
      sql_schema_log_path: [/var/log/cockroach-data/logs/cockroach-sql-schema.log]
      enable_telemetry_log: true
      telemetry_log_path: [/var/log/cockroach-data/logs/cockroach-telemetry.log]
      enable_kv_distribution_log: true
      kv_distribution_log_path: [/var/log/cockroach-data/logs/cockroach-kv-distribution.log]
      enable_pebble_log: true
      pebble_log_path: [/var/log/cockroach-data/logs/cockroach-pebble.log]
      start_at: end
      timezone: UTC
      save_log_record_original: false
      retain_raw_logs: false
      parse_to: body
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      parse: true
```
