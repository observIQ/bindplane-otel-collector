# Wildfly Plugin

Log parser for Wildfly

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| standalone_file_path | Path to standalone file logs | []string | `[/opt/wildfly/standalone/log/server.log]` | false |  |
| enable_domain_server | Enable to collect domain server logs | bool | `true` | false |  |
| domain_server_path | Path to domain server logs | []string | `[/opt/wildfly/domain/servers/*/log/server.log]` | false |  |
| enable_domain_controller | Enable to collect domain controller logs | bool | `true` | false |  |
| domain_controller_path | Path to domain controller logs | []string | `[/opt/wildfly/domain/log/*.log]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| offset_storage_dir | The directory that the offset storage file will be created | string | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/wildfly_logs.yaml
    parameters:
      standalone_file_path: [/opt/wildfly/standalone/log/server.log]
      enable_domain_server: true
      domain_server_path: [/opt/wildfly/domain/servers/*/log/server.log]
      enable_domain_controller: true
      domain_controller_path: [/opt/wildfly/domain/log/*.log]
      start_at: end
      timezone: UTC
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
