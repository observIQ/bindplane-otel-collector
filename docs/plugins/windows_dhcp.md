# Windows DHCP Plugin

Log parser for Windows DHCP

## Configuration Parameters

| Name                     | Description                                                                 | Type     | Default                                       | Required | Values             |
| :----------------------- | :-------------------------------------------------------------------------- | :------- | :-------------------------------------------- | :------- | :----------------- |
| file_path                | <nil>                                                                       | []string | `[C:/Windows/System32/dhcp/DhcpSrvLog-*.log]` | false    |                    |
| start_at                 | <nil>                                                                       | string   | `end`                                         | false    | `beginning`, `end` |
| offset_storage_dir       | The directory that the offset storage file will be created                  | string   | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage`      | false    |                    |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key. | string   | `false`                                       | false    |                    |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/windows_dhcp.yaml
    parameters:
      file_path: [C:/Windows/System32/dhcp/DhcpSrvLog-*.log]
      start_at: end
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
