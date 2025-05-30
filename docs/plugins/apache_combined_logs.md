# Apache Combined Plugin

Log parser for Apache combined format

## Configuration Parameters

| Name            | Description                                                                                                                                                          | Type     | Default                          | Required | Values               |
| :-------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------- | :------------------------------- | :------- | :------------------- |
| file_path       | Paths to Apache combined formatted log files                                                                                                                         | []string | `[/var/log/apache_combined.log]` | false    |                      |
| start_at        | At startup, where to start reading logs from the file (`beginning` or `end`)                                                                                         | string   | `end`                            | false    | `beginning`, `end`   |
| retain_raw_logs | When enabled will preserve the original log message in a `raw_log` key. This will either be in the `body` or `attributes` depending on how `parse_to` is configured. | bool     | `false`                          | false    |                      |
| parse_to        | Where to parse structured log parts                                                                                                                                  | string   | `body`                           | false    | `body`, `attributes` |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/apache_combined_logs.yaml
    parameters:
      file_path: [/var/log/apache_combined.log]
      start_at: end
      retain_raw_logs: false
      parse_to: body
      save_log_record_original: false
```
