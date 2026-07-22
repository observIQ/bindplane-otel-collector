# Solr Plugin

Log parser for Solr

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| file_log_path | The absolute path to the Solr logs | []string | `[/var/solr/logs/solr.log]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:BINDPLANE_COLLECTOR_STORAGE}` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/solr_logs.yaml
    parameters:
      file_log_path: [/var/solr/logs/solr.log]
      start_at: end
      offset_storage_dir: ${env:BINDPLANE_COLLECTOR_STORAGE}
      save_log_record_original: false
      parse: true
```
