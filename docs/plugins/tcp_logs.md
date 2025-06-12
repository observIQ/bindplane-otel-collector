# TCP Plugin

Log receiver for TCP

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| listen_port | A port which the agent will listen for tcp messages | int |  | true |  |
| listen_ip | The local IP address to listen for TCP connections on | string | `0.0.0.0` | false |  |
| log_type | Adds the specified 'Type' as a label to each log message. | string | `tcp` | false |  |
| add_attributes | Adds net.transport, net.peer.ip, net.peer.port, net.host.ip and net.host.port attributes | bool | `false` | false |  |
| enable_tls | Enable TLS for the TCP listener | bool | `false` | false |  |
| tls_certificate_path | File path for the X509 TLS certificate chain | string |  | false |  |
| tls_private_key_path | File path for the X509 TLS private key chain | string |  | false |  |
| tls_min_version | Minimum version of TLS to use | string | `1.2` | false | `1.0`, `1.1`, `1.2`, `1.3` |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/tcp_logs.yaml
    parameters:
      listen_port: $LISTEN_PORT
      listen_ip: 0.0.0.0
      log_type: tcp
      add_attributes: false
      enable_tls: false
      tls_min_version: 1.2
      parse: true
```
