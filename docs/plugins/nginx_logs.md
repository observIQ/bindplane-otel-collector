# NGINX Plugin

Log parser for NGINX

## Configuration Parameters

| Name                     | Description                                                                                                                                                                                                                                                                                                                                           | Type     | Default                                  | Required | Values                                                  |
| :----------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------- | :--------------------------------------- | :------- | :------------------------------------------------------ |
| log_format               | Specifies the format of the access log entries. When choosing 'default', the agent will expect the log entries to match the default nginx log format. When choosing 'observiq', the agent will expect the log entries to match an optimized JSON format that adheres to the observIQ specification. See the Nginx documentation for more information. | string   | `default`                                | false    | `default`, `observiq`                                   |
| enable_access_log        | Enable access log collection                                                                                                                                                                                                                                                                                                                          | bool     | `true`                                   | false    |                                                         |
| access_log_paths         | Path to access log file                                                                                                                                                                                                                                                                                                                               | []string | `[/var/log/nginx/access.log*]`           | false    |                                                         |
| enable_error_log         | Enable error log collection                                                                                                                                                                                                                                                                                                                           | bool     | `true`                                   | false    |                                                         |
| error_log_paths          | Path to error log file                                                                                                                                                                                                                                                                                                                                | []string | `[/var/log/nginx/error.log*]`            | false    |                                                         |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`)                                                                                                                                                                                                                                                                          | string   | `end`                                    | false    | `beginning`, `end`                                      |
| encoding                 | Specify the encoding of the file(s) being read. In most cases, you can leave the default option selected                                                                                                                                                                                                                                              | string   | `utf-8`                                  | false    | `nop`, `utf-8`, `utf-16le`, `utf-16be`, `ascii`, `big5` |
| data_flow                | High mode keeps all entries, low mode filters out based on http request status                                                                                                                                                                                                                                                                        | string   | `high`                                   | false    | `high`, `low`                                           |
| offset_storage_dir       | The directory that the offset storage file will be created                                                                                                                                                                                                                                                                                            | string   | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false    |                                                         |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.                                                                                                                                                                                                                                                                           | string   | `false`                                  | false    |                                                         |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/nginx_logs.yaml
    parameters:
      log_format: default
      enable_access_log: true
      access_log_paths: [/var/log/nginx/access.log*]
      enable_error_log: true
      error_log_paths: [/var/log/nginx/error.log*]
      start_at: end
      encoding: utf-8
      data_flow: high
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
