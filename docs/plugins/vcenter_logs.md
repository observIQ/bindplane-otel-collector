# VMware vCenter Plugin

Log parser for VMware vCenter

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| listen_port | A port which the agent will listen for vCenter messages | int | `5140` | false |  |
| listen_ip | The local IP address to listen for vCenter connections on | string | `0.0.0.0` | false |  |
| max_buffer_size | Maximum size taken up by collected logs | string | `1024kib` | false |  |
| enable_tls | Enable TLS for the vCenter listener | bool | `false` | false |  |
| certificate_file | Path to the x509 PEM certificate or certificate chain to use for TLS | string | `/opt/cert` | false |  |
| key_file | Path to the key file to use for TLS | string | `/opt/key` | false |  |
| retain_raw_logs | When enabled will preserve the original log message in a `raw_log` key. This will either be in the `body` or `attributes` depending on how `parse_to` is configured. | bool | `false` | false |  |
| parse_to | Where to parse structured log parts | string | `body` | false | `body`, `attributes` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | bool | `false` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/vcenter_logs.yaml
    parameters:
      listen_port: 5140
      listen_ip: 0.0.0.0
      max_buffer_size: 1024kib
      enable_tls: false
      certificate_file: /opt/cert
      key_file: /opt/key
      retain_raw_logs: false
      parse_to: body
      save_log_record_original: false
      parse: true
```
