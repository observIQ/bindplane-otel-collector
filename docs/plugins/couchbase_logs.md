# Couchbase Plugin

Log parser for Couchbase

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| enable_error_log | Enable to collect Couchbase error logs | bool | `true` | false |  |
| error_log_path | The absolute path to the Couchbase error logs | []string | `[/opt/couchbase/var/lib/couchbase/logs/error.log]` | false |  |
| enable_debug_log | Enable to collect Couchbase debug logs | bool | `false` | false |  |
| debug_log_path | The absolute path to the Couchbase debug logs | []string | `[/opt/couchbase/var/lib/couchbase/logs/debug.log]` | false |  |
| enable_info_log | Enable to collect Couchbase info logs | bool | `false` | false |  |
| info_log_path | The absolute path to the Couchbase information logs | []string | `[/opt/couchbase/var/lib/couchbase/logs/info.log]` | false |  |
| enable_access_log | Enable to collect Couchbase http access logs | bool | `true` | false |  |
| http_access_log_path | The absolute path to the Couchbase http access logs | []string | `[/opt/couchbase/var/lib/couchbase/logs/http_access.log]` | false |  |
| enable_internal_access_log | Enable to collect Couchbase internal http access logs | bool | `false` | false |  |
| http_internal_access_log_path | The absolute path to the Couchbase internal http access logs | []string | `[/opt/couchbase/var/lib/couchbase/logs/http_access_internal.log]` | false |  |
| enable_babysitter_log | Enable to collect Couchbase babysitter logs | bool | `true` | false |  |
| babysitter_log_path | The absolute path to the Couchbase babysitter logs | []string | `[/opt/couchbase/var/lib/couchbase/logs/babysitter.log]` | false |  |
| enable_xdcr_log | Enable to collect Couchbase Cross Datacenter Replication logs | bool | `false` | false |  |
| xdcr_log_path | The absolute path to the Couchbase Cross Datacenter Replication logs | []string | `[/opt/couchbase/var/lib/couchbase/logs/goxdcr.log]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/couchbase_logs.yaml
    parameters:
      enable_error_log: true
      error_log_path: [/opt/couchbase/var/lib/couchbase/logs/error.log]
      enable_debug_log: false
      debug_log_path: [/opt/couchbase/var/lib/couchbase/logs/debug.log]
      enable_info_log: false
      info_log_path: [/opt/couchbase/var/lib/couchbase/logs/info.log]
      enable_access_log: true
      http_access_log_path: [/opt/couchbase/var/lib/couchbase/logs/http_access.log]
      enable_internal_access_log: false
      http_internal_access_log_path: [/opt/couchbase/var/lib/couchbase/logs/http_access_internal.log]
      enable_babysitter_log: true
      babysitter_log_path: [/opt/couchbase/var/lib/couchbase/logs/babysitter.log]
      enable_xdcr_log: false
      xdcr_log_path: [/opt/couchbase/var/lib/couchbase/logs/goxdcr.log]
      start_at: end
      save_log_record_original: false
```
