# Apache HBase Plugin

Log parser for Apache HBase

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_master_log | Enable to collect HBase master logs | bool | `true` | false |  |
| master_log_path | The absolute path to the HBase master logs | []string | `[/usr/local/hbase*/logs/hbase*-master-*.log]` | false |  |
| enable_region_log | Enable to collect HBase region logs | bool | `true` | false |  |
| region_log_path | The absolute path to the HBase region logs | []string | `[/usr/local/hbase*/logs/hbase*-regionserver-*.log]` | false |  |
| enable_zookeeper_log | Enable to collect HBase zookeeper logs | bool | `false` | false |  |
| zookeeper_log_path | The absolute path to the HBase zookeeper logs | []string | `[/usr/local/hbase*/logs/hbase*-zookeeper-*.log]` | false |  |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/hbase_logs.yaml
    parameters:
      enable_master_log: true
      master_log_path: [/usr/local/hbase*/logs/hbase*-master-*.log]
      enable_region_log: true
      region_log_path: [/usr/local/hbase*/logs/hbase*-regionserver-*.log]
      enable_zookeeper_log: false
      zookeeper_log_path: [/usr/local/hbase*/logs/hbase*-zookeeper-*.log]
      timezone: UTC
      start_at: end
      save_log_record_original: false
```
