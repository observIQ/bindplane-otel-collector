# Kubernetes Kubelet Plugin

Log parser for Kubelet journald logs

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| journald_directory | Directory containing journal files to read entries from | string |  | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| retain_raw_logs | When enabled will preserve the original log message in a `raw_log` key. This will either be in the `body` or `attributes` depending on how `parse_to` is configured. | bool | `false` | false |  |
| parse_to | Where to parse structured log parts | string | `body` | false | `body`, `attributes` |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/kubelet_logs.yaml
    parameters:
      start_at: end
      timezone: UTC
      retain_raw_logs: false
      parse_to: body
      parse: true
```
