# Okta Receiver
This receiver is capable of collecting logs from an Okta domain.

## Minimum Agent Versions
- Introduced: [v1.59.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.59.0)

## Supported Pipelines
- Logs

## How It Works
1. The user configures this receiver in a pipeline.
2. The user configures a supported component to route telemetry from this receiver.

## Prerequisites
- An Okta API Token will be needed to authorize the receiver with your Okta Domain.

## Configuration
| Field                | Type      | Default          | Required | Description                                                                                                                                                                            |
|----------------------|-----------|------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| okta_domain               |  string   |                  | `true`   | The Okta domain the receiver should collect logs from (Do not include "https://"): [Find your Okta Domain](https://developer.okta.com/docs/guides/find-your-domain/main/)                                   |
| api_token            |  string   |                  | `true`   | An Okta API Token generated from the above Okta domain: [How to Create an Okta API Token](https://support.okta.com/help/s/article/How-to-create-an-API-token?language=en_US)                       |
| poll_interval        |  string   | 1m               | `false`  | The rate at which this receiver will poll Okta for logs. This value must be in the range [1 second - 24 hours] and must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration).     |

### Example Configuration
```yaml
receivers:
  okta:
    okta_domain: example.okta.com
    api_token: 11Z-XDEwgRIf4p0-RqbSFoplFh_84EOtC_ka4J7ylx
    poll_interval: 2m
exporters:
  googlecloud:
    project: my-gcp-project

service:
  pipelines:
    logs:
      receivers: [okta]
      exporters: [googlecloud]
```
