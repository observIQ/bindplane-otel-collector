# Cisco Meraki Plugin

Log parser for Cisco Meraki

## Configuration Parameters

| Name                     | Description                                                                 | Type   | Default   | Required | Values |
| :----------------------- | :-------------------------------------------------------------------------- | :----- | :-------- | :------- | :----- |
| listen_port              | A port which the agent will listen for syslog messages                      | int    | `5140`    | false    |        |
| listen_ip                | The local IP address to listen for syslog connections on                    | string | `0.0.0.0` | false    |        |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | string | `false`   | false    |        |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/cisco_meraki_logs.yaml
    parameters:
      listen_port: 5140
      listen_ip: 0.0.0.0
      save_log_record_original: false
```
