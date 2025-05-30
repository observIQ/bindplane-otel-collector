# Cisco Catalyst Plugin

Log parser for Cisco Catalyst

## Configuration Parameters

| Name                     | Description                                                                           | Type     | Default   | Required | Values |
| :----------------------- | :------------------------------------------------------------------------------------ | :------- | :-------- | :------- | :----- |
| listen_port              | A port which the agent will listen for udp messages                                   | int      | `5140`    | false    |        |
| listen_ip                | A UDP ip address                                                                      | string   | `0.0.0.0` | false    |        |
| add_attributes           | Adds net.transport, net.peer.ip, net.peer.port, net.host.ip and net.host.port labels. | bool     | `true`    | false    |        |
| timezone                 | Timezone to use when parsing the timestamp                                            | timezone | `UTC`     | false    |        |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.           | string   | `false`   | false    |        |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/cisco_catalyst_logs.yaml
    parameters:
      listen_port: 5140
      listen_ip: 0.0.0.0
      add_attributes: true
      timezone: UTC
      save_log_record_original: false
```
