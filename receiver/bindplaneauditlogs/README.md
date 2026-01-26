# Bindplane Audit Logs Receiver

This receiver is capable of collecting audit logs from a Bindplane instance.

## Minimum Agent Versions

- Introduced: [v1.73.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.73.0)

## Supported Pipelines

- Logs

## How It Works

1. The user configures this receiver in a pipeline.
2. The user connects to the receiver via API key. This API key has access to the audit logs of a single project.
3. The receiver connects to the Bindplane API using the provided endpoint and API key.
4. The receiver polls Bindplane for audit logs once per `poll_interval`.
5. The receiver converts the audit logs to OpenTelemetry logs and sends them to the collector.

## Prerequisites

- A Bindplane instance to collect audit logs from.
- A Bindplane API key with read access to audit logs.

## Configuration

| Field            | Type   | Default | Required | Description                                                                                                                                                                                                                  |
| ---------------- | ------ | ------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| api_key          | string |         | `true`   | The Bindplane API key with read access to audit logs. This API key has access to the audit logs of a single project.                                                                                                        |
| endpoint         | string |         | `true`   | The endpoint to collect logs from. (e.g. `https://app.bindplane.com`)                                                                                                                                                       |
| poll_interval    | string | 10s     | `false`  | The rate at which this receiver will poll Bindplane for logs. This value must be in the range [10 seconds - 24 hours] and must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). |
| parse_attributes | bool   | true    | `false`  | When true, parses audit log fields into log record attributes and sets the body to the description. When false, sets the body to the raw JSON event without attributes.                                                     |

### Example Configuration

```yaml
receivers:
  bindplaneauditlogs:
    api_key: 1234567890
    endpoint: https://app.bindplane.com
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
