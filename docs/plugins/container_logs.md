# Kubernetes Container Logs Plugin

Log parser for Kubernetes Container logs. This plugin is meant to be used with the OpenTelemetry Operator for Kubernetes (https://github.com/open-telemetry/opentelemetry-operator).

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| log_source | Where to read container logs from | string | `file` | false | `file`, `journald` |
| log_paths | A list of file glob patterns that match the file paths to be read | []string | `[/var/log/containers/*.log]` | false |  |
| journald_path | The path to read journald container logs from | string | `/var/log/journal` | false |  |
| exclude_file_log_path | A list of file glob patterns to exclude from reading | []string | `[/var/log/containers/observiq-*-collector-* /var/log/containers/bindplane-*-agent-*]` | false |  |
| body_json_parsing | If the application log is detected as json, parse the values into the log entry's body. | bool | `true` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| log_driver | The container runtime's log driver used to write container logs to disk. Valid options include `auto`, `docker-json-file` and `containerd-cri`. When set to `auto`, the format will be detected using regex. Format detection is convenient but comes with the cost of performing a regex match against every log entry read by the filelog receiver. | string | `auto` | false | `auto`, `docker-json-file`, `containerd-cri` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/container_logs.yaml
    parameters:
      log_source: file
      log_paths: [/var/log/containers/*.log]
      journald_path: /var/log/journal
      exclude_file_log_path: [/var/log/containers/observiq-*-collector-* /var/log/containers/bindplane-*-agent-*]
      body_json_parsing: true
      start_at: end
      log_driver: auto
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      parse: true
```
