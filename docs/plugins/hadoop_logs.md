# Apache Hadoop Plugin

Log parser for Apache Hadoop

## Configuration Parameters

| Name                     | Description                                                                  | Type     | Default                                               | Required | Values             |
| :----------------------- | :--------------------------------------------------------------------------- | :------- | :---------------------------------------------------- | :------- | :----------------- |
| enable_datanode_logs     | Enable collection of Hadoop data node logs                                   | bool     | `true`                                                | false    |                    |
| datanode_log_path        | The absolute path to the data node logs                                      | []string | `[/usr/local/hadoop/logs/hadoop-*-datanode-*.log]`    | false    |                    |
| enable_resourcemgr_logs  | Enable the collection of ResourceManager logs                                | bool     | `true`                                                | false    |                    |
| resourcemgr_log_path     | The absolute path to the DataNode logs                                       | []string | `[/usr/local/hadoop/logs/hadoop-*-resourcemgr-*.log]` | false    |                    |
| enable_namenode_logs     | Enable collection of Hadoop NameNode logs                                    | bool     | `true`                                                | false    |                    |
| namenode_log_path        | The absolute path to the NameNode logs                                       | []string | `[/usr/local/hadoop/logs/hadoop-*-namenode-*.log]`    | false    |                    |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`) | string   | `end`                                                 | false    | `beginning`, `end` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.  | string   | `false`                                               | false    |                    |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/hadoop_logs.yaml
    parameters:
      enable_datanode_logs: true
      datanode_log_path: [/usr/local/hadoop/logs/hadoop-*-datanode-*.log]
      enable_resourcemgr_logs: true
      resourcemgr_log_path: [/usr/local/hadoop/logs/hadoop-*-resourcemgr-*.log]
      enable_namenode_logs: true
      namenode_log_path: [/usr/local/hadoop/logs/hadoop-*-namenode-*.log]
      start_at: end
      save_log_record_original: false
```
