# PostgreSQL Plugin

Log parser for PostgreSQL. 
This plugin supports normal logging and slow query logging.
Slow query logging can be enabled via the following steps:

  1. Open postgresql.conf (found by running psql -U postgres -c 'SHOW config_file')
  2. Replace the line #log_min_duration_statement = -1 with log_min_duration_statement = 1000.
     This will log all queries executing longer than 1000ms (1s).
  3. Save the configuration and restart PostgreSQL


## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| postgresql_log_path | Path to the PostgreSQL log file | []string | `[/var/log/postgresql/postgresql*.log /var/lib/pgsql/data/log/postgresql*.log /var/lib/pgsql/*/data/log/postgresql*.log]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/postgresql_logs.yaml
    parameters:
      postgresql_log_path: [/var/log/postgresql/postgresql*.log /var/lib/pgsql/data/log/postgresql*.log /var/lib/pgsql/*/data/log/postgresql*.log]
      start_at: end
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
