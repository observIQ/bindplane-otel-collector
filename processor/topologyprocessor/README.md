# Topology Processor
This processor utilizes request headers to provide extended topology functionality in BindPlane.

## Minimum agent versions
- Introduced: [v1.68.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.68.0)

## Supported pipelines:
- Logs
- Metrics
- Traces

## Configuration
| Field                | Type      | Default | Description                                                                                                                                                               |
|----------------------|-----------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `interval`           | duration  | `1m`    | The interval at which topology data is sent to Bindplane via OpAMP.                                                                                                       |
| `organizationID`     | string    |         | The Organization ID of the Bindplane configuration where this processor is running.                                                                                       |
| `accountID`          | string    |         | The Account ID of the Bindplane configuration where this processor is running.                                                                                            |
| `configuration`      | string    |         | The name of the Bindplane configuration this processor is running on.                                                                                                     |


### Example configuration

```yaml
receivers:
  filelog:
    inclucde: ["/var/log/*.log"]

processors:
  topology:
    interval: 1m
    organizationID: "myOrganizationID"
    accountID: "myAccountID"
    configuration: "myConfiguration"


exporters:
  googlecloud:

service:
  pipelines:
    logs:
      receivers:
        - filelog
      processors:
        - topology
      exporters:
        - googlecloud
```
