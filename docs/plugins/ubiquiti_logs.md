# Ubiquiti Plugin

Log parser for Ubiquiti Unifi Devices

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| listen_port | A port which the agent will listen for syslog messages | int | `514` | false |  |
| listen_ip | The local IP address to listen for syslog connections on | string | `0.0.0.0` | false |  |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/ubiquiti_logs.yaml
    parameters:
      listen_port: 514
      listen_ip: 0.0.0.0
      timezone: UTC
      save_log_record_original: false
```
