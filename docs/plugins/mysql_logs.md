# MySQL Plugin

Log parser for MySQL.
This plugin supports error logging as well as query logging.

To enable general query logging run the following with an admin user:
SET GLOBAL general_log_file = '/var/log/mysql/general.log';
SET GLOBAL general_log = 'ON';

To enable slow query logging run the following with an admin user:
SET GLOBAL slow_query_log_file = '/var/log/mysql/slow-query.log';
SET GLOBAL slow_query_log = 'ON';

## Configuration Parameters

| Name                     | Description                                                                  | Type     | Default                                                                    | Required | Values             |
| :----------------------- | :--------------------------------------------------------------------------- | :------- | :------------------------------------------------------------------------- | :------- | :----------------- |
| enable_general_log       | Enable to collect MySQL general logs                                         | bool     | `true`                                                                     | false    |                    |
| general_log_paths        | Path to general log file                                                     | []string | `[/var/log/mysql/general.log]`                                             | false    |                    |
| enable_slow_log          | Enable to collect MySQL slow query logs                                      | bool     | `true`                                                                     | false    |                    |
| slow_query_log_paths     | Path to slow query log file                                                  | []string | `[/var/log/mysql/slow*.log]`                                               | false    |                    |
| enable_error_log         | Enable to collect MySQL error logs                                           | bool     | `true`                                                                     | false    |                    |
| error_log_paths          | Path to mysqld log file                                                      | []string | `[/var/log/mysqld.log /var/log/mysql/mysqld.log /var/log/mysql/error.log]` | false    |                    |
| enable_mariadb_audit_log | Enable to collect MySQL audit logs provided by MariaDB Audit plugin          | bool     | `false`                                                                    | false    |                    |
| mariadb_audit_log_paths  | Path to audit log file created by MariaDB plugin                             | []string | `[/var/log/mysql/audit.log]`                                               | false    |                    |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`) | string   | `end`                                                                      | false    | `beginning`, `end` |
| offset_storage_dir       | The directory that the offset storage file will be created                   | string   | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage`                                   | false    |                    |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.  | string   | `false`                                                                    | false    |                    |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/mysql_logs.yaml
    parameters:
      enable_general_log: true
      general_log_paths: [/var/log/mysql/general.log]
      enable_slow_log: true
      slow_query_log_paths: [/var/log/mysql/slow*.log]
      enable_error_log: true
      error_log_paths:
        [/var/log/mysqld.log /var/log/mysql/mysqld.log /var/log/mysql/error.log]
      enable_mariadb_audit_log: false
      mariadb_audit_log_paths: [/var/log/mysql/audit.log]
      start_at: end
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
