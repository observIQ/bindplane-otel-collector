# Active Directory Logs Plugin

Log parser for Active Directory

## Configuration Parameters

| Name                     | Description                                                                  | Type   | Default | Required | Values             |
| :----------------------- | :--------------------------------------------------------------------------- | :----- | :------ | :------- | :----------------- |
| enable_dns_server        | Enable to collect DNS server logs                                            | bool   | `true`  | false    |                    |
| enable_dfs_replication   | Enable to collect DFS replication logs                                       | bool   | `true`  | false    |                    |
| enable_file_replication  | Enable to collect file replication logs                                      | bool   | `false` | false    |                    |
| poll_interval            | Set the rate that logs are being collected                                   | string | `1s`    | false    |                    |
| max_reads                | Maximum number of logs collected                                             | int    | `1000`  | false    |                    |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end`   | false    | `beginning`, `end` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.  | string | `false` | false    |                    |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/active_directory_logs.yaml
    parameters:
      enable_dns_server: true
      enable_dfs_replication: true
      enable_file_replication: false
      poll_interval: 1s
      max_reads: 1000
      start_at: end
      save_log_record_original: false
```
