# Ingress NGINX Plugin

Log parser for Ingress NGINX

## Configuration Parameters

| Name | Description | Type | Default | Required | Values |
|:-- |:-- |:-- |:-- |:-- |:-- |
| file_path | Specify a single path or multiple paths to read one or many files. You may also use a wildcard (*) to read multiple files within a directory | []string | `[/var/log/containers/ingress-nginx-controller*.log]` | false |  |
| start_at | At startup, where to start reading logs from the file (`beginning` or `end`) | string | `end` | false | `beginning`, `end` |
| cluster_name | Optional cluster name to be included in logs | string | `` | false |  |
| parse | When enabled, parses the log fields into structured attributes. When disabled, sends the raw log message in the body field. | bool | `true` | false |  |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/ingress_nginx_logs.yaml
    parameters:
      file_path: [/var/log/containers/ingress-nginx-controller*.log]
      start_at: end
      parse: true
```
