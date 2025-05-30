# Common Event Format Plugin

File Input Common Event Format Parser

## Configuration Parameters

| Name                     | Description                                                                                                                                                                         | Type     | Default | Required | Values             |
| :----------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------- | :------ | :------- | :----------------- |
| file_log_path            | Specify a single path or multiple paths to read one or many files. You may also use a wildcard (\*) to read multiple files within a directory.                                      | []string |         | true     |                    |
| exclude_file_log_path    | Specify a single path or multiple paths to exclude one or many files from being read. You may also use a wildcard (\*) to exclude multiple files from being read within a directory | []string | `[]`    | false    |                    |
| log_type                 | Adds the specified 'Type' as a label to each log message.                                                                                                                           | string   | `cef`   | false    |                    |
| timezone                 | Timezone to use when parsing the timestamp                                                                                                                                          | timezone | `UTC`   | false    |                    |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`)                                                                                                        | string   | `end`   | false    | `beginning`, `end` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.                                                                                                         | string   | `false` | false    |                    |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/common_event_format_logs.yaml
    parameters:
      file_log_path: [$FILE_LOG_PATH]
      exclude_file_log_path: []
      log_type: cef
      timezone: UTC
      start_at: end
      save_log_record_original: false
```
