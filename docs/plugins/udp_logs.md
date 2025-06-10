# UDP Plugin

Log parser for UDP

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| listen_port | A port which the agent will listen for udp messages | int |  | true |  |
| log_type | Adds the specified 'Type' as a label to each log message | string | `udp` | false |  |
| listen_ip | The local IP address to listen for UDP connections on | string | `0.0.0.0` | false |  |
| add_attributes | Adds net.transport, net.peer.ip, net.peer.port, net.host.ip and net.host.port attributes | bool | `true` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/udp_logs.yaml
    parameters:
      listen_port: $LISTEN_PORT
      log_type: udp
      listen_ip: 0.0.0.0
      add_attributes: true
      parse: true
```
