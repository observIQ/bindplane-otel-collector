# CSV Plugin

Log parser for CSV

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| log_paths | Specify a single path or multiple paths to read one or many files. You may also use a wildcard (*) to read multiple files within a directory | []string |  | true |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| exclude_log_paths | Specify a single path or multiple paths to exclude one or many files from being read. You may also use a wildcard (*) to exclude multiple files from being read within a directory. | []string | `[]` | false |  |
| header | Comma separated header string to be used as keys | string |  | true |  |
| encoding | Specify the encoding of the file(s) being read. In most cases, you can leave the default option selected | string | `utf-8` | false | `nop`, `utf-8`, `utf-16le`, `utf-16be`, `ascii`, `big5` |
| log_type | Adds the specified 'Type' as a label to each log message | string | `csv` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/csv_logs.yaml
    parameters:
      log_paths: [$LOG_PATHS]
      start_at: end
      exclude_log_paths: []
      header: $HEADER
      encoding: utf-8
      log_type: csv
      save_log_record_original: false
```
