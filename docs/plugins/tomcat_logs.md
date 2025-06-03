# Apache Tomcat Plugin

Log parser for Apache Tomcat

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_access_log | Enable to collect Apache Tomcat access logs | bool | `true` | false |  |
| access_log_path | Path to access log file | []string | `[/usr/local/tomcat/logs/localhost_access_log.*.txt]` | false |  |
| access_retain_raw_logs | When enabled will preserve the original log message in a `raw_log` key. This will either be in the `body` or `attributes` depending on how `parse_to` is configured. | bool | `false` | false |  |
| enable_catalina_log | Enable to collect Apache Tomcat catalina logs | bool | `true` | false |  |
| catalina_log_path | Path to catalina log file | []string | `[/usr/local/tomcat/logs/catalina.out]` | false |  |
| catalina_retain_raw_logs | When enabled will preserve the original log message on the body in a `raw_log` key | bool | `false` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| timezone | Timezone to use when parsing the timestamp. | timezone | `UTC` | false |  |
| parse_to | Where to parse structured log parts | string | `body` | false | `body`, `attributes` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/tomcat_logs.yaml
    parameters:
      enable_access_log: true
      access_log_path: [/usr/local/tomcat/logs/localhost_access_log.*.txt]
      access_retain_raw_logs: false
      enable_catalina_log: true
      catalina_log_path: [/usr/local/tomcat/logs/catalina.out]
      catalina_retain_raw_logs: false
      start_at: end
      timezone: UTC
      parse_to: body
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
