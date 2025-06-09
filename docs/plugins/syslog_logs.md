# Syslog Plugin

Log receiver for Syslog

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| connection_type | Transport protocol to use (`udp` or `tcp`) | string | `udp` | false | `tcp`, `udp` |
| protocol | Protocol of received syslog messages (`rfc3164 (BSD)` or `rfc5424 (IETF)`) | string | `rfc5424` | false | `rfc3164`, `rfc5424` |
| timezone | Timezone to use when parsing the timestamp | timezone | `UTC` | false |  |
| listen_address | User's listen_ip and listen_port | string | `0.0.0.0:514` | false |  |
| enable_tls | Enable TLS for the Syslog receiver | bool | `false` | false |  |
| tls_certificate_path | Path to the x509 PEM certificate or certificate chain to use for TLS | string |  | false |  |
| tls_private_key_path | Path to the certificates x509 PEM private key to use for TLS | string |  | false |  |
| tls_ca_path | Path to the CA file to use for TLS | string |  | false |  |
| tls_min_version | Minimum version of TLS to use, client will negotiate highest possible | string | `1.2` | false | `1.0`, `1.1`, `1.2`, `1.3` |
| max_log_size | Maximum number of bytes for a single TCP message. Only applicable when connection_type is TCP | string | `1024kib` | false |  |
| data_flow | High mode keeps all entries, low mode filters out log entries with a debug severity of (7) | string | `high` | false | `high`, `low` |
| retain_raw_logs | When enabled will preserve the original log message in a `raw_log` key. This will either be in the `body` or `attributes` depending on how `parse_to` is configured. | bool | `false` | false |  |
| parse_to | Where to parse structured log parts | string | `body` | false | `body`, `attributes` |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/syslog_logs.yaml
    parameters:
      connection_type: udp
      protocol: rfc5424
      timezone: UTC
      listen_address: 0.0.0.0:514
      enable_tls: false
      tls_min_version: 1.2
      max_log_size: 1024kib
      data_flow: high
      retain_raw_logs: false
      parse_to: body
      parse: true
```
