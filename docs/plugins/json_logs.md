# JSON Plugin

Log parser for JSON

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| log_paths | Specify a single path or multiple paths to read one or many files. You may also use a wildcard (*) to read multiple files within a directory | []string |  | true |  |
| exclude_log_paths | Specify a single path or multiple paths to exclude one or many files from being read. You may also use a wildcard (*) to exclude multiple files from being read within a directory | []string | `[]` | false |  |
| encoding | Specify the encoding of the file(s) being read. In most cases, you can leave the default option selected. | string | `utf-8` | false | `nop`, `utf-8`, `utf-16le`, `utf-16be` |
| log_type | Adds the specified 'Type' as a label to each log message | string | `json` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| parse | When enabled, parses the JSON fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/json_logs.yaml
    parameters:
      log_paths: [$LOG_PATHS]
      exclude_log_paths: []
      encoding: utf-8
      log_type: json
      start_at: end
      parse: true
```
