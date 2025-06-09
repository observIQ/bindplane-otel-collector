# Elasticsearch Plugin

Log parser for Elasticsearch

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_json_logs | Enable to collect Elasticsearch json logs | bool | `true` | false |  |
| json_log_paths | The absolute path to the Elasticsearch json logs | []string | `[/var/log/elasticsearch/*_server.json /var/log/elasticsearch/*_deprecation.json /var/log/elasticsearch/*_index_search_slowlog.json /var/log/elasticsearch/*_index_indexing_slowlog.json /var/log/elasticsearch/*_audit.json]` | false |  |
| enable_gc_logs | Enable to collect Elasticsearch garbage collection logs | bool | `false` | false |  |
| gc_log_paths | The absolute path to the Elasticsearch garbage collection logs | []string | `[/var/log/elasticsearch/gc.log*]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/elasticsearch_logs.yaml
    parameters:
      enable_json_logs: true
      json_log_paths: [/var/log/elasticsearch/*_server.json /var/log/elasticsearch/*_deprecation.json /var/log/elasticsearch/*_index_search_slowlog.json /var/log/elasticsearch/*_index_indexing_slowlog.json /var/log/elasticsearch/*_audit.json]
      enable_gc_logs: false
      gc_log_paths: [/var/log/elasticsearch/gc.log*]
      start_at: end
      save_log_record_original: false
      parse: true
```
