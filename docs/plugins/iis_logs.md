# IIS Plugin

Log parser for IIS

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| file_path | Specify a single path or multiple paths to read one or many files. You may also use a wildcard (*) to read multiple files within a directory | []string | `[C:/inetpub/logs/LogFiles/W3SVC*/**/*.log]` | false |  |
| log_format | The format of the IIS logs. For more information on the various log formats, see: https://docs.microsoft.com/en-us/previous-versions/iis/6.0-sdk/ms525807%28v=vs.90%29 | string | `w3c` | false | `w3c`, `iis`, `ncsa` |
| w3c_header | The W3C header which specifies the field names. Only applicable if log_format is w3c. Fields are automatically detected if unspecified. 'start_at' must be beginning if unspecified. | string |  | false |  |
| exclude_file_log_path | Specify a single path or multiple paths to exclude one or many files from being read. You may also use a wildcard (*) to exclude multiple files from being read within a directory | []string | `[]` | false |  |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| include_file_name | Enable to include file name in logs | bool | `true` | false |  |
| include_file_path | Enable to include file path in logs | bool | `true` | false |  |
| include_file_name_resolved | Enable to include file name resolved in logs | bool | `false` | false |  |
| include_file_path_resolved | Enable to include file path resolved in logs | bool | `false` | false |  |
| max_concurrent_files | Max number of W3C files that will be open during a batch | int | `1024` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`). | string | `beginning` | false | `beginning`, `end` |
| retain_raw_logs | When enabled will preserve the original log message in a `raw_log` key. This will either be in the `body` or `attributes` depending on how `parse_to` is configured. | bool | `false` | false |  |
| parse_to | Where to parse structured log parts | string | `body` | false | `body`, `attributes` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/iis_logs.yaml
    parameters:
      file_path: [C:/inetpub/logs/LogFiles/W3SVC*/**/*.log]
      log_format: w3c
      exclude_file_log_path: []
      timezone: UTC
      include_file_name: true
      include_file_path: true
      include_file_name_resolved: false
      include_file_path_resolved: false
      max_concurrent_files: 1024
      start_at: beginning
      retain_raw_logs: false
      parse_to: body
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
      parse: true
```
