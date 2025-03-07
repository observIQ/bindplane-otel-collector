# Bindplane Audit Logs Receiver

This receiver is capable of collecting audit logs from a Bindplane instance.

## Minimum Agent Versions

- Introduced: [v1.70.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.70.0)

## Supported Pipelines

- Logs

## How It Works

1. The user configures this receiver in a pipeline.
2. The user configures a supported component to route telemetry from this receiver.

## Prerequisites

- A Bindplane instance to collect audit logs from.
- A Bindplane API key with read access to audit logs.

## Configuration

| Field                | Type   | Default | Required | Description                                                                                                                                                                                                                  |
| -------------------- | ------ | ------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| bindplane_url_string | string |         | `true`   | The Bindplane URL the receiver should collect logs from. (e.g. `https://app.bindplane.com`)                                                                                                                                  |
| api_key              | string |         | `true`   | The Bindplane API key with read access to audit logs.                                                                                                                                                                        |
| scheme               | string |         | `false`  | The scheme of the Bindplane URL. If not provided, the scheme will be `https`.                                                                                                                                                |
| poll_interval        | string | 1m      | `false`  | The rate at which this receiver will poll Bindplane for logs. This value must be in the range [10 seconds - 24 hours] and must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). |

### Example Configuration

```yaml
receivers:
  bindplaneauditlogs:
    bindplane_url_string: https://app.bindplane.com
    api_key: 1234567890
    scheme: https
    poll_interval: 10s
exporters:
  googlecloud:
    project: my-gcp-project

service:
  pipelines:
    logs:
      receivers: [bindplaneauditlogs]
      exporters: [googlecloud]
```
