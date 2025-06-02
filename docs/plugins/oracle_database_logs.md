# oracle_database Plugin

Oracle Database

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_audit_log | Enable to collect OracleDB audit logs | bool | `true` | false |  |
| audit_log_path | Path to the audit log file | []string | `[/u01/app/oracle/product/*/dbhome_1/admin/*/adump/*.aud]` | false |  |
| enable_alert_log | Enable to collect OracleDB alert logs | bool | `true` | false |  |
| alert_log_path | Path to the alert log file | []string | `[/u01/app/oracle/product/*/dbhome_1/diag/rdbms/*/*/trace/alert_*.log]` | false |  |
| enable_listener_log | Enable to collect OracleDB listener logs | bool | `true` | false |  |
| listener_log_path | Path to the listener log file | []string | `[/u01/app/oracle/product/*/dbhome_1/diag/tnslsnr/*/listener/alert/log.xml]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/oracle_database_logs.yaml
    parameters:
      enable_audit_log: true
      audit_log_path: [/u01/app/oracle/product/*/dbhome_1/admin/*/adump/*.aud]
      enable_alert_log: true
      alert_log_path: [/u01/app/oracle/product/*/dbhome_1/diag/rdbms/*/*/trace/alert_*.log]
      enable_listener_log: true
      listener_log_path: [/u01/app/oracle/product/*/dbhome_1/diag/tnslsnr/*/listener/alert/log.xml]
      start_at: end
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
