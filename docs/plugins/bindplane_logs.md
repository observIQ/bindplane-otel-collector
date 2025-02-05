# Bindplane OP Logs Plugin

Log parser for Bindplane Observability Pipeline.

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| log_paths | A list of file glob patterns that match the file paths to be read | []string | `[/var/log/bindplane/bindplane.log]` | false |  |
| exclude_file_log_path | A list of file glob patterns to exclude from reading | []string | `[/var/log/bindplane/bindplane-*.log.gz]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/bindplane_logs.yaml
    parameters:
      log_paths: [/var/log/bindplane/bindplane.log]
      exclude_file_log_path: [/var/log/bindplane/bindplane-*.log.gz]
      start_at: end
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
```
